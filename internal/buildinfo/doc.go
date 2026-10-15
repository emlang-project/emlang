// Package buildinfo holds the version string shared by the emlang CLI and
// the WebAssembly module. It's set at build time, with
// -ldflags "-X github.com/emlang-project/emlang/internal/buildinfo.Version=...",
// and never read or changed at runtime.
package buildinfo
