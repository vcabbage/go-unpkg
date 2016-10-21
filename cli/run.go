package cli

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/vcabbage/go-unpkg/npm"
)

func Run() int {
	p := npm.Parse(os.Args[1])

	if err := p.Resolve(); err != nil {
		fmt.Println(err)
		return 1
	}

	fmt.Printf("%#v\n", p)

	_, err := p.Download("cache")
	if err != nil {
		fmt.Println("Error downloading:", err)
		return 1
	}

	fullpath := filepath.Join("cache", p.FilePath())
	if p.IsDir {
		files, err := ioutil.ReadDir(fullpath)
		if err != nil {
			fmt.Println("error reading dir:", err)
		}

		for _, f := range files {
			var suffix string
			if f.IsDir() {
				suffix = "/"
			}
			fmt.Println(f.Name() + suffix)
		}
		return 0
	}

	f, err := os.Open(fullpath)
	if err != nil {
		fmt.Printf("error opening %q: %v\n", p.Path, err)
		return 1
	}
	defer f.Close()

	io.Copy(os.Stdout, f)

	return 0
}
