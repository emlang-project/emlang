# Contributing to emlang

Issues and pull requests are welcome.

## Reporting bugs and proposing ideas

Open an issue. Include enough detail to reproduce a bug, such as the
YAML input and the command you ran, or explain the problem an idea
would solve.

## Pull requests

- Open an issue first for anything beyond a small fix, so the approach
  can be discussed before you spend time on it.
- Keep pull requests focused on one change.
- Run `make fmt`, `make vet`, and `make test` before opening a pull
  request.
- A change the user can see updates the website in `docs/` in the same
  pull request.
- Follow the existing code and documentation style. The writing rules
  are in [CLAUDE.md](CLAUDE.md).

## Building from source

You need Go 1.27.2 or later.

```sh
make build
```

This builds `bin/emlang`. The Makefile bakes the output of
`git describe --tags --always --dirty` into the binary, by setting
`internal/buildinfo.Version` through `-ldflags "-X ..."`. A binary
built without the Makefile, with a plain `go build` or `go run`,
reports `dev`.

Other Makefile targets:

- `make build-all`: binaries for Linux, macOS and Windows.
- `make build-wasm`: the WebAssembly module, `bin/emlang.wasm`, and the
  matching `bin/wasm_exec.js` from the Go installation.
- `make build-dev`: a binary with debug symbols.
- `make test` / `make test-cover`: the tests, and a coverage report.
- `make fmt` / `make vet`: standard Go housekeeping.
- `make clean`: removes `bin/` and the coverage files.
- `make help`: lists every target.

## Checks

The CI runs on every push and pull request to `main`:

- `gofmt`, `go vet`, and the build, including the WebAssembly module;
- `go test ./...`, and `go test -race ./...`;
- [staticcheck](https://staticcheck.dev/) and
  [govulncheck](https://go.dev/doc/security/vuln/).

To run the last two locally:

```sh
go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
```

## WebAssembly module

The website's playground loads `emlang.wasm` with Go's `wasm_exec.js`.
The module calls `onEmlangReady()` if it is defined, then exposes these
global functions:

| Function | Returns |
|----------|---------|
| `emlangRender(source, css?)` | `{html, lint}`, or `{error}`. `css` is an optional object of CSS variable overrides. |
| `emlangFormat(source, keys?)` | `{yaml}`, or `{error}`. `keys` is `"short"` (default) or `"long"`. |
| `emlangVersion()` | The version string. |

## Website

The website, <https://emlang-project.github.io/>, is the static HTML in
`docs/`, without a build step. To preview it with the playground:

```sh
make build-wasm
mkdir -p tmp/site
cp -r docs/. bin/emlang.wasm bin/wasm_exec.js tmp/site/
python3 -m http.server -d tmp/site
```

The site is served at `http://localhost:8000/`. The navigation bar
shows `dev` as the version. The publishing workflow replaces it with the
latest tag.

`emlang.wasm` and `wasm_exec.js` are never committed: the publishing
workflow builds them.

## Releases

Pushing a `v*` tag runs two workflows:

- `release.yml` builds the binaries and the WebAssembly module with the
  tag as their version, and publishes them in a GitHub release with a
  `SHA256SUMS` file.
- `docs.yml` publishes the website, with the WebAssembly module of the
  tag, to the `emlang-project.github.io` repository, which serves it.

A maintainer can also publish the website by hand with
`gh workflow run docs.yml --ref <branch-or-tag>`. Never edit the
`emlang-project.github.io` repository: each publication replaces its
content.
