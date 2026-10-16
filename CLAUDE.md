# Emlang specification

The toolchain implements the Emlang specification, kept in its own
repository:

- https://github.com/emlang-project/spec/blob/main/SPEC.md
- https://github.com/emlang-project/spec/blob/main/schema.json

# Documentation writing guidelines

These rules apply to all documentation in this repo: `README.md`,
`CONTRIBUTING.md`, every page under `docs/`, and prose in Go comments.

## Style

- Short sentences. Simple, direct English.
- Avoid technical jargon that isn't needed, and business buzzwords
  (leverage, seamless, robust, best-in-class, synergy, etc.).
- Necessary technical terms stay (YAML, WebAssembly, swimlane, slice,
  Given-When-Then, etc.): they are the real names of the things being
  described, not jargon to eliminate. Simplify sentence length and
  phrasing around them, not the technical precision itself.
- Never use an em-dash. Use a comma, colon, semicolon, parentheses, or
  a new sentence instead.
- Never chain clauses into one long sentence with a colon. Split it into
  two or three sentences.
- Every sentence earns its place: it states what to do, or a behavior the
  reader relies on. No asides, no trivia, no comparisons with other
  tools.
- Never restate the previous sentence in other words. Say it once.

## No references to prior designs

Write the current design as if it had always been the only one. Never say
"unlike before," "the old X," "this replaces Y," or similar, in code
comments or docs. Describe the current thing on its own terms. If
historical rationale needs to be preserved, put it in a commit message or
PR description, not in the doc or comment itself.

## Documentation structure

The site serves two readers, in this order: whoever writes Emlang models
(with the CLI or the playground), and whoever integrates the toolchain
(CI pipelines, editors, the WebAssembly module).

The doc follows the code: a change the user can see updates `docs/` in
the same commit. A fact is written where its reader needs it, even if
another page says it too. Keep pages short: document an edge case only
when it affects the reader in practice.

Pages, under `docs/`. The navigation bar links to Specification, CLI
Reference, and Try Emlang, in that order.

- `index.html`: the home page. What Emlang is, examples, getting
  started (install, first diagram), use cases.
- `spec.html`: a readable summary of the specification. It follows
  `SPEC.md` in the spec repository and links to it; when the spec
  changes, this page changes too.
- `cli.html`: the CLI reference. Installation, then one section per
  command with its usage, arguments, flags, and examples.
- `config.html`: the only reference for `.emlang.yaml`, one section per
  key. Reached from `cli.html`, not from the navigation bar.
- `repl.html`: the playground ("Try Emlang"), running the WebAssembly
  module.

`README.md` is kept short: logo, a one-paragraph intro, badges, a short
feature list, the `go install` command, a single link to the site
without listing its pages, and a contributing note. No command tables,
no config examples, no build instructions live in the README itself.

How the toolchain works inside is not on the site: it lives in Go
comments, next to the code it explains (see below). Building, testing,
the WebAssembly API, previewing the site, and releasing are in
`CONTRIBUTING.md`.

## Page layout

- A page or section opens with one or two sentences on what it covers.
- Rules and behaviors as a bulleted list, one per bullet.
- Write rules as ordinary sentences. No RFC 2119 key words on the site.
- No "Why" paragraphs: the doc says what to do and what the toolchain
  does. A reader who wants the reason reads the code and its comments.
- Reuse the markup the pages already use: `code-wrapper` with a
  `copy-btn` for commands to copy, `table-wrapper` around tables,
  `ul.plain` or `ul.rules` for lists, and `hljs-*` spans for YAML
  highlighting. Don't add a CSS framework or a build step.
- YAML examples use the short key style (`t`, `c`, `e`, `x`, `v`), the
  formatter's default, unless the example is about the long style.
- Link between pages with relative paths and anchors, e.g.
  `config.html#fmt`.

## Toolchain internals in Go comments

How the toolchain works inside is documented in Go comments, never on the
site: a comment lives in the same diff as its code, so it follows its
changes.

- Each package has a `doc.go` holding its package comment: its role, how
  it works, and the reason for each design choice that a rewrite might
  undo. `go doc ./...` gives the role of every package.
- A rule the specification imposes goes in the comment of the code that
  enforces it, with a reference to the section of `SPEC.md`.
- Design options that were set aside don't go in comments: they live in
  commit messages and PR descriptions.

## Website

`docs/` is the website, plain static HTML with no build step, published
to <https://emlang-project.github.io/> by `.github/workflows/docs.yml`
on each `v*` tag, or by hand (`gh workflow run docs.yml`).

- The workflow adds `emlang.wasm` and `wasm_exec.js`, built from the same
  Go version. Never commit either one under `docs/`.
- The `emlang-project.github.io` repository only holds the published
  site: each publication replaces its content. Never edit it.
- Preview the site locally as described in `CONTRIBUTING.md`.

## Versioning context

emlang follows semantic versioning since v1.0.0. A change to the CLI's
output, flags, defaults, or config shape is a user-visible change: it is
listed in the release notes, with what a user has to change to keep the
previous behavior.
