# emlang

The toolchain for [Emlang](https://emlang-project.github.io/), a YAML-based domain-specific language for expressing [Event Modeling](https://eventmodeling.org/) patterns.

## Overview

This toolchain provides parsing, linting, formatting, and HTML diagram generation for Emlang documents, as a CLI and as a WebAssembly module. It implements the [Emlang Specification v1.0.0](https://github.com/emlang-project/spec).

## Installation

### Prebuilt binaries

Download the binary for your platform from the [releases page](https://github.com/emlang-project/emlang/releases). Each release includes a `SHA256SUMS` file to verify the downloads:

```bash
sha256sum -c SHA256SUMS --ignore-missing
```

### Using Go

Requires [Go](https://go.dev/dl/) 1.27.2 or later.

```bash
go install github.com/emlang-project/emlang/cmd/emlang@latest
```

### From source

Requires [Go](https://go.dev/dl/) 1.27.2 or later.

```bash
git clone https://github.com/emlang-project/emlang.git
cd emlang
make build
```

The binary will be available at `bin/emlang`.

The version reported by `emlang version` comes from the git tag (`git describe --tags --always --dirty`), for example `v1.1.0` on a tag or `v1.1.0-3-gabc1234` after it. A binary built without the Makefile, such as with a plain `go build`, reports `dev`.

## Usage

```bash
emlang [-c <config>] <command> [arguments]
```

### Flags

| Flag | Description |
|------|-------------|
| `-c`, `--config <file>` | Path to config file (default: `.emlang.yaml`, or `EMLANG_CONFIG` env) |

### Commands

| Command | Description |
|---------|-------------|
| `parse <file>` | Parse and display document structure |
| `lint <file>` | Analyze for issues and best practices |
| `fmt <file>` | Format a document (`-w` to write in place, `--keys short\|long` for the key style) |
| `diagram <file>` | Generate an HTML diagram (`-o file`, or `--serve [--address 127.0.0.1] [--port 8274]` for a live-reload server) |
| `init` | Create a `.emlang.yaml` config file with defaults |
| `version` | Print version information |
| `help` | Show help message |

Use `-` instead of a filename to read from stdin (not available with `fmt -w` or `diagram --serve`).

### Formatting

`emlang fmt` rewrites a document in a canonical form:

- element keys use one style: long (`event`, the default) or short (`e`);
- a slice without tests uses the direct form (list of elements, no `steps:` key);
- keys are ordered: `steps` before `tests`, `given`, `when` then `then`, the element type before `props`;
- element names and swimlanes are trimmed (`" Lane / Name "` becomes `"Lane/Name"`).

Comments, quoting, and prop values are kept as written.

## Configuration

The config file is resolved in order: `-c` flag, `EMLANG_CONFIG` env, `.emlang.yaml` in the current directory. Run `emlang init` to create one with all options documented.

```yaml
lint:
  ignore:
    - slice-missing-event
fmt:
  keys: short
diagram:
  serve:
    address: 127.0.0.1
    port: 8274
  css:
    --command-color: "#a5d8ff"
```

## Linter Rules

| Rule | Severity | Description |
|------|----------|-------------|
| `empty-slice` | error | Slice without elements |
| `slice-missing-event` | warning | Slice without events |
| `command-without-event` | warning | Command not followed by event or exception |
| `orphan-exception` | warning | Exception without preceding command |
| `test-missing-command` | error | Test without command (when) |
| `test-missing-then` | error | Test without outcomes (then) |
| `test-invalid-when` | error | When clause must be a command |
| `test-invalid-given` | error | Given can only contain events or views |
| `test-invalid-then` | error | Then can only contain events, views, or exceptions |
| `trigger-in-test` | error | Triggers not allowed in tests |

## WebAssembly

`make build-wasm` builds `bin/emlang.wasm` and copies Go's `wasm_exec.js` next to it. Once loaded, the module calls `onEmlangReady()` if defined, and exposes these global functions:

| Function | Returns |
|----------|---------|
| `emlangRender(source, css?)` | `{html, lint}`, or `{error}`. `css` is an optional object of CSS variable overrides. |
| `emlangFormat(source, keys?)` | `{yaml}`, or `{error}`. `keys` is `"long"` (default) or `"short"`. |
| `emlangVersion()` | The version string. |

## Development

```bash
make build       # Build optimized binary
make build-wasm  # Build WebAssembly module
make test        # Run tests
make vet         # Run go vet
make lint        # Run golangci-lint
make fmt         # Format code
make help        # Show all targets
```

The CI runs `gofmt`, `go vet`, the build (including WebAssembly), the tests with and without the race detector, [staticcheck](https://staticcheck.dev/) and [govulncheck](https://go.dev/doc/security/vuln/) on every push and pull request to `main`.

Pushing a `v*` tag publishes a release with binaries for Linux, macOS, Windows and WebAssembly, built with that tag as their version.

## License

[MIT](LICENSE)
