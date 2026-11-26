// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 OSTERA

package pub

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFormatMetadataRoundTrip(t *testing.T) {
	// formatMetadata writes every field, so reading what it writes gives
	// back exactly what it was given: for the test metadata, and for every
	// metadata.yaml in the repository, if there are any.
	found, _ := filepath.Glob("../../../docs/*/metadata.yaml")
	for _, path := range append([]string{""}, found...) {
		data := []byte(testMetadata)
		if path != "" {
			var err error
			if data, err = os.ReadFile(path); err != nil {
				t.Fatal(err)
			}
		}
		p := &Problems{}
		m := parseMetadata("metadata.yaml", data, p)
		if m == nil || p.Len() > 0 {
			t.Fatalf("%s: cannot parse: %+v", path, p.list)
		}
		again := parseMetadata("again.yaml", []byte(formatMetadata(m)), p)
		if again == nil || p.Len() > 0 {
			t.Fatalf("%s: cannot parse what formatMetadata wrote: %+v\n%s", path, p.list, formatMetadata(m))
		}
		m.Summary, again.Summary = strings.Join(strings.Fields(m.Summary), " "), strings.Join(strings.Fields(again.Summary), " ")
		for i := range m.Versions {
			m.Versions[i].Citation = strings.Join(strings.Fields(m.Versions[i].Citation), " ")
			again.Versions[i].Citation = strings.Join(strings.Fields(again.Versions[i].Citation), " ")
		}
		// An empty list may come back as an empty list or as no list.
		norm := func(x *Metadata) string {
			data, _ := json.Marshal(x)
			return strings.ReplaceAll(string(data), "null", "[]")
		}
		if norm(m) != norm(again) {
			t.Errorf("%s: round trip changed the metadata:\n%s\n%s", path, norm(m), norm(again))
		}
	}
}

func TestCitationDate(t *testing.T) {
	for in, want := range map[string]string{
		`A. Rundgren, "JCS," OSTERA, ver. 1.0, Nov. 24, 2026. [Online].`: "2026-11-24",
		`B. Jordan, "JSS," OSTERA, ver. 1.0, Sept. 5, 2026.`:             "2026-09-05",
		`B. Jordan, "JSS," OSTERA, ver. 1.1. [Online].`:                  "",
	} {
		if got := citationDate(in); got != want {
			t.Errorf("citationDate(%q) = %q, want %q", in, got, want)
		}
	}
}

