## INSTALL

personal script for supporting self sign sparkle release

```
go get github.com/soh335/sparkle-release
```

## USAGE

### setup

```
sparkle-release --app /path/to/build.app --setup
```

create sparkle.json for release meta data.

### release

```
sparkle-release --app /path/to/build.app --appcast /path/to/appcast.xml --output /path/to/latest.zip
```

* create zip file and move to ```output```
* create appcast.xml
    * fill enclosure tag (url, sparkle:version, sparkle:shortVersionString, length, type, sparkle:dsaSignature)
    * launch editor for writing description
