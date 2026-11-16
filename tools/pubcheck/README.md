# pubcheck

pubcheck validates and maintains the OSTERA Publications repository. The CI
runs it on every pull request and every push to main.

## Commands

Run these from the repository root. Directories are always relative to the
repository root.

```
go run -C tools/pubcheck . checksums docs/jcs/v1.0
go run -C tools/pubcheck . generate
go run -C tools/pubcheck . validate
go run -C tools/pubcheck . validate -base origin/main
```

* **checksums DIR...** writes `SHA512SUMS` for each directory, sorted by
  filename in byte order.
* **generate** writes the landing page (`index.md`) of each document and
  each version, `docs/catalog.json`, and the generated sections of
  `docs/index.md` and `docs/about.md`. Never edit those by hand.
* **validate** checks `registry.yaml`, every `metadata.yaml`, every
  published file and checksum, and every generated file.
* **validate -base REF** also checks that nothing published since REF was
  changed, removed, or renamed. It compares REF with the working tree, so
  it catches problems before you commit.

The rules these commands enforce are described in [AGENTS.md](../../AGENTS.md).

## License

pubcheck is licensed under the Apache License, Version 2.0. See
[LICENSE](LICENSE). This license covers the tool only, not the published
documents, which are covered by the repository's LICENSE.md.
