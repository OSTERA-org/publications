# The Makefile

The `Makefile` in the root of this repository is the one way to run pubcheck, the tool that builds the site pages and checks the repository. Run every command from the root of the repository (the `publications` folder). Typing `make` on its own lists the commands.

For everyday publishing you need only two commands, always in this order:

```
make update
make validate
```

`make update` brings the repository up to date after you have changed it, and `make validate` checks that the result is correct and allowed.

The everyday changes to `registry.yaml` have their own commands, so it never needs to be edited by hand: `make add-wg` adds a Working Group, and `make reserve` and `make assign` add a document number. `make publish` publishes a document. Each of them needs only a short name, such as `jcs`, or a Working Group id, such as `jss`. It looks up everything else on wg.ostera.org, and asks its questions with a suggested answer in brackets (press Enter to accept it). It asks before it changes anything, and runs `make update` and `make validate` for you.

Give the short name with `ID=`, such as `make assign ID=jcs`, or leave it out, and the command asks for it. The other commands are described below for completeness.

## What You Need

* **Git and Make.** On a Mac, both come with the Xcode command line tools (`xcode-select --install`).
* **Go**, version 1.24 or later (`brew install go`). The first command you run takes a moment while Go builds pubcheck.
* **A copy of the repository made with `git clone`**, so that `make validate` can find the repository on GitHub.

## make update

**What it does.** It makes the repository consistent after a change, in three steps:

1. **Clutter.** It deletes the files the operating system leaves in folders: `.DS_Store` and `._<name>` (macOS), and `Thumbs.db` and `desktop.ini` (Windows). Git already ignores them, and pubcheck never reports them, but deleting them keeps the folders clean.
2. **Checksums.** It looks in every version folder (such as `docs/jcs/v1.0/`) and every corrections folder (`docs/jcs/corrections/`). Where the `SHA512SUMS` file is missing, or does not match the files in the folder, it writes it again. Folders that are already correct are left alone, so running it twice changes nothing the second time.
3. **Pages.** It writes everything the site shows, from `registry.yaml` and each `metadata.yaml`:
   * the page of each document and each version
   * `docs/catalog.json`, the list of published documents
   * the generated sections of `docs/index.md` and `docs/about.md`

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

Second, it compares your copy with the `main` branch on GitHub:

* no published file or checksum was changed or removed
* no document number was removed or renamed
* published versions were changed only in the ways the procedures allow
* number states only moved forward (reserved, assigned, active, or a reserved or assigned number retired)

To make that comparison, it looks through `git remote -v` for the remote whose address is `github.com/OSTERA-org/publications`, whatever that remote is called, fetches it, and compares against its `main` branch. It prints the branch it compares against.

**When to use it.** After `make update`, and again after fixing anything it reports, until it prints `repository is valid`. Then commit and open the pull request.

**Why.** The Validate check that GitHub runs on every pull request uses the same command. If it passes on your computer, it passes on GitHub, so you find problems before anyone else sees them.

**Options.** You should rarely need these:

| Command | Use |
| --- | --- |
| `make validate REMOTE=<name>` | Use the git remote called `<name>` instead of finding it automatically. |
| `make validate BASE=<branch or commit>` | Compare against something other than `main` on GitHub. |
| `make validate BASE=` | Skip the comparison and only check the repository as it is. |
| `make validate FORCE=1` | Show the problems found by the comparison as warnings instead of errors, for a deliberate exception. Every other check still applies. It is refused in GitHub Actions, so the Validate check on the pull request still fails, and the pull request must be merged by an organization admin on the bypass list of the main branch ruleset. |

If no remote points to the repository, it stops and prints the command that adds one.

## make publish ID=short name

**What it does.** It publishes an approved draft from its Working Group site, and asks before it commits anything. In order, it:

