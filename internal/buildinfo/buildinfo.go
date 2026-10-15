package buildinfo

// Version is the running build's version, baked in at build time. It stays
// "dev" for a binary built without the Makefile's ldflags, e.g. via a plain
// `go build` or `go run`.
var Version = "dev"
