package main

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func zipRecursive(p string) (*os.File, error) {
	f, err := ioutil.TempFile("", "sparkle.zip")
	if err != nil {
		return nil, err
	}

	w := zip.NewWriter(f)
	defer w.Close()

	dir := filepath.Dir(p)

	err = filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Println("walk err:", err)
			return err
		}

		if info.IsDir() {
			return nil
		}

		r, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		header := &zip.FileHeader{
			Name:   r,
			Method: zip.Deflate,
		}

		if info.Mode() != 0 {
			header.SetMode(info.Mode())
		}

		w1, err := w.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.Mode()&os.ModeSymlink == os.ModeSymlink {
			s, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if _, err := w1.Write([]byte(s)); err != nil {
				return err
			}
		} else {
			w2, err := os.Open(path)
			if err != nil {
				return err
			}
			defer w2.Close()

			if _, err := io.Copy(w1, w2); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		os.Remove(f.Name())
		return nil, err
	}

	return f, nil
}
