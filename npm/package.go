package npm

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vcabbage/go-unpkg/extract"
)

type Package struct {
	Name    string
	Version string
	Hash    string
	URL     string
	Main    string
	Browser string
}

// GetMetadata retrieves package metadata from NPM and update Package
func GetMetadata(name, version string) (*Package, error) {
	url := "https://registry.npmjs.org/" + name + "/" + version
	client := &http.Client{Timeout: 10 * time.Second} // TODO: Reuse client
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New("bad response: " + resp.Status)
	}

	var n struct {
		Version string
		Main    string
		Browser string // TODO: Browser could be an object
		Dist    struct {
			SHASum  string
			TARBall string
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&n); err != nil {
		return nil, err
	}

	p := &Package{Name: name}

	p.Version = n.Version
	p.Main = n.Main
	p.Browser = n.Browser
	p.Hash = n.Dist.SHASum
	p.URL = strings.Replace(n.Dist.TARBall, "http://", "https://", 1) // Use HTTPS

	return p, nil
}

// Download downloads and extracts the package from NPM into dest.
//
// If the downloaded file does not match the provided hash an error is returned.
func Download(url, hash, dest string) error {
	resp, err := http.Get(url)
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

	if err := extract.TGZ(tee, dest); err != nil {
		fmt.Println("error extracting tgz:", err)
		return err
	}

	if dHash := hex.EncodeToString(hasher.Sum(nil)); dHash != hash {
		fmt.Println("Hashes differ :(")
		fmt.Println(dHash)
		fmt.Println(hash)
		return errors.New("Hash of downloaded file does not match hash from NPM")
	}
	return nil
}
