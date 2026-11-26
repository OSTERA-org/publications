---
layout: page
title: About These Documents
permalink: /about/
lead: "How OSTERA documents are identified, versioned, and licensed."
---

## Document Types

Each document has a type. The type code appears in the document identifier and filename (for example, `TS` in OSTERA TS 1000). Whether a document is subject to the patent licensing obligations of the IPR and Copyright Policy depends on its type.

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

A published version is final and never changes. Confirmed errors are corrected through an erratum or a corrigendum, published alongside the version it corrects, or through a new version. When a newer version is published, the page of the earlier version will say so.

If a version is published in more than one language or format, its page says which file is authoritative.

## Machine Readable Data

Every publication is listed in [catalog.json](/catalog.json), and each document's details are in its `metadata.yaml`, such as `https://pub.ostera.org/jcs/metadata.yaml`. Each version folder has a `SHA512SUMS` file for checking that a downloaded file is intact.

## Last Calls and Public Reviews

OSTERA encourages public comment on each draft before it is published. Each version goes through at least one Working Group Last Call of at least 14 days and at least one Public Review of at least 30 days. They are run by each Working Group on its own site, and every Public Review is listed on the [Public Reviews](https://wg.ostera.org/reviews/) page. To comment, see [How to Participate](https://wg.ostera.org/participate/).

## Reporting an Error in a Published Document

If you find an error in a published document, please either [open an issue](https://github.com/OSTERA-org/publications/issues) in the publications repository on GitHub or send an email to [secretariat@ostera.org](mailto:secretariat@ostera.org). Please include the document identifier and version (for example, OSTERA TS 1000, Version 1.0), the section, and a description of the problem.

<!-- BEGIN STANDARD TEXT -->

## Intellectual Property

Anything you submit to OSTERA, such as an issue, a pull request, a comment, or an email, is a contribution under the [OSTERA IPR and Copyright Policy](https://ostera.org/policies/), and by submitting it you agree to that policy. Do not submit anything you do not have the right to contribute.

If you have knowledge of intellectual property rights, including patents, copyrights, trademarks, and service marks, for software or text, the use of which may be required to implement or publish any of the work items being considered, please disclose it by following the instructions at [ostera.org/ipr/](https://ostera.org/ipr/) or contact the Secretariat at [secretariat@ostera.org](mailto:secretariat@ostera.org).

## Code of Conduct

All participants are expected to follow the OSTERA [Code of Conduct](https://github.com/OSTERA-org-Working-Groups/.github/blob/main/CODE_OF_CONDUCT.md).

## Questions

For general questions about OSTERA, visit [OSTERA.org](https://ostera.org/) or contact the Secretariat at [secretariat@ostera.org](mailto:secretariat@ostera.org).

<!-- END STANDARD TEXT -->
