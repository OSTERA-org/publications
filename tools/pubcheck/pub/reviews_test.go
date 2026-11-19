// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 OSTERA

package pub

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testReview = `schema_version: 1
shortname: jcs
draft: 1
type: TS
version: "1.0"
title: JSON Canonicalization Scheme (JCS)
working_group: jss
opens: 2026-10-01
closes: 2026-10-31
formats: [pdf]
summary: >
  A canonical form for JSON data.
`

const reviewDir = "docs/reviews/jcs/v1.0/01"

// addReview adds a complete, generated review to root.
func addReview(t *testing.T, root string) {
	t.Helper()
	write(t, root, reviewDir+"/review.yaml", testReview)
	write(t, root, reviewDir+"/draft-ts-jcs-01.pdf", "%PDF draft\n")
	mustSums(t, root, reviewDir)
	mustGenerate(t, root)
}

func commit(t *testing.T, root, msg string) {
	t.Helper()
	for _, args := range [][]string{
		{"add", "-A"},
		{"-c", "user.name=t", "-c", "user.email=t@example.org", "commit", "-q", "-m", msg},
	} {
		if _, err := git(root, args...); err != nil {
			t.Fatal(err)
		}
	}
}

func readText(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestReviewValid(t *testing.T) {
	root := fixture(t)
	addReview(t, root)
	if p := validate(root); p.Len() != 0 {
		t.Fatalf("expected no problems, got %+v", p.list)
	}
	for _, rel := range []string{"docs/reviews/jcs/index.md", reviewDir + "/index.md"} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("%s was not generated", rel)
		}
	}
	list := readText(t, root, "docs/reviews/index.md")
	for _, want := range []string{"[draft-ts-jcs-01](/reviews/jcs/v1.0/01/)", `data-opens="2026-10-01"`} {
		if !strings.Contains(list, want) {
			t.Errorf("reviews/index.md does not contain %q", want)
		}
	}
	page := readText(t, root, reviewDir+"/index.md")
	for _, want := range []string{
		`title: "Public Review of draft-ts-jcs-01"`,
		"[English PDF](/reviews/jcs/v1.0/01/draft-ts-jcs-01.pdf)",
		"https://github.com/OSTERA-org-Working-Groups/jss/issues/new/choose",
		"mailto:public-review@ostera.org?subject=draft-ts-jcs-01%20for%20version%201.0",
		"Published document:** [JSON Canonicalization Scheme (JCS)](/jcs/)",
		`<a href="/jcs/v1.0/">Version 1.0</a>, published 2026-11-24`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("review page does not contain %q", want)
		}
	}
	doc := readText(t, root, "docs/jcs/index.md")
	if !strings.Contains(doc, "## Public Reviews") || !strings.Contains(doc, `class="stages"`) {
		t.Error("document page does not show its public reviews")
	}
	if strings.Contains(readText(t, root, "docs/catalog.json"), "draft-") {
		t.Error("catalog.json lists a review draft")
	}
}

func TestReviewBeforeNumber(t *testing.T) {
	// A work item can be reviewed before it has a number or a document.
	root := fixture(t)
	write(t, root, "docs/reviews/jsf/v1.0/02/review.yaml", strings.NewReplacer("shortname: jcs", "shortname: jsf", "draft: 1", "draft: 2").Replace(testReview))
	write(t, root, "docs/reviews/jsf/v1.0/02/draft-ts-jsf-02.pdf", "%PDF draft\n")
	write(t, root, "docs/reviews/jsf/v1.0/02/draft-ts-jsf-02.md", "# draft\n")
	edit(t, root, "docs/reviews/jsf/v1.0/02/review.yaml", "formats: [pdf]", "formats: [pdf, md]")
	mustSums(t, root, "docs/reviews/jsf/v1.0/02")
	mustGenerate(t, root)
	if p := validate(root); p.Len() != 0 {
		t.Fatalf("expected no problems, got %+v", p.list)
	}
	if !strings.Contains(readText(t, root, "docs/reviews/jsf/index.md"), "<br>Not yet</li>") {
		t.Error("an unpublished work item should show Approved and Published as not yet")
	}
}

