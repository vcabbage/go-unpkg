package extract

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func TGZ(r io.Reader, dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	gr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}

	tr := tar.NewReader(gr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fullpath := filepath.Join(dir, strings.TrimPrefix(hdr.Name, "package"))

		os.MkdirAll(filepath.Dir(fullpath), 0755)

		f, err := os.OpenFile(fullpath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, hdr.FileInfo().Mode())
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, tr); err != nil {
			f.Close()
			return err
		}
		f.Close()
	}

	return nil
}