1. finds the work item on wg.ostera.org from its short name, reads its `registry.json`, and checks that the draft is ready: it is the latest draft of its version, the version had at least one Working Group Last Call of at least 14 days and at least one Public Review of at least 30 days, each closed and not withdrawn (shorter ones are fine too), the Working Group approved the draft on or after its release, it has a PDF, authors or editors, a citation, and keywords, and its number is reserved or assigned to this work item in `registry.yaml`. If the number is not in `registry.yaml` yet, but it is the next free number in its range, it is added. If anything is missing, it lists it and changes nothing
2. asks for the publication date (the date in the document, which is the last time the Working Group approved it, not the day it appears on the website), suggesting the version's approved date from its registry.yaml
3. asks for the branch name, suggesting `publish-<short name>-v<version>`, and creates the branch
4. downloads the PDF, and checks its SHA-512 sum against the Working Group site. Only the PDF is published
5. writes `docs/<short name>/metadata.yaml` (new, or with the new version added and the version it replaces marked `superseded`) and the number's state in `registry.yaml` (`active`), then does what `make update` does, and then what `make validate` does
6. shows a summary of the publication and the files to commit, and asks whether to commit them. It commits only those files, never anything else in your copy, and never pushes

**When to use it.** To publish a document, as `publish-document.md` in the secretariat repository explains, starting on an up to date `main` branch.

**Options.** If two Working Groups have a work item with the same short name, give its page instead, such as `make publish URL=https://wg.ostera.org/jss/work-items/jcs/`. The same works for `make reserve` and `make assign`.

**Why.** Everything it writes comes from what the Working Group site shows, so nothing is typed twice, and the summary shows exactly what will be published before anything is committed.

## make add-wg ID=id

**What it does.** It adds a Working Group to `working_groups` in `registry.yaml`, as `active`. It asks for:

1. the Working Group id, as in `https://wg.ostera.org/<id>/`, such as `jss`, unless it was given with `ID=`
2. its name, suggesting it from the list of Working Groups on wg.ostera.org (its name and abbreviation, such as `JSON Signature Scheme (JSS)`), or else from the website settings of its repository

It shows the entry, asks before it adds it, runs `make update` and `make validate`, and prints the git steps, suggesting the branch `add-wg-<id>`.

**When to use it.** When a new Working Group is chartered, before its first work item gets a number or is published.

## make reserve ID=short name and make assign ID=short name

**What they do.** They add the next free document number for a work item to `numbers` in `registry.yaml`. `make reserve` adds it as `reserved`, and `make assign` as `assigned`. They find the work item on wg.ostera.org from its short name, such as `jcs`, and then ask for:

1. its type code, suggesting the type on its page, or `TS`. The type decides the range: `PD` is in the organizational range (1 to 999), and every other type in the technical range (1000 and up)
2. the date allocated, suggesting today. It can be earlier, but not in the future
3. when its Working Group is not in `registry.yaml` yet, whether to add it in the same change, suggesting its name

If the work item cannot be found, for example because its page is not on the website yet, they say so and ask the same questions, suggesting `TS`. If its page already gives a different number, they print a note, because the chairs need to change it.

The number is always the next one in the range: the highest number ever allocated in it, including retired numbers, plus one. They show the entry, ask before they add it, run `make update` and `make validate`, and print the git steps, suggesting the branch `reserve-<short name>-number` or `assign-<short name>-number`.

If the work item already has a reserved number, `make assign` offers to change it to `assigned`, and keeps its number and its allocated date.

**When to use them.** `make assign` when a document is ready to publish, or earlier when its chairs ask for the number. `make reserve` when someone needs to know the number early, for example for a grant or a regulator. After `make assign`, tell the chairs the number, so they can put it on the final draft.

## make test

**What it does.** It runs pubcheck's own tests.

**When to use it.** Only after changing pubcheck itself, in `tools/pubcheck/`. The publishing procedures never need it. GitHub runs it on every pull request as part of the Validate check.

## make generate

**What it does.** It is the last step of `make update` on its own: it writes the pages and lists, but does not delete clutter or write checksums.

**When to use it.** Normally never; use `make update`. It is kept for working on pubcheck itself.

## make checksums DIR=folder

**What it does.** It is the checksum step of `make update` for a single folder: it writes `SHA512SUMS` in that folder, for example `make checksums DIR=docs/jcs/v1.0`, whether or not it was already correct.

**When to use it.** Normally never; use `make update`. It is kept for working on pubcheck itself.

## make help

**What it does.** It lists the commands with a one line description of each. Typing `make` on its own does the same.

## How It Works

Each command runs `go run -C tools/pubcheck . <command>`, so Go builds pubcheck from the code in this repository whenever it has changed. Nothing needs to be installed apart from Git, Make, and Go. The code and its own documentation are in [tools/pubcheck](tools/pubcheck/README.md).
