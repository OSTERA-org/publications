---
layout: page
title: About These Documents
permalink: /about/
lead: "How OSTERA documents are identified, versioned, and licensed."
---

## Document Types

Each document has a type. The type code appears in the document identifier and filename (for example `TS` in OSTERA TS 1000). Whether a document is subject to the patent licensing obligations of the IPR and Copyright Policy depends on its type.

<!-- BEGIN GENERATED document-types: pubcheck writes this section from registry.yaml and metadata.yaml; do not edit by hand -->

| Code | Document Type | Patent Licensing Obligations |
| --- | --- | --- |
| STD | Standard | Yes |
| TS | Technical Specification | Yes |
| SUP | Supplement | Same as the document it supplements |
| TN | Technical Note | No |
| TP | Technical Paper | No |
| TR | Technical Report | No |
| PD | Policy/Process Document | Not applicable |

<!-- END GENERATED document-types -->

Errata and corrigenda carry the same obligations as the document they correct.

## Versions

Each document has a permanent address, such as `https://pub.ostera.org/jcs/`, which lists all of its versions and points to the latest one. Each published version also has its own address, such as `https://pub.ostera.org/jcs/v1.0/`.

A published version is final and never changes. Confirmed errors are corrected through an erratum or a corrigendum, published alongside the version it corrects, or through a new version. When a newer version is published, the page of the earlier version says so.

If a version is published in more than one language or format, its page says which file is authoritative.

## Machine Readable Data

Every publication is listed in [catalog.json](/catalog.json), and each document's details are in its `metadata.yaml`, such as `https://pub.ostera.org/jcs/metadata.yaml`. Each version folder has a `SHA512SUMS` file for checking that a downloaded file is intact.

## Public Reviews

OSTERA encourages public comment on each draft before final approval. Public reviews are announced at [ostera.org/public-reviews](https://ostera.org/public-reviews). Comments are submitted through the working group's GitHub repository or by email to [public-review@ostera.org](mailto:public-review@ostera.org).

## Reporting an Error

If you find an error in a published document, please [open an issue](https://github.com/OSTERA-org/Publications/issues) or send an email to [secretariat@ostera.org](mailto:secretariat@ostera.org). Please include the document identifier and version (for example OSTERA TS 1000, Version 1.0), along with the section, and a description of the problem.

## Intellectual Property and License

OSTERA owns the copyright in each published document. The terms under which you may use the documents are in [LICENSE.md](https://github.com/OSTERA-org/Publications/blob/main/LICENSE.md), and a summary of how intellectual property rights apply is in [IPR.md](https://github.com/OSTERA-org/Publications/blob/main/IPR.md). The [OSTERA IPR and Copyright Policy](https://ostera.org/policies/) is the authoritative text.

To disclose patents or other IPR that may cover a document, follow the instructions at [ostera.org/ipr](https://ostera.org/ipr/).
