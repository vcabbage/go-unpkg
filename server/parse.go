package server

import (
	"fmt"
	"regexp"
)

// urlRegex parses [/]name[@version][path] into name, version, path
var urlRegex = regexp.MustCompile("^/?([^@/]+)@?([^/]*)?(/.*)?")

type parsed struct {
	Name    string
	Version string
	Path    string
}

// Parse parses a package@version/file/path into individual parts
func parseURL(s string) (*parsed, error) {
	p := &parsed{Version: "latest"} // Default to latest

	submatches := urlRegex.FindStringSubmatch(s)
	if len(submatches) < 4 {
		return nil, fmt.Errorf("unable to parse: %q", s)
	}

	p.Name = submatches[1]
	if ver := submatches[2]; ver != "" {
		p.Version = ver
	}
	p.Path = submatches[3]

	return p, nil
}