func TestReviewBroken(t *testing.T) {
	cases := []struct {
		name   string
		change func(t *testing.T, root string)
		want   string
	}{
		{"closes before opens", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", "closes: 2026-10-31", "closes: 2026-09-30")
		}, "closes must be after opens"},
		{"published without a 30 day review", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", "closes: 2026-10-31", "closes: 2026-10-16")
		}, "version 1.0 needs a public review of at least 30 days (Working Methods, Article 29), and none of the recorded reviews of its drafts lasted that long"},
		{"the 30 day review was withdrawn", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", "formats: [pdf]", "formats: [pdf]\nwithdrawn: 2026-10-10")
		}, "none of the recorded reviews of its drafts lasted that long"},
		{"leading zero draft", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", "draft: 1", "draft: 01")
		}, "numbers must not have leading zeros"},
		{"wrong folder", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", "draft: 1", "draft: 3")
		}, "draft 3 belongs in a folder named 03, not 01"},
		{"wrong version folder", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", `version: "1.0"`, `version: "1.1"`)
		}, "version 1.1 belongs in a folder named v1.1, not v1.0"},
		{"draft directly under the short name", func(t *testing.T, root string) {
			write(t, root, "docs/reviews/jcs/02/review.yaml", testReview)
		}, "does not belong in a review history folder; reviews go in v<version>/<draft>/"},
		{"unknown working group", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", "working_group: jss", "working_group: crgm")
		}, `working_group "crgm" is not defined in registry.yaml`},
		{"unknown type", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", "type: TS", "type: XX")
		}, `type "XX" is not defined in registry.yaml`},
		{"unquoted version", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", `version: "1.0"`, "version: 1.0")
		}, "version must be quoted"},
		{"unknown field", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", "formats: [pdf]", "formats: [pdf]\nnotes: hello")
		}, "cannot parse"},
		{"missing draft file", func(t *testing.T, root string) {
			os.Remove(filepath.Join(root, reviewDir, "draft-ts-jcs-01.pdf"))
		}, "missing draft-ts-jcs-01.pdf"},
		{"misnamed draft file", func(t *testing.T, root string) {
			write(t, root, reviewDir+"/draft-ts-jcs-1.pdf", "x")
		}, "does not match the filename grammar or the draft naming rule"},
		{"wrong checksum", func(t *testing.T, root string) {
			write(t, root, reviewDir+"/draft-ts-jcs-01.pdf", "%PDF changed\n")
		}, "SHA512SUMS"},
		{"no review.yaml", func(t *testing.T, root string) {
			write(t, root, "docs/reviews/jcs/v1.0/02/draft-ts-jcs-02.pdf", "x")
		}, "review directory has no review.yaml"},
		{"stray file", func(t *testing.T, root string) {
			write(t, root, "docs/reviews/notes.md", "x")
		}, "does not belong in docs/reviews"},
		{"stale page", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", "closes: 2026-10-31", "closes: 2026-11-15")
		}, "is missing or out of date; run pubcheck generate"},
		{"other working group than the document", func(t *testing.T, root string) {
			edit(t, root, "registry.yaml", "working_groups:\n", "working_groups:\n  crgm:\n    name: Cyber Risk Governance Metrics (CRGM)\n    state: active\n")
			edit(t, root, reviewDir+"/review.yaml", "working_group: jss", "working_group: crgm")
		}, "but the published document jcs belongs to jss"},
		{"reviews is reserved", func(t *testing.T, root string) {
			edit(t, root, "registry.yaml", "shortname: jcs", "shortname: reviews")
		}, `short name "reviews" is reserved`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := fixture(t)
			addReview(t, root)
			c.change(t, root)
			expectProblem(t, validate(root), c.want)
		})
	}
}

