<img src="docs/logo.svg" alt="Emlang" width="160">

The toolchain for [Emlang](https://emlang-project.github.io/), a YAML-based language for [Event Modeling](https://eventmodeling.org/). It implements the [Emlang Specification v1.0.0](https://github.com/emlang-project/spec).

[![release](https://img.shields.io/github/v/release/emlang-project/emlang)](https://github.com/emlang-project/emlang/releases/latest)
[![ci](https://github.com/emlang-project/emlang/actions/workflows/ci.yml/badge.svg)](https://github.com/emlang-project/emlang/actions/workflows/ci.yml)
[![license](https://img.shields.io/github/license/emlang-project/emlang)](LICENSE)

## Features

- Lint: checks a model against the specification and common Event Modeling mistakes.
- Format: rewrites a model in a canonical form, keeping its comments.
- Diagram: generates an HTML diagram, with a live-reload server while editing.
- WebAssembly: the same parser, linter, formatter and diagram run in the browser.
- One static binary for Linux, macOS and Windows.

## Installation

```bash
go install github.com/emlang-project/emlang/cmd/emlang@latest
```

This needs Go 1.27.2 or later. Precompiled binaries are on the [releases page](https://github.com/emlang-project/emlang/releases).

## Documentation

The full documentation, with a live playground, is at <https://emlang-project.github.io/>.

## Contributing

Issues and pull requests are welcome; see [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
