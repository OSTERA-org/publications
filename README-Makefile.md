# The Makefile

The `Makefile` in the root of this repository is the one way to run pubcheck, the tool that builds the site pages and checks the repository. Run every command from the root of the repository (the `publications` folder). Typing `make` on its own lists the commands.

For everyday publishing you need only two commands, always in this order:

```
make update
make validate
```

`make update` brings the repository up to date after you have changed it, and `make validate` checks that the result is correct and allowed. The other commands are described below for completeness.

## What You Need

* **Git and Make.** On a Mac, both come with the Xcode command line tools (`xcode-select --install`).
* **Go**, version 1.24 or later (`brew install go`). The first command you run takes a moment while Go builds pubcheck.
* **A copy of the repository made with `git clone`**, so that `make validate` can find the repository on GitHub.

## make update

**What it does.** It makes the repository consistent after a change, in two steps:

1. **Checksums.** It looks in every version folder (such as `docs/jcs/v1.0/`), every corrections folder (`docs/jcs/corrections/`), and every public review folder (such as `docs/reviews/jcs/v1.0/01/`). Where the `SHA512SUMS` file is missing, or does not match the files in the folder, it writes it again. Folders that are already correct are left alone, so running it twice changes nothing the second time.
2. **Pages.** It writes everything the site shows, from `registry.yaml`, each `metadata.yaml`, and each `review.yaml`:
   * the page of each document, each version, and each public review
   * the review history page of each work item
   * `docs/catalog.json`, the list of published documents
   * `docs/reviews/reviews.json`, the list of public reviews, which the Working Group sites read
   * the generated sections of `docs/index.md`, `docs/about.md`, and `docs/reviews/index.md`

Before writing any pages, it checks the YAML files. If one is wrong, it reports the problem and writes no pages. If a file in a folder is named wrongly, it reports that file and writes no checksums for that folder.

**When to use it.** After every change you make, before `make validate`. Adding files, editing a YAML file, or both: the same command covers them all.

**Why.** The checksums and the pages are derived from the files you add and the YAML you edit. Writing them by hand would be slow and easy to get wrong, and they must match exactly, or the Validate check on GitHub fails.

**What it does not do.** It never decides whether a change is allowed. If a published file was changed by mistake, `make update` records its new checksum like any other, and `make validate` then reports the changed file. Always run `make validate` after it.

## make validate

**What it does.** It checks the repository and changes nothing. It prints `repository is valid`, or a list of every problem it found, each with the file it is in. It works in two parts.

First, it checks the repository as it is now:

* the YAML files are complete and consistent: numbers are unique, every document type and Working Group exists, dates and versions are well formed, and folder names match what the YAML says
* each folder holds exactly the files its YAML lists, named by the naming rules, and nothing else
* each `SHA512SUMS` matches the files beside it
* the generated pages are current, meaning `make update` would not change them
* each published version had at least one public review of 30 days or more, when its reviews are recorded here

Second, it compares your copy with the `main` branch on GitHub:

* no published file or checksum was changed or removed
* no document number was removed or renamed
* published versions were changed only in the ways the procedures allow
* no public review was removed, its draft files were not changed, the facts that identify the draft (short name, draft number, type, version, and Working Group) were not changed, and formats were only added, never removed

To make that comparison, it looks through `git remote -v` for the remote whose address is `github.com/OSTERA-org/publications`, whatever that remote is called, fetches it, and compares against its `main` branch. It prints the branch it compares against.

**When to use it.** After `make update`, and again after fixing anything it reports, until it prints `repository is valid`. Then commit and open the pull request.

**Why.** The Validate check that GitHub runs on every pull request uses the same command. If it passes on your computer, it passes on GitHub, so you find problems before anyone else sees them.

**Options.** You should rarely need these:

| Command | Use |
| --- | --- |
| `make validate REMOTE=<name>` | Use the git remote called `<name>` instead of finding it automatically. |
| `make validate BASE=<branch or commit>` | Compare against something other than `main` on GitHub. |
| `make validate BASE=` | Skip the comparison and only check the repository as it is. |

If no remote points to the repository, it stops and prints the command that adds one.

## make test

**What it does.** It runs pubcheck's own tests.

**When to use it.** Only after changing pubcheck itself, in `tools/pubcheck/`. The publishing procedures never need it. GitHub runs it on every pull request as part of the Validate check.

## make generate

**What it does.** It is the second step of `make update` on its own: it writes the pages and lists, but not the checksums.

**When to use it.** Normally never; use `make update`. It is kept for working on pubcheck itself.

## make checksums DIR=folder

**What it does.** It is the first step of `make update` for a single folder: it writes `SHA512SUMS` in that folder, for example `make checksums DIR=docs/jcs/v1.0`, whether or not it was already correct.

**When to use it.** Normally never; use `make update`. It is kept for working on pubcheck itself.

## make help

**What it does.** It lists the commands with a one line description of each. Typing `make` on its own does the same.

## How It Works

Each command runs `go run -C tools/pubcheck . <command>`, so Go builds pubcheck from the code in this repository whenever it has changed. Nothing needs to be installed apart from Git, Make, and Go. The code and its own documentation are in [tools/pubcheck](tools/pubcheck/README.md).
