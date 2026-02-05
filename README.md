# emlang

The toolchain for [Emlang](https://emlang-project.github.io/), a YAML-based domain-specific language for expressing [Event Modeling](https://eventmodeling.org/) patterns.

## Overview

This toolchain provides parsing, linting, and HTML diagram generation for Emlang documents. It implements the [Emlang Specification v1.0.0](https://github.com/emlang-project/spec).

## Prerequisites

- [Go](https://go.dev/dl/) 1.21 or later

## Installation

### From source

```bash
git clone https://github.com/emlang-project/emlang.git
cd emlang
make build
```

The binary will be available at `bin/emlang`.

### Using Go

```bash
go install github.com/emlang-project/emlang/cmd/emlang@latest
```

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
| `diagram <file>` | Generate an HTML diagram (`-o file`) |
| `version` | Print version information |
| `help` | Show help message |

Use `-` instead of a filename to read from stdin.

## Configuration

The config file is resolved in order: `-c` flag, `EMLANG_CONFIG` env, `.emlang.yaml` in the current directory.

```yaml
lint:
  ignore:
    - slice-missing-event
diagram:
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

## Development

```bash
make build       # Build optimized binary
make test        # Run tests
make lint        # Run golangci-lint
make fmt         # Format code
make help        # Show all targets
```

## License

[MIT](LICENSE)
