package npm

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
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

func Parse(s string) *Package {
	p := &Package{Version: "latest"}

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

type npmResponse struct {
	Version string
	Main    string
	Browser string // TODO: Browser could be an object
	Dist    struct {
		SHASum  string
		TARBall string
	}
}

func Resolve(p *Package) error {
	if p.IsResolved {
		return nil
	}

	url := "http://registry.npmjs.org/" + p.Name + "/" + p.Version
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New("bad response: " + resp.Status)
	}

	var n npmResponse
	if err := json.NewDecoder(resp.Body).Decode(&n); err != nil {
		return err
	}

	p.Version = n.Version

	if p.Path == "" && !p.IsDir {
		switch {
		case n.Browser != "":
			p.Path = n.Browser
		case n.Main != "":
			p.Path = n.Main
		default:
			return errors.New("browser/main requested but not available")
		}
	}

	p.Hash = n.Dist.SHASum
	p.URL = n.Dist.TARBall

	p.IsResolved = true

	return nil
}
