package main

import (
	"io/ioutil"
	"os"
	"os/exec"
)

func _dmg(p string, volname string, codesign string) (*os.File, error) {
	d, err := ioutil.TempDir("", "sparkle-bundle")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(d)

	if err := os.Chdir(d); err != nil {
		return nil, err
	}
	cmd := exec.Command("cp", "-R", p, d)
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	cmd = exec.Command("ln", "-s", "/Applications", "Applications")
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	dmg := d + ".dmg"

	cmd = exec.Command("hdiutil", "create", "-volname", volname, "-srcfolder", d, "-ov", "-format", "UDZO", dmg)
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	if codesign != "" {
		cmd := exec.Command("codesign", "-s", codesign, dmg)
		if err := cmd.Run(); err != nil {
			return nil, err
		}
	}

	return os.Open(dmg)
}
