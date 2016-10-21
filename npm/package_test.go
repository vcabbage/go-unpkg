package npm

import "testing"

var parseTests = map[string]struct {
	in   string
	want Package
}{
	"name,version,filepath": {
		in:   "react@15.3.1/dist/react.min.js",
		want: Package{Name: "react", Version: "15.3.1", Path: "dist/react.min.js"},
	},
	"name,version,directory": {
		in:   "react@15.3.1/dist/",
		want: Package{Name: "react", Version: "15.3.1", Path: "dist/", IsDir: true},
	},
	"name,root dir": {
		in:   "react/",
		want: Package{Name: "react", Version: "latest", Path: "", IsDir: true},
	},
	"name only": {
		in:   "react",
		want: Package{Name: "react", Version: "latest", Path: ""},
	},
	"name,bad version,filepath": {
		in:   "react@/dist/react.min.js",
		want: Package{Name: "react", Version: "latest", Path: "dist/react.min.js"},
	},
	"name,filepath": {
		in:   "react/dist/react.min.js",
		want: Package{Name: "react", Version: "latest", Path: "dist/react.min.js"},
	},
}

func TestParse(t *testing.T) {
	for label, tt := range parseTests {
		t.Run(label, func(t *testing.T) {
			if got := Parse(tt.in); tt.want != *got {
				t.Errorf("Parse(%s) = %+v, want %+v", tt.in, *got, tt.want)
			}
		})
	}
}
