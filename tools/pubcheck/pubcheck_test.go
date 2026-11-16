// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 OSTERA

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const testRegistry = `schema_version: 1
ranges:
  organizational:
    first: 1
    last: 999
    types: [PD]
  technical:
    first: 1000
    types: [STD, TS, SUP, TN, TP, TR]
types:
  PD:  { name: Policy/Process Document, track: organizational }
  STD: { name: Standard,                track: standards }
  TS:  { name: Technical Specification, track: standards }
  SUP: { name: Supplement,              track: parent }
  TN:  { name: Technical Note,          track: informative }
  TP:  { name: Technical Paper,         track: informative }
  TR:  { name: Technical Report,        track: informative }
working_groups:
  jss:
    name: JSON Signature Scheme (JSS)
    state: active
numbers:
  - number: 1000
    shortname: jcs
    state: active
    assigned: 2026-11-04
`

const testMetadata = `schema_version: 1
number: 1000
shortname: jcs
title: JSON Canonicalization Scheme (JCS)
working_group: jss
authoritative_language: en
authoritative_format: pdf
keywords:
  - Canonicalization
  - JSON
summary: >
  Defines a canonical representation of JSON data.
canonical_url: https://pub.ostera.org/jcs/
obsoletes:
  - id: IETF RFC 8785
    url: https://www.rfc-editor.org/info/rfc8785
supplements: null
supplemented_by: []
related: []
implementations:
  - language: Go
    url: https://github.com/gowebpki/jcs
versions:
  - version: "1.0"
    type: TS
    status: published
    published: 2026-11-24
    developed_by: working_group
    languages: [en]
    formats: [pdf, md]
    authors:
      - name: Bret Jordan
        affiliation: Individual
    editors:
      - name: Bret Jordan
        affiliation: Individual
    incorporates:
      - IETF RFC 8785 Errata ID 6292
    corrections: []
`

// fixture builds a complete, valid repository in a temporary directory.
func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, "registry.yaml", testRegistry)
	write(t, root, "docs/index.md", "---\nlayout: home\n---\n\nIntroduction.\n\n"+beginMarker("documents")+"\n"+endMarker("documents")+"\n")
	write(t, root, "docs/about.md", "---\nlayout: page\n---\n\n## Document Types\n\n"+beginMarker("document-types")+"\n"+endMarker("document-types")+"\n")
	write(t, root, "docs/reviews/index.md", "---\nlayout: page\n---\n\n"+beginMarker("reviews")+"\n"+endMarker("reviews")+"\n")
	write(t, root, "docs/_config.yml", "title: OSTERA Publications\n")
	write(t, root, "docs/CNAME", "pub.ostera.org\n")
	write(t, root, "docs/404.md", "---\nlayout: page\n---\n")
	write(t, root, "docs/jcs/metadata.yaml", testMetadata)
	write(t, root, "docs/jcs/v1.0/OSTERA-TS-1000-2026-JCS-v1.0-en.pdf", "%PDF fake\n")
	write(t, root, "docs/jcs/v1.0/OSTERA-TS-1000-2026-JCS-v1.0-en.md", "# JCS\n")
	mustSums(t, root, "docs/jcs/v1.0")
	mustGenerate(t, root)
	return root
}

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func edit(t *testing.T, root, rel, old, new string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), old) {
		t.Fatalf("%s does not contain %q", rel, old)
	}
	write(t, root, rel, strings.Replace(string(data), old, new, 1))
}

func mustSums(t *testing.T, root, rel string) {
	t.Helper()
	if err := writeSums(filepath.Join(root, rel)); err != nil {
		t.Fatal(err)
	}
}

func mustGenerate(t *testing.T, root string) {
	t.Helper()
	p, err := generate(root)
	if err != nil {
		t.Fatal(err)
	}
	if p.Len() > 0 {
		t.Fatalf("generate reported problems: %+v", p.list)
	}
}

