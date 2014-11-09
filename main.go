package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/DHowett/go-plist"
)

var (
	app     = flag.String("app", "", "path of app")
	config  = flag.String("config", "sparkle.json", "path of sparkle.json")
	setup   = flag.Bool("setup", false, "initialize config file")
	appcast = flag.String("appcast", "appcast.xml", "path of appcast.xml")
	output  = flag.String("output", "latest.zip", "path of ziped file")
)

const tmpl = `<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0" xmlns:sparkle="http://www.andymatuschak.org/xml-namespaces/sparkle"  xmlns:dc="http://purl.org/dc/elements/1.1/">
<channel>
<title>{{ .Title }}</title>
{{ if .Link }}<link>{{ .Link }}</link>{{ end }}
{{ if .Description }}<description>{{ .Description }}</description>{{ end }}
<item>
<title>{{ .ItemTitle }}</title>
<description>
<![CDATA[
{{ .ItemDescription }}]]>
</description>
<pubDate>{{ .ItemPubDate }}</pubDate>
<enclosure url="{{ .ItemEnclosureUrl }}" sparkle:version="{{ .ItemEnclosureVersion }}" sparkle:shortVersionString="{{ .ItemEnclosureShortVersionString }}" length="{{ .ItemEnclosureLength }}" type="application/octet-stream" sparkle:dsaSignature="{{ .ItemEnclosureDataSignature }}" />
{{ if .ItemMinimumSystemVersion }}<sparkle:minimumSystemVersion>{{ .ItemMinimumSystemVersion }}</sparkle:minimumSystemVersion>{{ end }}
</item>
</channel>
</rss>
`

const pubDateFormat = "Mon, 2 Jan 2006 15:04:05 -0700"

type InfoPlist struct {
	MinimunSystemVersion     string `plist:"LSMinimumSystemVersion"`
	BundleVersion            string `plist:"CFBundleVersion"`
	BundleShortVersionString string `plist:"CFBundleShortVersionString"`
	BundleName               string `plist:"CFBundleName"`
}

type Config struct {
	Title          string `json:"title"`
	Link           string `json:"link"`
	OutputBaseLink string `json:"output-base-link"`
	Description    string `json:"description"`
	PrivateKey     string `json:"private-key"`
}

type Tmpl struct {
	Title                           string
	Link                            string
	Description                     string
	ItemTitle                       string
	ItemDescription                 string
	ItemPubDate                     string
	ItemEnclosureUrl                string
	ItemEnclosureVersion            string
	ItemEnclosureShortVersionString string
	ItemEnclosureLength             int64
	ItemEnclosureDataSignature      string
	ItemMinimumSystemVersion        string
}

func main() {
	if err := _main(); err != nil {
		log.Fatal(err)
	}
}

func _main() error {
	flag.Parse()

	file, err := os.Open(filepath.Join(*app, "Contents", "Info.plist"))
	if err != nil {
		return err
	}
	var i InfoPlist
	if err := plist.NewDecoder(file).Decode(&i); err != nil {
		return err
	}

	if *setup {
		return _setup(&i)
	}

	file, err = os.Open(*config)
	if err != nil {
		return err
	}
	var c Config
	if err := json.NewDecoder(file).Decode(&c); err != nil {
		return err
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		return fmt.Errorf("EDITOR is not defined")
	}

	byt, err := edit()
	if err != nil {
		return err
	}

	if len(byt) == 0 {
		return fmt.Errorf("description seems to be empty")
	}

	ziped, err := zipRecursive(*app)
	if err != nil {
		return err
	}
	defer os.Remove(ziped.Name())

	sig, err := sign(ziped.Name(), c.PrivateKey)
	if err != nil {
		return err
	}

	stat, err := ziped.Stat()
	if err != nil {
		return err
	}

	if err := os.Rename(ziped.Name(), *output); err != nil {
		return err
	}

	a, err := ioutil.TempFile("", "sparkle-appcast")
	if err != nil {
		return err
	}

	base, err := url.Parse(c.OutputBaseLink)
	if err != nil {
		return err
	}
	r, err := url.Parse(filepath.Base(*output))
	if err != nil {
		return err
	}
	u := base.ResolveReference(r)

	// fill template
	err = template.Must(template.New("sparkle").Parse(tmpl)).Execute(a, Tmpl{
		Title:                           c.Title,
		Link:                            c.Link,
		Description:                     c.Description,
		ItemTitle:                       fmt.Sprintf("%s Version %s", i.BundleName, i.BundleShortVersionString),
		ItemDescription:                 string(byt),
		ItemPubDate:                     time.Now().Format(pubDateFormat),
		ItemMinimumSystemVersion:        i.MinimunSystemVersion,
		ItemEnclosureUrl:                u.String(),
		ItemEnclosureVersion:            i.BundleVersion,
		ItemEnclosureShortVersionString: i.BundleShortVersionString,
		ItemEnclosureLength:             stat.Size(),
		ItemEnclosureDataSignature:      sig,
	})

	if err != nil {
		os.Remove(a.Name())
		return err
	}

	if err := os.Rename(a.Name(), *appcast); err != nil {
		return err
	}

	return nil
}

func _setup(i *InfoPlist) error {
	if _, err := os.Stat(*config); err == nil {
		return fmt.Errorf("%s is exists", *config)
	}

	r := bufio.NewReader(os.Stdin)

	defaultTitle := fmt.Sprintf("%s ChangeLog", i.BundleName)
	title := prompt(fmt.Sprintf("Title:[%s]", defaultTitle), r)
	if title == "" {
		title = defaultTitle
	}

	link := prompt("Link:", r)

	outputBaseLink := prompt(`OutputBaseLink:(like "http://example.com/output/")`, r)
	if outputBaseLink == "" {
		return fmt.Errorf("OutputBaseLink is required")
	}
	if _, err := url.Parse(outputBaseLink); err != nil {
		return fmt.Errorf("cant parse url")
	}
	if outputBaseLink[len(outputBaseLink)-1:] != "/" {
		outputBaseLink += "/"
	}

	description := prompt("Description:", r)

	defaultPrivateKey := "dsa_priv.pem"
	privateKey := prompt(fmt.Sprintf("PrivateKey:[%s]", defaultPrivateKey), r)
	if privateKey == "" {
		privateKey = defaultPrivateKey
	}

	if _, err := os.Stat(privateKey); err != nil {
		switch err {
		case os.ErrNotExist:
			return fmt.Errorf("not exist private key:%s", privateKey)
		default:
			return err
		}
	}

	f, err := os.OpenFile(*config, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	c := Config{
		Title:          title,
		Link:           link,
		OutputBaseLink: outputBaseLink,
		Description:    description,
		PrivateKey:     privateKey,
	}

	//TODO: indent
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(&c); err != nil {
		return err
	}

	var out bytes.Buffer
	if err := json.Indent(&out, b.Bytes(), "", "    "); err != nil {
		return err
	}

	if _, err := io.Copy(f, &out); err != nil {
		return err
	}

	return nil
}

func prompt(left string, r *bufio.Reader) string {
	fmt.Print(left)
	s, _ := r.ReadString('\n')
	return strings.TrimSuffix(s, "\n")
}
