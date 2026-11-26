# OSTERA Publications

This repository holds the approved documents published by OSTERA, The Open Standards and Technology Engineering and Research Association. They are served at [pub.ostera.org/](https://pub.ostera.org/).

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
| `docs/index.md`, `docs/about.md`, and `docs/404.md` | The home, About These Documents, and not found pages |
| `docs/_config.yml` | The site configuration |
| `docs/CNAME` | The custom domain (pub.ostera.org). Do not delete it. |
| `registry.yaml` | Every document number ever allocated, plus the document types and Working Groups |
| `schemas/` | JSON Schemas for `registry.yaml` and `metadata.yaml` |
| `tools/pubcheck/` | The tool that generates the site pages and validates the repository |
| `Makefile` | The commands for running pubcheck, explained in [README-Makefile.md](README-Makefile.md) |
| `README-Makefile.md` | What each `make` command does, and when and why to use it |

The look and feel comes from the shared [OSTERA Pages Theme](https://github.com/OSTERA-org/ostera-pages-theme/).

## How Publishing Works

`registry.yaml` and each `metadata.yaml` are the only places where facts about a document are recorded. `make update` writes everything the site shows from them: the page of each document and each version, `catalog.json`, the list of documents on the home page, and the table of document types on the About page. Those generated files are never edited by hand.

Every change is made through a pull request, and the Validate check runs `pubcheck` on it. A published version never changes: corrections are published alongside it, and later releases get a new version.

Drafts never appear here. Each Working Group keeps its drafts, its Working Group Last Calls, and its Public Reviews on its own site. How to take part, and how to comment during a Public Review, are explained at [wg.ostera.org/participate/](https://wg.ostera.org/participate/).

## Participating

Participation is open to OSTERA members. However, anyone may comment on a draft during its Public Review. See [How to Participate](https://wg.ostera.org/participate/), and read the [Contributing Guidelines](CONTRIBUTING.md) before you open an issue or a pull request.

<!-- BEGIN STANDARD TEXT -->

## Policies

The official OSTERA policies, including the IPR and Copyright Policy, the Working Methods, and the Code of Conduct, are published at [ostera.org/policies/](https://ostera.org/policies/).

## Intellectual Property

Anything you submit to OSTERA, such as an issue, a pull request, a comment, or an email, is a contribution under the [OSTERA IPR and Copyright Policy](https://ostera.org/policies/), and by submitting it you agree to that policy. Do not submit anything you do not have the right to contribute.

If you have knowledge of intellectual property rights, including patents, copyrights, trademarks, and service marks, for software or text, the use of which may be required to implement or publish any of the work items being considered, please disclose it by following the instructions at [ostera.org/ipr/](https://ostera.org/ipr/) or contact the Secretariat at [secretariat@ostera.org](mailto:secretariat@ostera.org).

## Code of Conduct

All participants are expected to follow the OSTERA [Code of Conduct](https://github.com/OSTERA-org-Working-Groups/.github/blob/main/CODE_OF_CONDUCT.md).

## Questions

For general questions about OSTERA, visit [OSTERA.org](https://ostera.org/) or contact the Secretariat at [secretariat@ostera.org](mailto:secretariat@ostera.org).

<!-- END STANDARD TEXT -->

## License

See [LICENSE.md](LICENSE.md).