// expectProblem fails unless some problem message contains want.
func expectProblem(t *testing.T, p *Problems, want string) {
	t.Helper()
	for _, x := range p.list {
		if strings.Contains(x.Path+": "+x.Message, want) {
			return
		}
	}
	t.Fatalf("expected a problem containing %q; got %+v", want, p.list)
}

func TestValidFixture(t *testing.T) {
	root := fixture(t)
	if p := validate(root); p.Len() != 0 {
		t.Fatalf("expected no problems, got %+v", p.list)
	}
}

func TestFilenameGrammar(t *testing.T) {
	good := []string{
		"OSTERA-TS-1000-2026-JCS-v1.0-en.pdf",
		"OSTERA-TS-1000-2026-JCS-v1.0-ERRATUM-1-en.pdf",
		"OSTERA-TS-1000-2026-JCS-v1.0-CORRIGENDUM-1-en.pdf",
		"OSTERA-PD-0003-2025-WORKINGMETHODS-v1.0-nl.pdf",
		"OSTERA-TS-1000-2026-JCS-v1.0.1-en.md",
	}
	bad := []string{
		"OSTERA-PD-3-2025-WORKINGMETHODS-v1.0-en.pdf",
		"OSTERA-TS-1000-2026-JCS-v1.0.-en.pdf",
		"OSTERA-TS-1000-2026-JCS-v01.0-en.pdf",
		"OSTERA-TS-1000-2026-jcs-v1.0-en.pdf",
		"OSTERA-TS-1000:2026-JCS-v1.0-en.pdf",
		"OSTERA-TS-1000-2026-JCS-v1.0-ERRATUM-0-en.pdf",
		"OSTERA-TS-1000-2026-JCS-v1.0-EN.pdf",
		"OSTERA_TS_1000_2026_JCS_v1.0_en.pdf", // the old underscore separator
		"OSTERA-TS-1000-2026-JCS-v1.0_en.pdf", // mixed separators
	}
	for _, n := range good {
		if !filenameRE.MatchString(n) {
			t.Errorf("should match: %s", n)
		}
	}
	for _, n := range bad {
		if filenameRE.MatchString(n) {
			t.Errorf("should not match: %s", n)
		}
	}
}

