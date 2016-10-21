package server

import (
	"errors"
	"sync"

	"time"

	"github.com/vcabbage/go-unpkg/npm"
)

// cache handles retrieving package metadata from NPM
type cache struct {
	// chaches the resolved package for the resolved version
	// these should not change and are not timed out
	resolvedMu   sync.RWMutex
	resolvedPkgs map[string]npm.Package

	// caches the resolved Package for the unresolved version
	// theses are timed out
	unresolvedMu      sync.RWMutex
	unresolvedPkgs    map[string]npm.Package
	unresolvedTimeIdx []timeIdx

	// used by the cache cleaner
	timeout time.Duration
}

// timeIdx correlates a index with the time it was added
type timeIdx struct {
	unixTime int64
	i        string
}

// newCache creates a new cache and starts the cache cleaner goroutine
func newCache(timeout time.Duration) *cache {
	c := &cache{
		resolvedPkgs:   make(map[string]npm.Package),
		unresolvedPkgs: make(map[string]npm.Package),
		timeout:        timeout,
	}

	c.startCleaner()

	return c
}

// errVersionChanged is used to indicate thate the version changed after resolving the package
var errVersionChanged = errors.New("version changed")

// getPackage tries retrieving packages from cache, failing that it will resolve the package
func (c *cache) getPackage(pkg string) (npm.Package, error) {
	p := npm.Parse(pkg)

	c.resolvedMu.RLock()
	cached, ok := c.resolvedPkgs[p.Name+p.Version]
	c.resolvedMu.RUnlock()
	if ok {
		cached.Path = p.Path
		cached.IsDir = p.IsDir
		return cached, nil
	}

	c.unresolvedMu.RLock()
	cached, ok = c.unresolvedPkgs[p.Name+p.Version]
	c.unresolvedMu.RUnlock()
	if ok {
		cached.Path = p.Path
		cached.IsDir = p.IsDir
		return cached, nil
	}

	origVer := p.Version
	if err := p.Resolve(); err != nil {
		return *p, err
	}
	c.addPackage(p, origVer)

	if origVer != p.Version {
		return *p, errVersionChanged
	}

	return *p, nil
}

// addPackage adds a resolved package to the cache. If any unresolvedVersions
// are specified, they will be added to the unresolvedCache.
func (c *cache) addPackage(p *npm.Package, unresolvedVersions ...string) {
	c.resolvedMu.Lock()
	c.resolvedPkgs[p.Name+p.Version] = *p
	c.resolvedMu.Unlock()

	c.unresolvedMu.Lock()
	for _, version := range unresolvedVersions {
		i := p.Name + version
		c.unresolvedPkgs[i] = *p
		c.unresolvedTimeIdx = append(c.unresolvedTimeIdx, timeIdx{time.Now().Unix(), i})
	}
	c.unresolvedMu.Unlock()
}

// startCleaners starts the cleaner goroutine and returns
func (c *cache) startCleaner() {
	if c.timeout <= 0 {
		return
	}
	go func() {
		timeout := time.NewTicker(c.timeout)
		for {
			<-timeout.C
			c.clean()
		}
	}()
}

// clean removes all entries from unresolvedCache that are older
// than timeout
func (c *cache) clean() {
	keep := time.Now().Unix() - int64(c.timeout.Seconds())

	var lastIdx int
	c.unresolvedMu.Lock()
	for i, timeI := range c.unresolvedTimeIdx {
		if timeI.unixTime > keep {
			lastIdx = i
			break
		}
		delete(c.unresolvedPkgs, timeI.i)
	}
	c.unresolvedTimeIdx = c.unresolvedTimeIdx[lastIdx:]
	c.unresolvedMu.Unlock()
}
