# pubcheck

pubcheck validates and maintains the OSTERA Publications repository. The CI
runs it on every pull request and every push to main.

## Layout

| File | Contents |
| --- | --- |
| `main.go` | The command: reads the command line and calls the package |
| `pub/` | The package that does the work: `pub.Validate`, `pub.Generate`, `pub.WriteChecksums`, and `pub.FindRoot`, with the tests |
| `go.mod`, `go.sum` | The Go module, `github.com/OSTERA-org/publications/tools/pubcheck` |

## Commands

The usual way to run pubcheck is through the Makefile in the repository
root, from the repository root:

```
make generate
make checksums DIR=docs/jcs/v1.0
make checksums DIR=docs/reviews/jcs/v1.0/02
make validate
make test
```

`make validate` finds the git remote whose address is
github.com/OSTERA-org/publications, whatever it is called, fetches it, and
compares against its `main` branch. Use `make validate REMOTE=<name>` to
choose the remote yourself, or
`make validate BASE=` to skip the comparison, or `BASE=<ref>` to compare
against something else.

The Makefile runs `go run -C tools/pubcheck . <command>`. From this folder,
`go run main.go <command>` does the same. Directories given to `checksums`
are always relative to the repository root.

The commands are:

* **checksums DIR...** writes `SHA512SUMS` for each directory (a version,
  a corrections folder, or a public review), sorted by filename in byte
  order.
* **generate** writes the landing page (`index.md`) of each document, each
  version, each public review, and each review history, `docs/catalog.json`,
  and the generated sections of `docs/index.md`, `docs/about.md`, and
  `docs/reviews/index.md`. Never edit those by hand. It first checks
  `registry.yaml`, every `metadata.yaml`, and every `review.yaml`, and
  writes nothing if any of them is wrong.
* **validate** checks `registry.yaml`, every `metadata.yaml`, every
  `review.yaml`, every published file, review draft, and checksum, and
  every generated file.
* **validate -base REF** also checks that nothing published or announced
  since REF was changed, removed, or renamed. It compares REF with the
  working tree, so it catches problems before you commit.

The generated pages never depend on today's date, so `validate` gives the
same answer on any day. Whether a public review is upcoming, open, or
closed is worked out in the reader's browser by `assets/js/ostera.js` in
the [OSTERA Pages Theme](https://github.com/OSTERA-org/ostera-pages-theme).

The rules these commands enforce are described in [AGENTS.md](../../AGENTS.md).

## License

pubcheck is licensed under the Apache License, Version 2.0. See
[LICENSE](LICENSE). This license covers the tool only, not the published
documents, which are covered by the repository's LICENSE.md.