// wgSite serves a work item's registry.json and its PDF, like a working
// group site.
func wgSite(t *testing.T, id string, item map[string]any, pdfs map[string][]byte) *httptest.Server {
	t.Helper()
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/jss/work-items/"+id+"/registry.json", func(w http.ResponseWriter, r *http.Request) { w.Write(data) })
	for path, body := range pdfs {
		body := body
		mux.HandleFunc("/jss/work-items/"+id+"/"+path, func(w http.ResponseWriter, r *http.Request) { w.Write(body) })
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func sha512Hex(b []byte) string {
	s := sha512.Sum512(b)
	return hex.EncodeToString(s[:])
}

const jesCitation = `B. Jordan, "JSON Encryption Scheme (JES)," OSTERA, Geneva, Switzerland, OSTERA TS 1001, ver. 1.0, Dec. 20, 2026. [Online]. Available: https://pub.ostera.org/jes/`

// jesItem is the registry.json of a work item that is ready to publish:
// two drafts of version 1.0, a Last Call of each, and a 30 day Public
// Review of the second, all closed.
func jesItem(pdf []byte) map[string]any {
	person := func(name string) map[string]any { return map[string]any{"name": name, "affiliation": "Individual"} }
	return map[string]any{
		"name": "JSON Encryption Scheme (JES)", "number": "OSTERA TS 1001", "type": "TS", "status": "Approved",
		"summary": "Defines how to encrypt JSON data.",
		"versions": []any{map[string]any{"version": "1.0", "approved": map[string]any{"draft": "02", "date": "2026-12-20"}, "published": nil, "drafts": []any{
			map[string]any{"draft": "01", "name": "draft-ts-jordan-jes-01", "released": "2026-08-25",
				"authors":   []any{person("Bret Jordan")},
				"last_call": map[string]any{"opens": "2026-09-01", "closes": "2026-09-15"},
				"files":     []any{map[string]any{"name": "draft-ts-jordan-jes-01.pdf", "format": "PDF", "sha512": "00"}}},
			map[string]any{"draft": "02", "name": "draft-ts-jordan-jes-02", "released": "2026-10-10",
				"authors":       []any{person("Bret Jordan")},
				"editors":       []any{person("Anders Rundgren")},
				"obsoletes":     []any{map[string]any{"id": "IETF RFC 7516", "url": "https://www.rfc-editor.org/info/rfc7516"}},
				"citation":      jesCitation,
				"keywords":      []any{"JSON", "Encryption", "DEMO"},
				"last_call":     map[string]any{"opens": "2026-10-15", "closes": "2026-10-29"},
				"public_review": map[string]any{"opens": "2026-11-01", "closes": "2026-11-30"},
				"files": []any{
					map[string]any{"name": "draft-ts-jordan-jes-02.pdf", "format": "PDF", "sha512": sha512Hex(pdf)},
					map[string]any{"name": "draft-ts-jordan-jes-02.docx", "format": "DOCX", "sha512": "00"},
				}},
		}}},
	}
}

// publishFixture is a committed repository with number 1001 assigned to
// jes, which is not published yet.
func publishFixture(t *testing.T) string {
	t.Helper()
	root := gitFixture(t)
	edit(t, root, "registry.yaml", "    allocated: 2026-11-04\n", "    allocated: 2026-11-04\n  - number: 1001\n    shortname: jes\n    state: assigned\n    allocated: 2026-08-25\n")
	if _, err := git(root, "-c", "user.name=t", "-c", "user.email=t@example.org", "commit", "-qam", "assign 1001"); err != nil {
		t.Fatal(err)
	}
	return root
}

var publishNow = time.Date(2027, 1, 10, 12, 0, 0, 0, time.UTC)

func TestPublishNewDocument(t *testing.T) {
	root := publishFixture(t)
	write(t, root, "notes.txt", "not part of the publication\n")
	pdf := []byte("%PDF jes 02\n")
	srv := wgSite(t, "jes", jesItem(pdf), map[string][]byte{"v1.0/draft-ts-jordan-jes-02.pdf": pdf})

	var out bytes.Buffer
	// The defaults for the date and the branch, and yes to the commit.
	if err := Publish(root, srv.URL+"/jss/work-items/jes/", strings.NewReader("\n\nyes\n"), &out, publishNow); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	branch, _ := git(root, "branch", "--show-current")
	if strings.TrimSpace(branch) != "publish-jes-v1.0" {
		t.Errorf("branch is %q", branch)
	}
	committed, _ := git(root, "show", "--name-only", "--format=%s", "HEAD")
	want := "Publish OSTERA TS 1001, Version 1.0 (JES)\n\ndocs/catalog.json\ndocs/index.md\ndocs/jes/index.md\ndocs/jes/metadata.yaml\ndocs/jes/v1.0/OSTERA-TS-1001-2026-JES-v1.0-en.pdf\ndocs/jes/v1.0/SHA512SUMS\ndocs/jes/v1.0/index.md\nregistry.yaml\n"
	if committed != want {
		t.Errorf("committed:\n%s\nwant:\n%s", committed, want)
	}
	if status, _ := git(root, "status", "--porcelain"); status != "?? notes.txt\n" {
		t.Errorf("after the commit, git status is %q; only notes.txt should be left, uncommitted", status)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "docs/jes/v1.0/OSTERA-TS-1001-2026-JES-v1.0-en.pdf")); !bytes.Equal(got, pdf) {
		t.Errorf("the published PDF is not the draft PDF")
	}
	meta := readText(t, root, "docs/jes/metadata.yaml")
	for _, s := range []string{
		"keywords:\n  - DEMO\n  - Encryption\n  - JSON\n",
		"    published: 2026-12-20\n",
		"    formats: [pdf]\n",
		"    authors:\n      - name: Bret Jordan\n        affiliation: Individual\n    editors:\n      - name: Anders Rundgren\n",
		"obsoletes:\n  - id: IETF RFC 7516\n",
	} {
		if !strings.Contains(meta, s) {
			t.Errorf("metadata.yaml is missing %q:\n%s", s, meta)
		}
	}
	if !strings.Contains(readText(t, root, "registry.yaml"), "  - number: 1001\n    shortname: jes\n    state: active\n") {
		t.Errorf("registry.yaml does not show number 1001 as active")
	}
	p := validate(root)
	if err := checkHistory(root, "main", p); err != nil {
		t.Fatal(err)
	}
	if p.Len() != 0 {
		t.Fatalf("expected no problems, got %+v", p.list)
	}
	for _, s := range []string{"Summary", "number 1001 changes from assigned to active", "git push -u origin publish-jes-v1.0", "make published, for version 1.0 of jes, published 2026-12-20"} {
		if !strings.Contains(out.String(), s) {
			t.Errorf("output is missing %q:\n%s", s, out.String())
		}
	}
}

func TestPublishNewVersion(t *testing.T) {
	// Version 1.1 of jcs, which is already published as version 1.0.
	root := gitFixture(t)
	pdf := []byte("%PDF jcs 1.1\n")
	item := map[string]any{
		"name": "JSON Canonicalization Scheme (JCS)", "number": "OSTERA TS 1000", "type": "TS",
		"summary": "Defines a canonical representation of JSON data.",
		"versions": []any{
			map[string]any{"version": "1.0", "published": "2026-11-24", "drafts": []any{}},
			map[string]any{"version": "1.1", "approved": map[string]any{"draft": "01", "date": "2027-03-01"}, "drafts": []any{map[string]any{
				"draft": "01", "name": "draft-ts-jordan-jcs-01", "released": "2027-01-05",
				"authors":       []any{map[string]any{"name": "Bret Jordan", "affiliation": "Individual"}},
				"citation":      `B. Jordan, "JCS," OSTERA, ver. 1.1, Mar. 1, 2027.`,
				"keywords":      []any{"JSON"},
				"last_call":     map[string]any{"opens": "2027-01-06", "closes": "2027-01-19"},
				"public_review": map[string]any{"opens": "2027-01-20", "closes": "2027-02-18"},
				"files":         []any{map[string]any{"name": "draft-ts-jordan-jcs-01.pdf", "format": "PDF", "sha512": sha512Hex(pdf)}},
			}}},
		},
	}
	srv := wgSite(t, "jcs", item, map[string][]byte{"v1.1/draft-ts-jordan-jcs-01.pdf": pdf})
	var out bytes.Buffer
	now := time.Date(2027, 3, 1, 12, 0, 0, 0, time.UTC)
	if err := Publish(root, srv.URL+"/jss/work-items/jcs/v1.1/01/", strings.NewReader("\nmy-branch\ny\n"), &out, now); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	meta := readText(t, root, "docs/jcs/metadata.yaml")
	if !strings.Contains(meta, "  - version: \"1.0\"\n    type: TS\n    status: superseded\n") || !strings.Contains(meta, "  - version: \"1.1\"\n    type: TS\n    status: published\n    published: 2027-03-01\n") {
		t.Errorf("metadata.yaml:\n%s", meta)
	}
	committed, _ := git(root, "show", "--name-only", "--format=", "HEAD")
	if strings.Contains(committed, "registry.yaml") || !strings.Contains(committed, "docs/jcs/v1.1/OSTERA-TS-1000-2027-JCS-v1.1-en.pdf") {
		t.Errorf("committed:\n%s", committed)
	}
	if b, _ := git(root, "branch", "--show-current"); strings.TrimSpace(b) != "my-branch" {
		t.Errorf("branch is %q", b)
	}
	p := validate(root)
	if err := checkHistory(root, "main", p); err != nil {
		t.Fatal(err)
	}
	if p.Len() != 0 {
		t.Fatalf("expected no problems, got %+v", p.list)
	}
}

func TestPublishRefused(t *testing.T) {
	pdf := []byte("%PDF jes 02\n")
	cases := []struct {
		name   string
		change func(item map[string]any)
		url    string
		now    time.Time
		want   string
	}{
		{"review still open", nil, "", time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC), "has no Public Review of at least 30 days that has closed"},
		{"not the latest draft", nil, "v1.0/01/", publishNow, "is not the latest draft"},
		{"short review", func(item map[string]any) {
			draft02(item)["public_review"] = map[string]any{"opens": "2026-11-01", "closes": "2026-11-20"}
		}, "", publishNow, "has no Public Review of at least 30 days that has closed"},
		{"short last calls", func(item map[string]any) {
			item["versions"].([]any)[0].(map[string]any)["drafts"].([]any)[0].(map[string]any)["last_call"] = map[string]any{"opens": "2026-09-01", "closes": "2026-09-10"}
			draft02(item)["last_call"] = map[string]any{"opens": "2026-10-15", "closes": "2026-10-20"}
		}, "", publishNow, "has no Working Group Last Call of at least 14 days"},
		{"withdrawn review", func(item map[string]any) {
			draft02(item)["public_review"] = map[string]any{"opens": "2026-11-01", "closes": "2026-11-30", "withdrawn": "2026-11-15"}
		}, "", publishNow, "has no Public Review of at least 30 days that has closed"},
		{"no people", func(item map[string]any) { delete(draft02(item), "authors"); delete(draft02(item), "editors") }, "", publishNow, "lists no authors and no editors"},
		{"no number", func(item map[string]any) { item["number"] = "" }, "", publishNow, "has no document number"},
		{"not approved", func(item map[string]any) {
			item["versions"].([]any)[0].(map[string]any)["approved"] = nil
		}, "", publishNow, "has not been approved by the working group"},
		{"approved before the latest draft", func(item map[string]any) {
			item["versions"].([]any)[0].(map[string]any)["approved"] = map[string]any{"draft": "01", "date": "2026-09-20"}
		}, "", publishNow, "the working group must approve the latest draft"},
		{"approved before the release", func(item map[string]any) {
			item["versions"].([]any)[0].(map[string]any)["approved"] = map[string]any{"draft": "02", "date": "2026-10-01"}
		}, "", publishNow, "was approved on 2026-10-01, before"},
		{"obsoletes without url", func(item map[string]any) {
			draft02(item)["obsoletes"] = []any{map[string]any{"id": "IETF RFC 7516"}}
		}, "", publishNow, "needs a url starting with https://"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := publishFixture(t)
			item := jesItem(pdf)
			if c.change != nil {
				c.change(item)
			}
			srv := wgSite(t, "jes", item, map[string][]byte{"v1.0/draft-ts-jordan-jes-02.pdf": pdf})
			var out bytes.Buffer
			err := Publish(root, srv.URL+"/jss/work-items/jes/"+c.url, strings.NewReader("\n\nyes\n"), &out, c.now)
			if err == nil || !strings.Contains(out.String()+err.Error(), c.want) {
				t.Fatalf("want an error with %q, got %v\n%s", c.want, err, out.String())
			}
			if b, _ := git(root, "branch", "--list"); strings.TrimSpace(b) != "* main" {
				t.Errorf("a branch was created: %q", b)
			}
		})
	}

	// Shorter reviews may come before a long enough one: a 30 day Public
	// Review of draft 01 is enough, even though draft 02 only had 20 days.
	// The authors are removed so that the publication still stops, before
	// anything is written.
	t.Run("one long enough review", func(t *testing.T) {
		root := publishFixture(t)
		item := jesItem(pdf)
		item["versions"].([]any)[0].(map[string]any)["drafts"].([]any)[0].(map[string]any)["public_review"] = map[string]any{"opens": "2026-09-16", "closes": "2026-10-15"}
		draft02(item)["public_review"] = map[string]any{"opens": "2026-11-01", "closes": "2026-11-20"}
		delete(draft02(item), "authors")
		delete(draft02(item), "editors")
		srv := wgSite(t, "jes", item, map[string][]byte{"v1.0/draft-ts-jordan-jes-02.pdf": pdf})
		var out bytes.Buffer
		err := Publish(root, srv.URL+"/jss/work-items/jes/", strings.NewReader("\n\nyes\n"), &out, publishNow)
		if err == nil || !strings.Contains(out.String()+err.Error(), "lists no authors and no editors") {
			t.Fatalf("want only the authors error, got %v\n%s", err, out.String())
		}
		if strings.Contains(out.String()+err.Error(), "Public Review") {
			t.Errorf("a 30 day Public Review of draft 01 should be enough:\n%s", out.String())
		}
	})

	t.Run("wrong sum", func(t *testing.T) {
		root := publishFixture(t)
		srv := wgSite(t, "jes", jesItem(pdf), map[string][]byte{"v1.0/draft-ts-jordan-jes-02.pdf": []byte("%PDF changed\n")})
		var out bytes.Buffer
		err := Publish(root, srv.URL+"/jss/work-items/jes/", strings.NewReader("\n\nyes\n"), &out, publishNow)
		if err == nil || !strings.Contains(err.Error(), "does not match its SHA-512 sum") {
			t.Fatalf("want a sum error, got %v", err)
		}
		if head, _ := git(root, "log", "-1", "--format=%s"); strings.TrimSpace(head) != "assign 1001" {
			t.Errorf("something was committed: %q", head)
		}
	})

	t.Run("not confirmed", func(t *testing.T) {
		root := publishFixture(t)
		srv := wgSite(t, "jes", jesItem(pdf), map[string][]byte{"v1.0/draft-ts-jordan-jes-02.pdf": pdf})
		var out bytes.Buffer
		if err := Publish(root, srv.URL+"/jss/work-items/jes/", strings.NewReader("\n\nno\n"), &out, publishNow); err != nil {
			t.Fatal(err)
		}
		if head, _ := git(root, "log", "-1", "--format=%s"); strings.TrimSpace(head) != "assign 1001" {
			t.Errorf("something was committed: %q", head)
		}
		if !strings.Contains(out.String(), "Nothing was committed") {
			t.Errorf("output:\n%s", out.String())
		}
	})
}

func draft02(item map[string]any) map[string]any {
	return item["versions"].([]any)[0].(map[string]any)["drafts"].([]any)[1].(map[string]any)
}
