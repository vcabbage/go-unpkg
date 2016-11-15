package server

import "testing"

var parseTests = map[string]struct {
	in   string
	want parsed
}{
	"name,version,filepath": {
		in:   "react@15.3.1/dist/react.min.js",
		want: parsed{Name: "react", Version: "15.3.1", Path: "/dist/react.min.js"},
	},
	"name,version,directory": {
		in:   "react@15.3.1/dist/",
		want: parsed{Name: "react", Version: "15.3.1", Path: "/dist/"},
	},
	"name,pattern version,directory": {
		in:   "react@^14.0.0/dist/",
		want: parsed{Name: "react", Version: "^14.0.0", Path: "/dist/"},
	},
	"name,root dir": {
		in:   "react/",
		want: parsed{Name: "react", Version: "latest", Path: "/"},
	},
	"name only": {
		in:   "react",
		want: parsed{Name: "react", Version: "latest", Path: ""},
	},
	"name,bad version,filepath": {
		in:   "react@/dist/react.min.js",
		want: parsed{Name: "react", Version: "latest", Path: "/dist/react.min.js"},
	},
	"name,filepath": {
		in:   "react/dist/react.min.js",
		want: parsed{Name: "react", Version: "latest", Path: "/dist/react.min.js"},
	},
}

func TestParse(t *testing.T) {
	for label, tt := range parseTests {
		t.Run(label, func(t *testing.T) {
			if got, _ := parseURL(tt.in); tt.want != *got {
				t.Errorf("Parse(%s) = %+v, want %+v", tt.in, *got, tt.want)
			}
		})
	}
}