func TestGenerateChecksReviewsFirst(t *testing.T) {
	root := fixture(t)
	addReview(t, root)
	before := readText(t, root, reviewDir+"/index.md")
	edit(t, root, reviewDir+"/review.yaml", "closes: 2026-10-31", "closes: 2026-09-30")
	p, err := generate(root)
	if err != nil {
		t.Fatal(err)
	}
	expectProblem(t, p, "closes must be after opens")
	if readText(t, root, reviewDir+"/index.md") != before {
		t.Fatal("generate wrote files although review.yaml was wrong")
	}
}

func TestReviewHistory(t *testing.T) {
	cases := []struct {
		name   string
		change func(t *testing.T, root string)
		want   string // empty when the change is allowed
	}{
		{"extend", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", "closes: 2026-10-31", "closes: 2026-11-15")
		}, ""},
		{"withdraw", func(t *testing.T, root string) {
			edit(t, root, "docs/reviews/jsf/v1.0/01/review.yaml", "formats: [pdf]", "formats: [pdf]\nwithdrawn: 2026-10-10")
		}, ""},
		{"record disposition", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", "formats: [pdf]", "formats: [pdf]\ndisposition: https://github.com/OSTERA-org-Working-Groups/jss/issues/7")
		}, ""},
		{"a new review", func(t *testing.T, root string) {
			write(t, root, "docs/reviews/jcs/v1.0/02/review.yaml", strings.NewReplacer("draft: 1", "draft: 2", "2026-10-01", "2026-12-01", "2026-10-31", "2026-12-31").Replace(testReview)+"changes: >\n  Fixed the examples.\n")
			write(t, root, "docs/reviews/jcs/v1.0/02/draft-ts-jcs-02.pdf", "%PDF draft 2\n")
			mustSums(t, root, "docs/reviews/jcs/v1.0/02")
		}, ""},
		{"postpone", func(t *testing.T, root string) {
			edit(t, root, "docs/reviews/jsf/v1.0/01/review.yaml", "opens: 2026-10-01", "opens: 2026-10-08")
			edit(t, root, "docs/reviews/jsf/v1.0/01/review.yaml", "closes: 2026-10-31", "closes: 2026-11-07")
		}, ""},
		{"shorten", func(t *testing.T, root string) {
			edit(t, root, "docs/reviews/jsf/v1.0/01/review.yaml", "closes: 2026-10-31", "closes: 2026-10-15")
		}, ""},
		{"correct the title and summary", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", "A canonical form", "A canonical format")
			edit(t, root, reviewDir+"/review.yaml", "title: JSON Canonicalization Scheme (JCS)", "title: The JSON Canonicalization Scheme (JCS)")
		}, ""},
		{"add and remove changes", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", "formats: [pdf]", "formats: [pdf]\nchanges: >\n  Fixed the examples.")
		}, ""},
		{"change the type", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", "type: TS", "type: TN")
		}, "type cannot change once a review is announced, because it identifies the draft"},
		{"change the working group", func(t *testing.T, root string) {
			edit(t, root, "registry.yaml", "working_groups:\n", "working_groups:\n  crgm:\n    name: Cyber Risk Governance Metrics (CRGM)\n    state: active\n")
			edit(t, root, "docs/reviews/jsf/v1.0/01/review.yaml", "working_group: jss", "working_group: crgm")
		}, "working_group cannot change once a review is announced"},
		{"replace the draft", func(t *testing.T, root string) {
			write(t, root, reviewDir+"/draft-ts-jcs-01.pdf", "%PDF replaced\n")
			mustSums(t, root, reviewDir)
		}, "is part of a public review that has been announced"},
		{"add a format later", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", "formats: [pdf]", "formats: [pdf, md]")
			write(t, root, reviewDir+"/draft-ts-jcs-01.md", "# draft\n")
			mustSums(t, root, reviewDir)
		}, ""},
		{"add a file without adding its format", func(t *testing.T, root string) {
			write(t, root, reviewDir+"/draft-ts-jcs-01.md", "# draft\n")
			mustSums(t, root, reviewDir)
		}, "cannot be added to a public review that has been announced; only the file for a format newly added"},
		{"remove a format", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", "formats: [pdf]", "formats: [md]")
			os.Remove(filepath.Join(root, reviewDir, "draft-ts-jcs-01.pdf"))
			write(t, root, reviewDir+"/draft-ts-jcs-01.md", "# draft\n")
			mustSums(t, root, reviewDir)
		}, "formats can only be added to the end of the list"},
		{"remove the review", func(t *testing.T, root string) {
			os.RemoveAll(filepath.Join(root, reviewDir))
		}, "a public review is part of the record and is never removed"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := gitFixture(t)
			addReview(t, root)
			// A review of a work item that is not published yet.
			write(t, root, "docs/reviews/jsf/v1.0/01/review.yaml", strings.Replace(testReview, "shortname: jcs", "shortname: jsf", 1))
			write(t, root, "docs/reviews/jsf/v1.0/01/draft-ts-jsf-01.pdf", "%PDF\n")
			mustSums(t, root, "docs/reviews/jsf/v1.0/01")
			mustGenerate(t, root)
			commit(t, root, "announce")
			c.change(t, root)
			if c.want == "" {
				mustGenerate(t, root)
			} else {
				generate(root) // may refuse; the history check still runs
			}
			p := &Problems{}
			if err := checkHistory(root, "HEAD", p); err != nil {
				t.Fatal(err)
			}
			if c.want == "" {
				if p.Len() != 0 {
					t.Fatalf("expected no problems, got %+v", p.list)
				}
				if v := validate(root); v.Len() != 0 {
					t.Fatalf("expected a valid repository, got %+v", v.list)
				}
				return
			}
			expectProblem(t, p, c.want)
		})
	}
}

