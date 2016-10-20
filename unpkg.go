package unpkg

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/vcabbage/unpkg/extract"
	"github.com/vcabbage/unpkg/npm"
)

func Get(name string) (*npm.Package, error) {
	p := npm.Parse(name)
	if err := npm.Resolve(p); err != nil {
		fmt.Println(err)
		return nil, err
	}

	return p, nil
}

func Download(p *npm.Package, cacheDir string) error {
	name := path.Base(p.URL)
	dir := filepath.Join(cacheDir, name)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Println(name, "not cached, downloading...")
	} else {
		fmt.Println(name, " cached.")
		return nil
	}

	resp, err := http.Get(p.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Println("bad response when downloading: ", resp.Status)
		return err
	}

	hasher := sha1.New()

	tee := io.TeeReader(resp.Body, hasher)

	if err := extract.TGZ(tee, dir); err != nil {
		fmt.Println("error extracting tgz:", err)
		return err
	}

	if hash := hex.EncodeToString(hasher.Sum(nil)); hash == p.Hash {
		fmt.Println("Hashes match!")
	} else {
		fmt.Println("Hashes differ :(")
		fmt.Println(hash)
		fmt.Println(p.Hash)
		if err := os.Remove(dir); err != nil {
			fmt.Println("error removing file:", err)
		}
		return errors.New("Hash of downloaded file does not match NPM")
	}
	return nil
}
