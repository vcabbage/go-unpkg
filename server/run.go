package server

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/vcabbage/go-unpkg/npm"

	"os/signal"

	"golang.org/x/sync/singleflight"
)

var Metrics = struct {
	requests *prometheus.CounterVec
}{
	requests: prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "unpkg_requests_count",
		Help: "Count of requested packages",
	}, []string{"package"}),
}

// Run starts the HTTP server
//
// Returns the process exit code for use in a main package
func Run() int {
	var (
		cacheDir      = flag.String("cacheDir", "cache", "directory to store cached packages")
		cacheTimeout  = flag.Duration("cacheTimeout", 5*time.Minute, "length of time to cache package metadata")
		listen        = flag.String("listen", "localhost:8080", "Address and port to listen on")
		enableMetrics = flag.Bool("metrics", true, "enable prometheus metric collection")
	)
	flag.Parse()

	c := newCache(*cacheTimeout)

	mux := http.NewServeMux()

	mux.Handle("/", &handler{c: c, cacheDir: *cacheDir})

	if *enableMetrics {
		mux.Handle("/metrics", prometheus.Handler())

		prometheus.MustRegister(Metrics.requests)
	}

	srv := http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 60 * time.Second,
		Handler:      mux,
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
	urlPath := r.URL.Path // Trim starting slash
	log.Printf("New Request for %q\n", urlPath)
	Metrics.requests.WithLabelValues(urlPath).Add(1)

	parsed := npm.Parse(urlPath)
	path := parsed.Path

	// Get the package metadata from the cache
	pkg, err := h.c.getPackage(parsed.Name, parsed.Version)
	if err != nil {
		// Not in cache
		pkg, err = npm.Resolve(parsed.Name, parsed.Version)
		if err != nil {
			log.Println("Error resolving package:", err)
			// TODO: Don't pass the raw error through
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.c.addPackage(pkg, parsed.Version)
	}
	if pkg.Version != parsed.Version {
		// If the version changed from what was requested, send a redirect
		http.Redirect(w, r, unpkgURL(pkg.Name, pkg.Version, path), http.StatusTemporaryRedirect)
		return
	}

	fullpath := pkg.DownloadPath(h.cacheDir, path)

	// Try to send from file cache
	if tryFileCache(w, r, fullpath) {
		log.Printf("Found %q in file cache\n", fullpath)
		// Success, we're done
		return
	}

	// Need to download the package
	log.Printf("%q not found in file cache, downloading...\n", fullpath)

	// Use singleflight to supress downloading the same package concurrently
	_, err, _ = h.sf.Do(pkg.URL, func() (interface{}, error) {
		return npm.Download(pkg.Name, pkg.Version, h.cacheDir)
	})
	if err != nil {
		log.Printf("Error downloading %q: %v\n", pkg.URL, err)
		// TODO: Don't pass the raw error through
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("%q %s download complete\n", pkg.Name, pkg.Version)

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

// unpkgURL returns the relative URL for this package for an unpkg server.
func unpkgURL(name, version, path string) string {
	s := "/" + name
	if version != "" {
		s += "@" + version
	}
	s += path

	return s
}