func TestSameDraftNumberInTwoVersions(t *testing.T) {
	// A Working Group may start its draft numbers again for a new version.
	root := fixture(t)
	addReview(t, root)
	dir := "docs/reviews/jcs/v1.1/01"
	write(t, root, dir+"/review.yaml", strings.NewReplacer(`version: "1.0"`, `version: "1.1"`, "2026-10-01", "2027-02-01", "2026-10-31", "2027-03-03").Replace(testReview))
	write(t, root, dir+"/draft-ts-jcs-01.pdf", "%PDF draft for 1.1\n")
	mustSums(t, root, dir)
	mustGenerate(t, root)
	if p := validate(root); p.Len() != 0 {
		t.Fatalf("expected no problems, got %+v", p.list)
	}
	list := readText(t, root, "docs/reviews/index.md")
	for _, want := range []string{"(/reviews/jcs/v1.0/01/)", "(/reviews/jcs/v1.1/01/)"} {
		if !strings.Contains(list, want) {
			t.Errorf("reviews/index.md does not link %s", want)
		}
	}
	if !strings.Contains(readText(t, root, "docs/reviews/jcs/index.md"), "<h3>Version 1.1</h3>") {
		t.Error("the review history should show one progress bar per version")
	}
}

func TestShortReviews(t *testing.T) {
	// Reviews may be any length, as long as one review of the version's
	// drafts lasted at least 30 days before the version is published.
	root := fixture(t)
	write(t, root, reviewDir+"/review.yaml", strings.Replace(testReview, "closes: 2026-10-31", "closes: 2026-10-16", 1))
	write(t, root, reviewDir+"/draft-ts-jcs-01.pdf", "%PDF draft\n")
	mustSums(t, root, reviewDir)
	dir := "docs/reviews/jcs/v1.0/02"
	write(t, root, dir+"/review.yaml", strings.NewReplacer("draft: 1", "draft: 2", "2026-10-01", "2026-10-20", "2026-10-31", "2026-11-19").Replace(testReview))
	write(t, root, dir+"/draft-ts-jcs-02.pdf", "%PDF draft 2\n")
	mustSums(t, root, dir)
	mustGenerate(t, root)
	if p := validate(root); p.Len() != 0 {
		t.Fatalf("a 15 day review and a 30 day review should be valid, got %+v", p.list)
	}

	// A work item that is not published yet may have only short reviews.
	write(t, root, "docs/reviews/jsf/v1.0/01/review.yaml", strings.NewReplacer("shortname: jcs", "shortname: jsf", "closes: 2026-10-31", "closes: 2026-10-08").Replace(testReview))
	write(t, root, "docs/reviews/jsf/v1.0/01/draft-ts-jsf-01.pdf", "%PDF\n")
	mustSums(t, root, "docs/reviews/jsf/v1.0/01")
	mustGenerate(t, root)
	if p := validate(root); p.Len() != 0 {
		t.Fatalf("a short review of an unpublished work item should be valid, got %+v", p.list)
	}
}

