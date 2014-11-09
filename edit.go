package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

func edit() ([]byte, error) {
	f, err := ioutil.TempFile("", "sparkle")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f.Name())

	oldStat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		return nil, fmt.Errorf("EDITOR is not defined")
	}

	editorPath, err := exec.LookPath(editor)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(editorPath, f.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	f, err = os.Open(f.Name())
	if err != nil {
		return nil, err
	}

	newStat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if oldStat.ModTime().Unix() >= newStat.ModTime().Unix() {
		return nil, fmt.Errorf("seems not to be edited")
	}

	return ioutil.ReadAll(f)
}
