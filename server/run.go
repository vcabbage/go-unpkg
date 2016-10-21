package server

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"os/signal"

	"golang.org/x/sync/singleflight"
)

// Run starts the HTTP server
//
// Returns the process exit code for use in a main package
func Run() int {
	var (
		cacheTimeout = flag.Duration("cacheTimeout", 5*time.Minute, "length of time to cache package metadata")
		listen       = flag.String("listen", "localhost:8080", "Address and port to listen on")
	)
	flag.Parse()

	c := newCache(*cacheTimeout)

	srv := http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 60 * time.Second,
		Handler:      &handler{c: c, cacheDir: "cache"},
		Addr:         *listen,
	}

	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			log.Fatalln("ListenAndServe:", err)
		}
	}()

	log.Printf("Listening on %s...\n", srv.Addr)

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)
	<-sig
	log.Println("Shutting down...")
	return 0
}

// handler contains dependencies shared between all requests
type handler struct {
	c        *cache
	cacheDir string
	sf       singleflight.Group
}

// ServeHTTP handles each request to the server in a seperate goroutine
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pkg := strings.TrimPrefix(r.URL.Path, "/") // Trim starting slash
	log.Printf("New Request for %q\n", pkg)

	// Get the package from the cache
	p, err := h.c.getPackage(pkg)
	if err == errVersionChanged {
		// If the version was changed from what was requested, send a redirect
		http.Redirect(w, r, p.UnpkgURL(), http.StatusTemporaryRedirect)
		return
	}
	if err != nil {
		log.Println("Error retriving package from cache:", err)
		// TODO: Don't pass the raw error through
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fullpath := filepath.Join(h.cacheDir, p.FilePath())

	// Try to send from file cache
	if tryFileCache(w, r, fullpath) {
		log.Printf("Found %q in file cache\n", fullpath)
		// Success, we're done
		return
	}

	// Need to download the package
	log.Printf("%q not found in file cache, downloading...\n", fullpath)

	// Use singleflight to supress downloading the same package concurrently
	_, err, _ = h.sf.Do(p.URL, func() (interface{}, error) {
		return p.Download(h.cacheDir)
	})
	if err != nil {
		log.Printf("Error downloading %q: %v\n", p.URL, err)
		// TODO: Don't pass the raw error through
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("%q %s download complete\n", p.Name, p.Version)

	serveFile(w, r, fullpath)
}

// fileTypes defines custom content types for file extensions
//
// http.ServeFile will handle more common file extensions
var fileTypes = map[string]string{
	".md": "text/x-markdown",
}

// serveFile sends the file or lists the directory at p
func serveFile(w http.ResponseWriter, r *http.Request, p string) {
	if ct, ok := fileTypes[strings.ToLower(path.Ext(p))]; ok {
		w.Header().Set("Content-Type", ct)
	}
	http.ServeFile(w, r, p)
}

// tryFileCache will send the file at p if the file exists, returns true if successful
func tryFileCache(w http.ResponseWriter, r *http.Request, p string) bool {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return false
	}

	serveFile(w, r, p)
	return true
}
