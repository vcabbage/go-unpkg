package npm

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/vcabbage/go-unpkg/extract"
)

type Package struct {
	Name       string
	Version    string
	Path       string
	IsDir      bool
	IsResolved bool
	Hash       string
	URL        string
}

// Parse parses a package@version/file/path into a Package
func Parse(s string) *Package {
	p := &Package{Version: "latest"} // Default to latest

	s = strings.TrimPrefix(s, "/") // Remove leading any leading "/"

	// If there is a trailing slash a directory listing is being requested
	if s[len(s)-1] == '/' {
		p.IsDir = true
	}

	nameFile := strings.SplitN(s, "/", 2)
	nameVer := strings.SplitN(nameFile[0], "@", 2)

	p.Name = nameVer[0]
	if len(nameFile) == 2 {
		p.Path = nameFile[1]
	}

	if len(nameVer) == 2 && nameVer[1] != "" {
		p.Version = nameVer[1]
	}
	return p
}

// Resolve retrieves package metadata from NPM and update Package
func (p *Package) Resolve() error {
	if p.IsResolved {
		return nil
	}

	url := "https://registry.npmjs.org/" + p.Name + "/" + p.Version
	client := &http.Client{Timeout: 10 * time.Second} // TODO: Reuse client
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New("bad response: " + resp.Status)
	}

	var n struct {
		Version string
		Main    string
		Bower   string // TODO: Bower could be an object
		Dist    struct {
			SHASum  string
			TARBall string
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&n); err != nil {
		return err
	}

	p.Version = n.Version

	if p.Path == "" && !p.IsDir {
		switch {
		case n.Bower != "":
			p.Path = n.Bower
		case n.Main != "":
			p.Path = n.Main
		default:
			return errors.New("bower/main requested but not available")
		}
	}

	p.Hash = n.Dist.SHASum
	p.URL = strings.Replace(n.Dist.TARBall, "http://", "https://", 1) // Use HTTPS

	p.IsResolved = true

	return nil
}

// Download downloads and extracts the package from NPM into dest.
//
// If the hash does not match the metadata from NPM the downloaded file is deleted
// and and error is returned.
func (p *Package) Download(dest string) (string, error) {
	name := path.Base(p.URL)
	dir := filepath.Join(dest, name[:len(name)-len(filepath.Ext(name))])

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Println(name, "not cached, downloading...")
	} else {
		fmt.Println(name, "cached.")
		return "", nil
	}

	resp, err := http.Get(p.URL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Println("bad response when downloading: ", resp.Status)
		return "", err
	}

	hasher := sha1.New()

	tee := io.TeeReader(resp.Body, hasher)

	if err := extract.TGZ(tee, dir); err != nil {
		fmt.Println("error extracting tgz:", err)
		return "", err
	}

	if hash := hex.EncodeToString(hasher.Sum(nil)); hash != p.Hash {
		fmt.Println("Hashes differ :(")
		fmt.Println(hash)
		fmt.Println(p.Hash)
		if err := os.RemoveAll(dir); err != nil {
			fmt.Println("error removing file:", err)
		}
		return "", errors.New("Hash of downloaded file does not match hash from NPM")
	}
	return dir, nil
}

// UnpkgURL returns the relative URL for this package for an unpkg server.
func (p *Package) UnpkgURL() string {
	s := "/" + p.Name
	if p.Version != "" {
		s += "@" + p.Version
	}
	if p.Path != "" {
		s += "/" + p.Path
	} else if p.IsDir {
		s += "/"
	}

	return s
}

// FilePath returns the Path relative the the dest directory after
// downloading the Package with Download.
func (p *Package) FilePath() string {
	return filepath.Join(p.Name+"-"+p.Version, p.Path)
}
