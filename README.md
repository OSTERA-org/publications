# OSTERA Publications

This repository holds the approved documents published by OSTERA, the Open Standards and Technology Engineering and Research Association. They are served at https://pub.ostera.org/.

This repository is not used to discuss, develop, or write documents. Documents are developed in the [OSTERA Working Groups](https://wg.ostera.org/). See [CONTRIBUTING.md](CONTRIBUTING.md) for how to take part, comment during a public review, or report an error.

## Layout

| Path | Contents |
| --- | --- |
| `docs/` | The site, built by GitHub Pages and served at https://pub.ostera.org/ |
| `docs/<id>/` | One published document, such as `docs/jcs/`, served at `https://pub.ostera.org/<id>/` |
| `docs/<id>/metadata.yaml` | Everything about the document: title, Working Group, versions, and files |
| `docs/<id>/v<version>/` | One published version, such as `docs/jcs/v1.0/`, with its files and `SHA512SUMS` |
| `docs/<id>/corrections/` | Errata and corrigenda, when there are any |
| `docs/catalog.json` | A machine readable list of every publication, served at https://pub.ostera.org/catalog.json |
| `docs/reviews/` | Public reviews, served at https://pub.ostera.org/reviews/ |
| `docs/reviews/reviews.json` | A machine readable list of every public review, which the Working Group sites read to show the reviews of each work item |
| `docs/reviews/<id>/v<version>/<draft>/` | One public review, such as `docs/reviews/jcs/v1.0/02/` for `draft-ts-jcs-02` of version 1.0, with its `review.yaml`, the draft, and `SHA512SUMS` |
| `docs/index.md`, `docs/about.md`, and `docs/404.md` | The home, About These Documents, and not found pages |
| `docs/_config.yml` | The site configuration |
| `docs/CNAME` | The custom domain (pub.ostera.org). Do not delete it. |
| `registry.yaml` | Every document number ever assigned, plus the document types and Working Groups |
| `schemas/` | JSON Schemas for `registry.yaml`, `metadata.yaml`, and `review.yaml` |
| `tools/pubcheck/` | The tool that generates the site pages and validates the repository |
| `Makefile` | The commands for running pubcheck, explained in [README-Makefile.md](README-Makefile.md) |
| `README-Makefile.md` | What each `make` command does, and when and why to use it |
| `AGENTS.md` | The rules for automated agents working in this repository |

The look and feel comes from the shared [OSTERA Pages Theme](https://github.com/OSTERA-org/ostera-pages-theme/).

## How Publishing Works

`registry.yaml`, each `metadata.yaml`, and each `review.yaml` are the only places where facts about a document or a public review are recorded. `make update` writes everything the site shows from them: the page of each document, each version, and each public review, `catalog.json`, `reviews.json`, the list of documents on the home page, the table of document types on the About page, and the list of public reviews. Those generated files are never edited by hand.

Every change is made through a pull request, and the Validate check runs `pubcheck` on it. A published version never changes: corrections are published alongside it, and later releases get a new version. The draft in an announced public review never changes either. Its dates, title, summary, and description of changes can be corrected, the draft can be added in another format, and the review can be withdrawn or linked to the response to its comments, but a revised draft gets a new review.

Public reviews are the one place drafts appear here. They are listed only under https://pub.ostera.org/reviews/, never with the published documents or in `catalog.json`.

## Intellectual Property and License

See [IPR.md](IPR.md) and [LICENSE.md](LICENSE.md).