func TestSumsFormat(t *testing.T) {
	root := fixture(t)
	data, err := os.ReadFile(filepath.Join(root, "docs/jcs/v1.0/SHA512SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	var names []string
	for _, l := range lines {
		f := strings.SplitN(l, "  ", 2)
		if len(f) != 2 || len(f[0]) != 128 {
			t.Fatalf("bad line %q", l)
		}
		names = append(names, f[1])
	}
	if !sort.StringsAreSorted(names) {
		t.Fatalf("not sorted: %v", names)
	}
	// Cross check against the system tool when it is available.
	if _, err := exec.LookPath("sha512sum"); err == nil {
		cmd := exec.Command("sha512sum", "-c", "--quiet", sumsFile)
		cmd.Dir = filepath.Join(root, "docs/jcs/v1.0")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("sha512sum -c failed: %v %s", err, out)
		}
	}
}

func TestBrokenRepositories(t *testing.T) {
	const md = "docs/jcs/metadata.yaml"
	cases := []struct {
		name   string
		break_ func(t *testing.T, root string)
		want   string
	}{
		{"unquoted version", func(t *testing.T, r string) { edit(t, r, md, `version: "1.0"`, `version: 1.0`) }, "version must be quoted"},
		{"leading zero", func(t *testing.T, r string) { edit(t, r, "registry.yaml", "number: 1000", "number: 01000") }, "leading zeros"},
		{"unknown field", func(t *testing.T, r string) { edit(t, r, md, "title:", "titel: x\ntitle:") }, "unknown field"},
		{"old approval block", func(t *testing.T, r string) {
			edit(t, r, md, "    developed_by: working_group\n", "    developed_by: working_group\n    approval:\n      body: working_group\n      date: 2026-11-20\n")
		}, "unknown field"},
		{"bad status", func(t *testing.T, r string) { edit(t, r, md, "status: published", "status: final") }, `status "final"`},
		{"type not in range", func(t *testing.T, r string) { edit(t, r, md, "type: TS", "type: PD") }, "not allowed in the technical range"},
		{"missing working group", func(t *testing.T, r string) { edit(t, r, md, "working_group: jss\n", "") }, "working_group is required"},
		{"unknown working group", func(t *testing.T, r string) { edit(t, r, md, "working_group: jss", "working_group: xyz") }, `working_group "xyz" is not defined`},
		{"supplements on a TS", func(t *testing.T, r string) { edit(t, r, md, "supplements: null", "supplements: 1000") }, "supplements must be null"},
		{"canonical url", func(t *testing.T, r string) { edit(t, r, md, "pub.ostera.org/jcs/", "pub.ostera.org/xyz/") }, "canonical_url must be https://pub.ostera.org/jcs/"},
		{"authoritative format not published", func(t *testing.T, r string) {
			edit(t, r, md, "authoritative_format: pdf", "authoritative_format: html")
		}, "formats must include the authoritative format html"},
		{"registry mismatch", func(t *testing.T, r string) { edit(t, r, md, "number: 1000", "number: 1001") }, "does not match registry.yaml"},
		{"extra file", func(t *testing.T, r string) {
			write(t, r, "docs/jcs/v1.0/OSTERA-TS-1000-2026-JCS-v1.0-en.html", "x")
		}, "is not expected from metadata.yaml"},
		{"badly named file", func(t *testing.T, r string) { write(t, r, "docs/jcs/v1.0/jcs.pdf", "x") }, "does not match the filename grammar"},
		{"missing artifact", func(t *testing.T, r string) {
			os.Remove(filepath.Join(r, "docs/jcs/v1.0/OSTERA-TS-1000-2026-JCS-v1.0-en.md"))
		}, "missing OSTERA-TS-1000-2026-JCS-v1.0-en.md"},
		{"tampered artifact", func(t *testing.T, r string) {
			write(t, r, "docs/jcs/v1.0/OSTERA-TS-1000-2026-JCS-v1.0-en.pdf", "%PDF changed\n")
		}, "has the wrong digest"},
		{"unlisted version directory", func(t *testing.T, r string) { write(t, r, "docs/jcs/v2.0/x", "x") }, "not listed in metadata.yaml"},
		{"active without directory", func(t *testing.T, r string) {
			edit(t, r, "registry.yaml", "    assigned: 2026-11-04\n", "    assigned: 2026-11-04\n  - number: 1001\n    shortname: other\n    state: active\n    assigned: 2026-11-05\n")
		}, "is active but docs/other/metadata.yaml does not exist"},
		{"stale landing page", func(t *testing.T, r string) { edit(t, r, md, "title: JSON", "title: The JSON") }, "docs/jcs/index.md: is missing or out of date"},
		{"stale version page", func(t *testing.T, r string) { edit(t, r, md, "title: JSON", "title: The JSON") }, "docs/jcs/v1.0/index.md: is missing or out of date"},
		{"hand edited version page", func(t *testing.T, r string) { edit(t, r, "docs/jcs/v1.0/index.md", "Published", "Final") }, "docs/jcs/v1.0/index.md: is missing or out of date"},
		{"stale catalog", func(t *testing.T, r string) { edit(t, r, md, "title: JSON", "title: The JSON") }, "docs/catalog.json: is missing or out of date"},
		{"reserved short name", func(t *testing.T, r string) {
			edit(t, r, "registry.yaml", "    assigned: 2026-11-04\n", "    assigned: 2026-11-04\n  - number: 1001\n    shortname: about\n    state: assigned\n    assigned: 2026-11-05\n")
		}, `short name "about" is reserved`},
		{"missing home markers", func(t *testing.T, r string) { write(t, r, "docs/index.md", "---\nlayout: home\n---\n") }, "docs/index.md: is missing the markers"},
		{"stray file in a document", func(t *testing.T, r string) { write(t, r, "docs/jcs/README.md", "x") }, "docs/jcs/README.md: does not belong in a document directory"},
		{"folder without metadata", func(t *testing.T, r string) { write(t, r, "docs/assets/style.css", "x") }, "docs/assets: document directory has no metadata.yaml"},
		{"hand edited home section", func(t *testing.T, r string) { edit(t, r, "docs/index.md", "| [JSON Canonicalization", "| [Edited") }, "the generated documents section is out of date"},
		{"hand edited types section", func(t *testing.T, r string) { edit(t, r, "docs/about.md", "| Technical Report |", "| Report |") }, "the generated document-types section is out of date"},
		{"corrections without files", func(t *testing.T, r string) {
			edit(t, r, md, "    corrections: []", "    corrections:\n      - kind: erratum\n        sequence: 1\n        date: 2027-01-15\n        languages: [en]\n        formats: [pdf]")
		}, "corrections/ is missing"},
		{"correction sequence gap", func(t *testing.T, r string) {
			edit(t, r, md, "    corrections: []", "    corrections:\n      - kind: erratum\n        sequence: 2\n        date: 2027-01-15\n        languages: [en]\n        formats: [pdf]")
		}, "expected sequence 1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := fixture(t)
			c.break_(t, root)
			expectProblem(t, validate(root), c.want)
		})
	}
}

