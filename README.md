unpkg in Go. WIP, but many features work.

Missing Features:
* Bower Bundle Generation (`/react-swap/bower.zip`)
* `main` query parameter (overrides package.json field to use as entry point)
* `json` query parameter (list directories in JSON)

Download
```
go get -u -v github.com/vcabbage/go-unpkg
```

Run
```
$GOPATH/bin/go-unpkg serve [-listen ":80"] [-cacheDir "/tmp/unpkg"]
```