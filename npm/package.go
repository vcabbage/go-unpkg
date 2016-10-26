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
	"regexp"
	"strings"
	"time"

	"github.com/vcabbage/go-unpkg/extract"
)

// urlRegex parses [/]name[@version][path] into name, version, path
var urlRegex = regexp.MustCompile("^([^@/]+)@?([^/]*)?(/.*)?")

type Package struct {
	Name    string
	Version string
	Hash    string
	URL     string
}

type Parsed struct {
	Name    string
	Version string
	Path    string
}

// Parse parses a package@version/file/path into a Package
func Parse(s string) *Parsed {
	p := &Parsed{Version: "latest"} // Default to latest

	submatches := urlRegex.FindStringSubmatch(s)
	if len(submatches) < 4 {
		return p // TODO: error?
	}

	p.Name = submatches[1]
	if ver := submatches[2]; ver != "" {
		p.Version = ver
	}
	p.Path = submatches[3]

	return p
}

// Resolve retrieves package metadata from NPM and update Package
func Resolve(name, version string) (*Package, error) {
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
		Bower   string // TODO: Bower could be an object
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

	// if p.Path == "" && Is {
	// 	switch {
	// 	case n.Bower != "":
	// 		p.Path = n.Bower
	// 	case n.Main != "":
	// 		p.Path = n.Main
	// 	default:
	// 		return errors.New("bower/main requested but not available")
	// 	}
	// }

	p.Hash = n.Dist.SHASum
	p.URL = strings.Replace(n.Dist.TARBall, "http://", "https://", 1) // Use HTTPS

	return p, nil
}

// Download downloads and extracts the package from NPM into dest.
//
// If the hash does not match the metadata from NPM the downloaded file is deleted
// and and error is returned.
func Download(url, hash, dest string) (string, error) {
	name := path.Base(url)
	dir := filepath.Join(dest, name[:len(name)-len(filepath.Ext(name))])

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Println(name, "not cached, downloading...")
	} else {
		fmt.Println(name, "cached.")
		return "", nil
	}

	resp, err := http.Get(url)
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

	if dHash := hex.EncodeToString(hasher.Sum(nil)); dHash != hash {
		fmt.Println("Hashes differ :(")
		fmt.Println(dHash)
		fmt.Println(hash)
		if err := os.RemoveAll(dir); err != nil {
			fmt.Println("error removing file:", err)
		}
		return "", errors.New("Hash of downloaded file does not match hash from NPM")
	}
	return dir, nil
}

// DownloadPath returns the Path relative the the dest directory after
// downloading the Package with Download.
func (p *Package) DownloadPath(dir, path string) string {
	return filepath.Join(dir, p.Name+"-"+p.Version, path)
}
