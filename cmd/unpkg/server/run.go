package server

import (
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/vcabbage/unpkg"
)

func Run() int {
	http.HandleFunc("/", handler)

	http.ListenAndServe(":8090", nil)
	return 0
}

func handler(w http.ResponseWriter, r *http.Request) {
	pkg := strings.TrimPrefix(r.URL.Path, "/")
	fmt.Println("request:", pkg)
	p, err := unpkg.Get(pkg)
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), 500)
		return
	}

	fmt.Printf("%#v\n", p)

	if err := unpkg.Download(p, "cache"); err != nil {
		fmt.Println("Error downloading:", err)
		http.Error(w, err.Error(), 500)
		return
	}

	dir := filepath.Join("cache", path.Base(p.URL))

	if ct, ok := fileTypes[strings.ToLower(path.Ext(p.Path))]; ok {
		w.Header().Set("Content-Type", ct)
	}

	fullpath := filepath.Join(dir, p.Path)
	http.ServeFile(w, r, fullpath)
}

var fileTypes = map[string]string{
	".md": "text/x-markdown",
}