func TestCorrectionsAccepted(t *testing.T) {
	root := fixture(t)
	edit(t, root, "docs/jcs/metadata.yaml", "    corrections: []", "    corrections:\n      - kind: erratum\n        sequence: 1\n        date: 2027-01-15\n        languages: [en]\n        formats: [pdf]")
	write(t, root, "docs/jcs/corrections/OSTERA-TS-1000-2026-JCS-v1.0-ERRATUM-1-en.pdf", "%PDF erratum\n")
	mustSums(t, root, "docs/jcs/corrections")
	mustGenerate(t, root)
	if p := validate(root); p.Len() != 0 {
		t.Fatalf("expected no problems, got %+v", p.list)
	}
}

// gitFixture is a valid fixture committed to a git repository, so the
// immutability checks have a base to compare against.
func gitFixture(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := fixture(t)
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"-c", "user.name=t", "-c", "user.email=t@example.org", "add", "-A"},
		{"-c", "user.name=t", "-c", "user.email=t@example.org", "commit", "-q", "-m", "base"},
	} {
		if _, err := git(root, args...); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestHistory(t *testing.T) {
	const md = "docs/jcs/metadata.yaml"
	const pdf = "docs/jcs/v1.0/OSTERA-TS-1000-2026-JCS-v1.0-en.pdf"
	cases := []struct {
		name   string
		break_ func(t *testing.T, root string)
		want   string
	}{
		{"modified published file", func(t *testing.T, r string) { write(t, r, pdf, "%PDF changed\n"); mustSums(t, r, "docs/jcs/v1.0") }, "cannot be changed or deleted"},
		{"deleted published file", func(t *testing.T, r string) { os.Remove(filepath.Join(r, pdf)) }, "cannot be changed or deleted"},
		{"added to published version", func(t *testing.T, r string) { write(t, r, "docs/jcs/v1.0/extra.txt", "x") }, "already published"},
		{"renamed number", func(t *testing.T, r string) { edit(t, r, "registry.yaml", "shortname: jcs", "shortname: jcs2") }, "changed its short name"},
		{"edited version entry", func(t *testing.T, r string) { edit(t, r, md, "published: 2026-11-24", "published: 2026-11-25") }, "may only change its status"},
		{"status backwards", func(t *testing.T, r string) {
			edit(t, r, md, "status: published", "status: superseded")
			if _, err := git(r, "-c", "user.name=t", "-c", "user.email=t@example.org", "commit", "-qam", "s"); err != nil {
				t.Fatal(err)
			}
			edit(t, r, md, "status: superseded", "status: published")
		}, "cannot change status from superseded to published"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := gitFixture(t)
			c.break_(t, root)
			p := &Problems{}
			base := "main"
			if c.name == "status backwards" {
				base = "HEAD"
			}
			if err := checkHistory(root, base, p); err != nil {
				t.Fatal(err)
			}
			expectProblem(t, p, c.want)
		})
	}

	t.Run("allowed changes", func(t *testing.T) {
		root := gitFixture(t)
		// Supersede version 1.0 and publish version 1.1.
		edit(t, root, md, "status: published", "status: superseded")
		data, _ := os.ReadFile(filepath.Join(root, md))
		v11 := strings.Replace(string(data)[strings.Index(string(data), "  - version: \"1.0\""):], `"1.0"`, `"1.1"`, 1)
		v11 = strings.Replace(v11, "status: superseded", "status: published", 1)
		v11 = strings.Replace(v11, "published: 2026-11-24", "published: 2027-03-01", 1)
		write(t, root, md, string(data)+v11)
		write(t, root, "docs/jcs/v1.1/OSTERA-TS-1000-2027-JCS-v1.1-en.pdf", "%PDF 1.1\n")
		write(t, root, "docs/jcs/v1.1/OSTERA-TS-1000-2027-JCS-v1.1-en.md", "# JCS 1.1\n")
		mustSums(t, root, "docs/jcs/v1.1")
		mustGenerate(t, root)
		page, _ := os.ReadFile(filepath.Join(root, "docs/jcs/v1.0/index.md"))
		if !strings.Contains(string(page), "superseded by [Version 1.1](/jcs/v1.1/)") {
			t.Fatalf("version 1.0 page does not show the superseded notice:\n%s", page)
		}
		p := validate(root)
		if err := checkHistory(root, "main", p); err != nil {
			t.Fatal(err)
		}
		if p.Len() != 0 {
			t.Fatalf("expected no problems, got %+v", p.list)
		}
	})
}

