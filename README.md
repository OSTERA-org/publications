# OSTERA Publications

This repository holds the approved documents published by OSTERA, the Open Standards and Technology Engineering and Research Association. They are served at https://pub.ostera.org/.

This repository is not used to discuss, develop, or write documents. Documents are developed in the [OSTERA Working Groups](https://wg.ostera.org/). See [CONTRIBUTING.md](CONTRIBUTING.md) for how to take part, comment during a public review, or report an error.

## Layout

| Path | Contents |
| --- | --- |
| `docs/<id>/v<version>/` | A published version of a document (for example `docs/jcs/v1.0/`), served at `https://pub.ostera.org/<id>/v<version>/` |
| `docs/<id>/index.html` | The page that lists the published versions of a document |
| `docs/_data/documents.yml` | The list of documents and their versions |
| `docs/_data/working_groups.yml` | The working groups the documents are grouped under |
| `docs/_data/document_types.yml` | The OSTERA document types |
| `docs/_includes/` | Page parts used by this site |
| `docs/index.md`, `docs/about.md`, and `docs/404.md` | The home, About These Documents, and not found pages |
| `docs/CNAME` | The custom domain (pub.ostera.org). Do not delete it. |

The look and feel comes from the shared [OSTERA Pages Theme](https://github.com/OSTERA-org/ostera-pages-theme).

## Publishing a Version

1. Put the rendered document in a new folder, `docs/<id>/v<version>/`, with `index.html` as its main file.
2. Add the version to the top of the document's `versions` list in `docs/_data/documents.yml`.
3. For a new document, also create `docs/<id>/index.html` (copy an existing one and change the `id`).

A published version folder never changes once it is merged. Corrections go in a new version folder or in an erratum.

Files without front matter, such as the rendered HTML of a document, are copied to the site exactly as they are. Jekyll skips files and folders whose names start with `_` or `.`, so do not use such names inside a version folder.

## Intellectual Property and License

See [IPR.md](IPR.md) and [LICENSE.md](LICENSE.md).
