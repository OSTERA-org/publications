# pubcheck

pubcheck validates and maintains the OSTERA Publications repository. The CI
runs it on every pull request and every push to main.

## Layout

| File | Contents |
| --- | --- |
| `main.go` | The command: reads the command line and calls the package |
| `pub/` | The package that does the work: `pub.Update`, `pub.Validate`, `pub.Generate`, `pub.WriteChecksums`, `pub.Publish`, `pub.AddWorkingGroup`, `pub.AllocateNumber`, and `pub.FindRoot`, with the tests |
| `go.mod`, `go.sum` | The Go module, `github.com/OSTERA-org/publications/tools/pubcheck` |

## Commands

The usual way to run pubcheck is through the Makefile in the repository
root, from the repository root. [README-Makefile.md](../../README-Makefile.md)
explains each target:

```
make update
make validate
make test
make generate
make checksums DIR=docs/jcs/v1.0
make publish ID=jcs
make add-wg ID=jss
make reserve ID=jcs
make assign ID=jcs
```

The Makefile runs `go run -C tools/pubcheck . <command>`. From this folder,
`go run main.go <command>` does the same. Directories given to `checksums`
are always relative to the repository root.

The commands are:

* **update** deletes the files the operating system leaves in folders
  (`.DS_Store`, `._<name>`, `Thumbs.db`, and `desktop.ini`), writes
  `SHA512SUMS` in every version and corrections folder where it is
  missing or does not match the files, and then does what `generate` does.
  The publishing procedures use only this command and `validate`.
* **checksums DIR...** writes `SHA512SUMS` for each directory (a version,
  or a corrections folder), sorted by filename in byte
  order.
* **generate** writes the landing page (`index.md`) of each document and
  each version, `docs/catalog.json`, and the generated sections of
  `docs/index.md` and `docs/about.md`. Never edit those by hand. It first
  checks `registry.yaml` and every `metadata.yaml`, and writes nothing if
  any of them is wrong.
* **validate** checks `registry.yaml`, every `metadata.yaml`, every
  published file and checksum, and every generated file.
* **validate -base REF** also checks that nothing published since REF was changed, removed, or renamed. It compares REF with the
  working tree, so it catches problems before you commit.
* **validate -base REF -force** (`make validate FORCE=1`) shows those
  history problems as warnings instead of errors, for a deliberate
  exception. Every other check still applies. It is refused inside GitHub
  Actions, so the Validate check on the pull request still fails, and the
  change is merged by someone on the bypass list of the main branch
  ruleset.

* **publish [SHORTNAME | URL]** publishes an approved draft from its
  Working Group site, asking for the publication date and the branch
  name, and commits only the files of the publication after asking. It
  never pushes.
* **add-wg [ID]** adds a Working Group to `registry.yaml`, suggesting its
  name from the list of Working Groups on wg.ostera.org.
* **reserve [SHORTNAME | URL]** and **assign [SHORTNAME | URL]** add the
  next free document number for a work item, as `reserved` or
  `assigned`, suggesting its type and today's date. When its Working
  Group is not in `registry.yaml`, they offer to add it too. **assign**
  also changes a reserved number to assigned.

Given a short name, such as `jcs`, the commands look for the work item on
the site of every Working Group at once (those listed on wg.ostera.org,
and those in `registry.yaml`). Without one, they ask for it.

**add-wg**, **reserve**, and **assign** edit `registry.yaml` line by line,
so its comments are kept, read it back to check the change, and run
**update** and **validate**. They never use git; they print the git steps.

pubcheck ignores the operating system files listed under **update**
wherever it reads a folder, so they never cause a problem.

The generated pages never depend on today's date, so `validate` gives the
same answer on any day.

## License

pubcheck is licensed under the Apache License, Version 2.0. See
[LICENSE](LICENSE). This license covers the tool only, not the published
documents, which are covered by the repository's LICENSE.md.