// TestSchemaEnums keeps the Go enums and the JSON Schemas in step, so the
// schemas used by editors and agents never disagree with the validator.
// pdfOnlyBase commits a repository where version 1.0 was published as a
// PDF only, so the tests can add the Markdown format afterward.
func pdfOnlyBase(t *testing.T) string {
	t.Helper()
	root := gitFixture(t)
	const md = "docs/jcs/metadata.yaml"
	edit(t, root, md, "formats: [pdf, md]", "formats: [pdf]")
	os.Remove(filepath.Join(root, "docs/jcs/v1.0/OSTERA-TS-1000-2026-JCS-v1.0-en.md"))
	mustSums(t, root, "docs/jcs/v1.0")
	mustGenerate(t, root)
	if p := validate(root); p.Len() != 0 {
		t.Fatalf("PDF only base is invalid: %+v", p.list)
	}
	if _, err := git(root, "-c", "user.name=t", "-c", "user.email=t@example.org", "commit", "-qam", "pdf only"); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestAddFormatLater(t *testing.T) {
	const md = "docs/jcs/metadata.yaml"
	const mdFile = "docs/jcs/v1.0/OSTERA-TS-1000-2026-JCS-v1.0-en.md"

	t.Run("appending a format is allowed", func(t *testing.T) {
		root := pdfOnlyBase(t)
		edit(t, root, md, "formats: [pdf]", "formats: [pdf, md]")
		write(t, root, mdFile, "# JCS\n")
		mustSums(t, root, "docs/jcs/v1.0")
		mustGenerate(t, root)
		p := validate(root)
		if err := checkHistory(root, "HEAD", p); err != nil {
			t.Fatal(err)
		}
		if p.Len() != 0 {
			t.Fatalf("expected no problems, got %+v", p.list)
		}
	})

	t.Run("a file without a new format is rejected", func(t *testing.T) {
		root := pdfOnlyBase(t)
		write(t, root, mdFile, "# JCS\n")
		mustSums(t, root, "docs/jcs/v1.0")
		p := &Problems{}
		if err := checkHistory(root, "HEAD", p); err != nil {
			t.Fatal(err)
		}
		expectProblem(t, p, "only the files for a format newly appended")
	})

	t.Run("the published file still cannot change", func(t *testing.T) {
		root := pdfOnlyBase(t)
		edit(t, root, md, "formats: [pdf]", "formats: [pdf, md]")
		write(t, root, mdFile, "# JCS\n")
		write(t, root, "docs/jcs/v1.0/OSTERA-TS-1000-2026-JCS-v1.0-en.pdf", "%PDF changed\n")
		mustSums(t, root, "docs/jcs/v1.0")
		p := &Problems{}
		if err := checkHistory(root, "HEAD", p); err != nil {
			t.Fatal(err)
		}
		expectProblem(t, p, "an existing checksum line was changed or removed")
		expectProblem(t, p, "en.pdf: is in a published version directory and cannot be changed")
	})

	t.Run("reordering formats is rejected", func(t *testing.T) {
		root := gitFixture(t)
		edit(t, root, md, "formats: [pdf, md]", "formats: [md, pdf]")
		p := &Problems{}
		if err := checkHistory(root, "main", p); err != nil {
			t.Fatal(err)
		}
		expectProblem(t, p, "formats may only be appended")
	})
}

func TestSchemaEnums(t *testing.T) {
	load := func(name string) map[string]any {
		data, err := os.ReadFile(filepath.Join("..", "..", "schemas", name))
		if err != nil {
			t.Fatal(err)
		}
		var v map[string]any
		if err := json.Unmarshal(data, &v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	reg, meta := load("registry.schema.json"), load("metadata.schema.json")
	check := func(name string, schema map[string]any, path []string, want []string) {
		var cur any = schema
		for _, key := range path {
			cur = cur.(map[string]any)[key]
		}
		var got []string
		for _, x := range cur.([]any) {
			got = append(got, x.(string))
		}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%s %v: schema has %v, Go has %v", name, path, got, want)
		}
	}
	check("registry", reg, []string{"properties", "types", "additionalProperties", "properties", "track", "enum"}, tracks)
	check("registry", reg, []string{"properties", "working_groups", "additionalProperties", "properties", "state", "enum"}, wgStates)
	check("registry", reg, []string{"properties", "numbers", "items", "properties", "state", "enum"}, numberStates)
	short := reg["properties"].(map[string]any)["numbers"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)["shortname"].(map[string]any)
	reserved := short["allOf"].([]any)[1].(map[string]any)["not"].(map[string]any)
	check("registry", reserved, []string{"enum"}, reservedShortnames)
	v := []string{"$defs", "version", "properties"}
	check("metadata", meta, append(v, "status", "enum"), versionStatuses)
	check("metadata", meta, append(v, "developed_by", "enum"), developedBys)
	check("metadata", meta, append(v, "corrections", "items", "properties", "kind", "enum"), correctionKinds)
}

func TestGenerateChecksMetadataFirst(t *testing.T) {
	t.Run("empty metadata.yaml", func(t *testing.T) {
		root := fixture(t)
		write(t, root, "docs/jcs/metadata.yaml", "")
		p, err := generate(root)
		if err != nil {
			t.Fatal(err)
		}
		expectProblem(t, p, "docs/jcs/metadata.yaml")
	})
	t.Run("no versions", func(t *testing.T) {
		root := fixture(t)
		before, _ := os.ReadFile(filepath.Join(root, "docs/index.md"))
		data, _ := os.ReadFile(filepath.Join(root, "docs/jcs/metadata.yaml"))
		s := string(data)
		write(t, root, "docs/jcs/metadata.yaml", s[:strings.Index(s, "versions:")]+"versions: []\n")
		p, err := generate(root)
		if err != nil {
			t.Fatal(err)
		}
		expectProblem(t, p, "versions must list at least one published version")
		after, _ := os.ReadFile(filepath.Join(root, "docs/index.md"))
		if string(before) != string(after) {
			t.Fatal("generate wrote files although metadata.yaml was wrong")
		}
	})
	t.Run("files on disk are not needed", func(t *testing.T) {
		root := fixture(t)
		os.Remove(filepath.Join(root, "docs/jcs/index.md"))
		os.Remove(filepath.Join(root, "docs/jcs/v1.0/index.md"))
		mustGenerate(t, root)
		if p := validate(root); p.Len() != 0 {
			t.Fatalf("expected no problems, got %+v", p.list)
		}
	})
}