func TestReviewsJSON(t *testing.T) {
	root := fixture(t)
	addReview(t, root)
	text := readText(t, root, "docs/reviews/reviews.json")
	for _, want := range []string{
		`"name": "draft-ts-jcs-01"`,
		`"url": "https://pub.ostera.org/reviews/jcs/v1.0/01/"`,
		`"history_url": "https://pub.ostera.org/reviews/jcs/"`,
		`"working_group_name": "JSON Signature Scheme (JSS)"`,
		`"opens": "2026-10-01"`,
		`"files": [` + "\n" + `        "https://pub.ostera.org/reviews/jcs/v1.0/01/draft-ts-jcs-01.pdf"`,
		`"document_url": "https://pub.ostera.org/jcs/"`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("reviews.json does not contain %s", want)
		}
	}
	if strings.Contains(text, "withdrawn") {
		t.Error("reviews.json should leave out withdrawn when it is empty")
	}
	write(t, root, "docs/reviews/reviews.json", "{}\n")
	expectProblem(t, validate(root), "docs/reviews/reviews.json")
	if !strings.Contains(readText(t, root, "docs/reviews/jcs/index.md"), "width: wide") {
		t.Error("the review history page should use the wide layout")
	}
}

func TestUpdate(t *testing.T) {
	root := fixture(t)
	dir := "docs/reviews/jcs/v1.0/01"
	write(t, root, dir+"/review.yaml", testReview)
	write(t, root, dir+"/draft-ts-jcs-01.pdf", "%PDF draft\n")
	written, p, err := Update(root)
	if err != nil {
		t.Fatal(err)
	}
	if p.Len() != 0 || len(written) != 1 || written[0] != dir {
		t.Fatalf("expected only %s to be written, got %v and %+v", dir, written, p.list)
	}
	if v := validate(root); v.Len() != 0 {
		t.Fatalf("expected a valid repository after update, got %+v", v.list)
	}

	// Nothing to do the second time.
	written, _, _ = Update(root)
	if len(written) != 0 {
		t.Fatalf("expected nothing to be written, got %v", written)
	}

	// A new format in a published version adds a line to its SHA512SUMS.
	edit(t, root, "docs/jcs/metadata.yaml", "formats: [pdf, md]", "formats: [pdf, md, txt]")
	write(t, root, "docs/jcs/v1.0/OSTERA-TS-1000-2026-JCS-v1.0-en.txt", "JCS\n")
	written, p, _ = Update(root)
	if p.Len() != 0 || len(written) != 1 || written[0] != "docs/jcs/v1.0" {
		t.Fatalf("expected docs/jcs/v1.0 to be written, got %v and %+v", written, p.list)
	}
	if !strings.Contains(readText(t, root, "docs/jcs/v1.0/SHA512SUMS"), "OSTERA-TS-1000-2026-JCS-v1.0-en.txt") {
		t.Fatal("the new file is not in SHA512SUMS")
	}

	// A misnamed file stops update before anything is generated.
	write(t, root, dir+"/draft-ts-jcs-1.pdf", "x")
	_, p, _ = Update(root)
	expectProblem(t, p, "draft-ts-jcs-1.pdf: does not match the filename grammar or the draft naming rule")
}

func TestVersionLess(t *testing.T) {
	if !versionLess("1.9", "1.10") || versionLess("2.0", "1.10") || versionLess("1.0", "1.0") || !versionLess("1.0", "1.0.1") {
		t.Fatal("versionLess orders versions wrongly")
	}
}
