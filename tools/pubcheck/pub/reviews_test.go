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
		{"too short", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", "closes: 2026-10-31", "closes: 2026-10-30")
		}, "closes must be at least 30 days after opens (Working Methods, Article 29); the earliest allowed is 2026-10-31"},
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
	edit(t, root, reviewDir+"/review.yaml", "closes: 2026-10-31", "closes: 2026-10-02")
	p, err := generate(root)
	if err != nil {
		t.Fatal(err)
	}
	expectProblem(t, p, "closes must be at least 30 days after opens")
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
			edit(t, root, reviewDir+"/review.yaml", "formats: [pdf]", "formats: [pdf]\nwithdrawn: 2026-10-10")
		}, ""},
		{"record disposition", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", "formats: [pdf]", "formats: [pdf]\ndisposition: https://github.com/OSTERA-org-Working-Groups/jss/issues/7")
		}, ""},
		{"a new review", func(t *testing.T, root string) {
			write(t, root, "docs/reviews/jcs/v1.0/02/review.yaml", strings.NewReplacer("draft: 1", "draft: 2", "2026-10-01", "2026-12-01", "2026-10-31", "2026-12-31").Replace(testReview)+"changes: >\n  Fixed the examples.\n")
			write(t, root, "docs/reviews/jcs/v1.0/02/draft-ts-jcs-02.pdf", "%PDF draft 2\n")
			mustSums(t, root, "docs/reviews/jcs/v1.0/02")
		}, ""},
		{"shorten", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", "closes: 2026-10-31", "closes: 2026-10-15")
		}, "a review can be extended, but never shortened"},
		{"edit the summary", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", "A canonical form", "A canonical format")
		}, "only closes (to a later date), withdrawn, and disposition may change"},
		{"replace the draft", func(t *testing.T, root string) {
			write(t, root, reviewDir+"/draft-ts-jcs-01.pdf", "%PDF replaced\n")
			mustSums(t, root, reviewDir)
		}, "is part of a public review that has been announced"},
		{"add a format later", func(t *testing.T, root string) {
			edit(t, root, reviewDir+"/review.yaml", "formats: [pdf]", "formats: [pdf, md]")
			write(t, root, reviewDir+"/draft-ts-jcs-01.md", "# draft\n")
			mustSums(t, root, reviewDir)
		}, "is part of a public review that has been announced"},
		{"remove the review", func(t *testing.T, root string) {
			os.RemoveAll(filepath.Join(root, reviewDir))
		}, "a public review is part of the record and is never removed"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := gitFixture(t)
			addReview(t, root)
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

func TestVersionLess(t *testing.T) {
	if !versionLess("1.9", "1.10") || versionLess("2.0", "1.10") || versionLess("1.0", "1.0") || !versionLess("1.0", "1.0.1") {
		t.Fatal("versionLess orders versions wrongly")
	}
}
