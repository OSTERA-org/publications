// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 OSTERA

package pub

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

var registryNow = time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)

// emptyLists is a fixture with no Working Groups and no numbers, as a new
// repository starts. It has no published documents either.
func emptyLists(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	reg := strings.Replace(testRegistry, "working_groups:\n  jss:\n    name: JSON Signature Scheme (JSS)\n    state: active\n", "working_groups: {}\n\n# Document numbers\n", 1)
	reg = reg[:strings.Index(reg, "numbers:")] + "numbers: []\n"
	write(t, root, "registry.yaml", reg)
	write(t, root, "docs/index.md", "---\nlayout: home\n---\n\n"+beginMarker("documents")+"\n"+endMarker("documents")+"\n")
	write(t, root, "docs/about.md", "---\nlayout: page\n---\n\n"+beginMarker("document-types")+"\n"+endMarker("document-types")+"\n")
	write(t, root, "docs/_config.yml", "title: OSTERA Publications\n")
	write(t, root, "docs/CNAME", "pub.ostera.org\n")
	write(t, root, "docs/404.md", "---\nlayout: page\n---\n")
	mustGenerate(t, root)
	return root
}

// TestMain keeps the tests off the network: every lookup goes to an
// address that refuses connections, unless a test sets up its own sites.
func TestMain(m *testing.M) {
	hubDataURL = "http://127.0.0.1:1/hub.yml"
	wgConfigURL = "http://127.0.0.1:1/%s.yml"
	lookupSiteURL = "http://127.0.0.1:1"
	os.Exit(m.Run())
}

const testHub = `- id: jss
  name: "JSON Signature Scheme"
  abbr: JSS
  purpose: "Signing JSON."
  website: true
  repo: true
  charter: ""
- id: crgm
  name: "Cyber Risk Governance Metrics"
  abbr: CRGM
  website: true
- id: future
  name: "Not Started Yet"
  website: false
`

// sites serves the list of Working Groups on wg.ostera.org, the website
// settings of Working Group repositories, and, when wg is not nil, the
// Working Group sites, for the length of one test.
func sites(t *testing.T, hub string, configs map[string]string, wg *httptest.Server) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/hub.yml", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(hub)) })
	for id, cfg := range configs {
		cfg := cfg
		mux.HandleFunc("/cfg/"+id+".yml", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(cfg)) })
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	oldHub, oldCfg, oldSite := hubDataURL, wgConfigURL, lookupSiteURL
	hubDataURL, wgConfigURL = srv.URL+"/hub.yml", srv.URL+"/cfg/%s.yml"
	if wg != nil {
		lookupSiteURL = wg.URL
	}
	t.Cleanup(func() { hubDataURL, wgConfigURL, lookupSiteURL = oldHub, oldCfg, oldSite })
}

func mustRegistry(t *testing.T, root string) *Registry {
	t.Helper()
	reg, _, err := readRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func TestAddWorkingGroup(t *testing.T) {
	sites(t, testHub, map[string]string{
		"cnscp": "title: \"OSTERA CNSCP Working Group\"\nshort_title: \"Connectivity Naming System (CNSCP)\"\n",
	}, nil)
	root := emptyLists(t)

	// The first one replaces {}, and takes the name from wg.ostera.org.
	var out bytes.Buffer
	if err := AddWorkingGroup(root, "", strings.NewReader("jss\n\n\n"), &out); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	for _, s := range []string{"Name of Working Group jss, such as \"JSON Signature Scheme (JSS)\" [JSON Signature Scheme (JSS)]", "the repository is valid", "git commit -m \"Add Working Group JSS\""} {
		if !strings.Contains(out.String(), s) {
			t.Errorf("output is missing %q:\n%s", s, out.String())
		}
	}
	// The second one is given as an argument, and goes after the first,
	// before the comments that follow.
	out.Reset()
	if err := AddWorkingGroup(root, "crgm", strings.NewReader("\nyes\n"), &out); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	// One that is not on wg.ostera.org yet: the name comes from the
	// website settings of its repository.
	out.Reset()
	if err := AddWorkingGroup(root, "cnscp", strings.NewReader("\n\n"), &out); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	// One with neither needs the name typed in.
	out.Reset()
	if err := AddWorkingGroup(root, "", strings.NewReader("xyz\nThe XYZ Group (XYZ)\n\n"), &out); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "so there is no suggestion") {
		t.Errorf("output does not say there is no suggestion:\n%s", out.String())
	}
	text := readText(t, root, "registry.yaml")
	want := "working_groups:\n  jss:\n    name: JSON Signature Scheme (JSS)\n    state: active\n  crgm:\n    name: Cyber Risk Governance Metrics (CRGM)\n    state: active\n  cnscp:\n    name: Connectivity Naming System (CNSCP)\n    state: active\n  xyz:\n    name: The XYZ Group (XYZ)\n    state: active\n\n# Document numbers\n"
	if !strings.Contains(text, want) {
		t.Errorf("registry.yaml does not contain:\n%s\nit is:\n%s", want, text)
	}
	if n := len(mustRegistry(t, root).WorkingGroups); n != 4 {
		t.Errorf("registry.yaml has %d Working Groups, want 4", n)
	}

	// Refused, and nothing changed.
	for input, want := range map[string]string{
		"jss\n":          "already in registry.yaml",
		"JSS\n":          "is not a Working Group id",
		"\n":             "no Working Group id was given",
		"newwg\n\n":      "no name was given",
		"newwg\nX\nno\n": "",
	} {
		before := readText(t, root, "registry.yaml")
		out.Reset()
		err := AddWorkingGroup(root, "", strings.NewReader(input), &out)
		switch {
		case want == "" && err != nil:
			t.Errorf("%q: %v", input, err)
		case want != "" && (err == nil || !strings.Contains(err.Error(), want)):
			t.Errorf("%q: got %v, want an error containing %q", input, err, want)
		}
		if readText(t, root, "registry.yaml") != before {
			t.Errorf("%q changed registry.yaml", input)
		}
	}
}

func TestReserveAndAssign(t *testing.T) {
	root := emptyLists(t)
	var out bytes.Buffer
	run := func(state, address, input string) {
		t.Helper()
		out.Reset()
		if err := AllocateNumber(root, state, address, strings.NewReader(input), &out, registryNow); err != nil {
			t.Fatalf("%s %q: %v\n%s", state, input, err, out.String())
		}
	}

	// The first technical number, with an earlier date.
	run("assigned", "", "jcs\n\n2026-11-19\n\n")
	for _, s := range []string{"Type code (STD, TS, SUP, TN, TP, TR, PD) [TS]", "Number: 1000, the next free number in the technical range", "Date allocated (yyyy-mm-dd) [2026-11-26]", "OSTERA TS 1000", "git commit -m \"Assign number 1000 to JCS\""} {
		if !strings.Contains(out.String(), s) {
			t.Errorf("output is missing %q:\n%s", s, out.String())
		}
	}
	// The next one, reserved, with today's date.
	run("reserved", "", "jes\nts\n\n\n")
	// An organizational number comes from the other range.
	run("assigned", "", "rules\nPD\n\nyes\n")
	// The reserved one, now assigned: it keeps its number and date.
	run("assigned", "", "jes\n\n")
	if !strings.Contains(out.String(), "Number 1001 is reserved for jes, allocated 2026-11-26") {
		t.Errorf("output does not show the reserved number:\n%s", out.String())
	}

	text := readText(t, root, "registry.yaml")
	want := "numbers:\n" +
		"  - number: 1000\n    shortname: jcs\n    state: assigned\n    allocated: 2026-11-19\n" +
		"  - number: 1001\n    shortname: jes\n    state: assigned\n    allocated: 2026-11-26\n" +
		"  - number: 1\n    shortname: rules\n    state: assigned\n    allocated: 2026-11-26\n"
	if !strings.HasSuffix(text, want) {
		t.Errorf("registry.yaml does not end with:\n%s\nit is:\n%s", want, text)
	}

	// The next technical number counts retired numbers too.
	edit(t, root, "registry.yaml", "    shortname: jes\n    state: assigned\n", "    shortname: jes\n    state: retired\n")
	run("reserved", "", "jsp\n\n\n\n")
	if !strings.Contains(out.String(), "Number: 1002") {
		t.Errorf("the next number after a retired 1001 is not 1002:\n%s", out.String())
	}

	// Refused, and nothing changed.
	for _, c := range []struct{ state, input, want string }{
		{"reserved", "jcs\n", "jcs already has number 1000, which is assigned"},
		{"assigned", "jcs\n", "jcs already has number 1000, which is assigned"},
		{"assigned", "about\n", "is reserved for the site"},
		{"assigned", "JCS\n", "is not a short name"},
		{"assigned", "\n", "no short name was given"},
		{"assigned", "new\nXX\n", "is not a type code"},
		{"assigned", "new\n\n2026-11-31\n", "is not a date"},
		{"assigned", "new\n\n2026-11-27\n", "is in the future"},
		{"assigned", "new\n\n\nno\n", ""},
	} {
		before := readText(t, root, "registry.yaml")
		out.Reset()
		err := AllocateNumber(root, c.state, "", strings.NewReader(c.input), &out, registryNow)
		switch {
		case c.want == "" && err != nil:
			t.Errorf("%q: %v", c.input, err)
		case c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)):
			t.Errorf("%q: got %v, want an error containing %q", c.input, err, c.want)
		}
		if readText(t, root, "registry.yaml") != before {
			t.Errorf("%q changed registry.yaml", c.input)
		}
	}
}

// wgSites serves work items on several Working Group sites, keyed by
// "<wg>/<short name>".
func wgSites(t *testing.T, items map[string]map[string]any) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for key, item := range items {
		data, err := json.Marshal(item)
		if err != nil {
			t.Fatal(err)
		}
		wg, id, _ := strings.Cut(key, "/")
		mux.HandleFunc("/"+wg+"/work-items/"+id+"/registry.json", func(w http.ResponseWriter, r *http.Request) { w.Write(data) })
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestAssignLooksUpTheWorkItem(t *testing.T) {
	// Only the short name is given. The work item is found on the jss
	// site (not on crgm), its type is suggested, and its Working Group,
	// which registry.yaml does not have yet, is added in the same change.
	item := jesItem([]byte("%PDF\n"))
	item["type"] = "TN"
	item["number"] = "OSTERA TN 1000"
	sites(t, testHub, nil, wgSites(t, map[string]map[string]any{"jss/jes": item}))
	root := emptyLists(t)
	var out bytes.Buffer
	if err := AllocateNumber(root, "assigned", "", strings.NewReader("jes\n\n\n\n\n\n"), &out, registryNow); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	for _, s := range []string{
		"Found JSON Encryption Scheme (JES), in Working Group jss",
		"[TN]",
		"Working Group jss is not in registry.yaml yet.",
		"[JSON Signature Scheme (JSS)]",
		"OSTERA TN 1000",
		"git commit -m \"Assign number 1000 to JES, and add Working Group JSS\"",
	} {
		if !strings.Contains(out.String(), s) {
			t.Errorf("output is missing %q:\n%s", s, out.String())
		}
	}
	if strings.Contains(out.String(), "note: the work item already gives") {
		t.Errorf("the number on the page matches, so there should be no note:\n%s", out.String())
	}
	reg := mustRegistry(t, root)
	if !wgIs(reg, "jss", "JSON Signature Scheme (JSS)") || !numberIs(reg, 1000, "jes", "assigned", "2026-11-26") {
		t.Errorf("registry.yaml is wrong:\n%s", readText(t, root, "registry.yaml"))
	}

	// A number on the page that is not the next free one gets a note.
	item = jesItem([]byte("%PDF\n"))
	item["number"] = "OSTERA TS 1007"
	sites(t, testHub, nil, wgSites(t, map[string]map[string]any{"jss/jsp": item}))
	out.Reset()
	if err := AllocateNumber(root, "reserved", "jsp", strings.NewReader("\n\n\n"), &out, registryNow); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "note: the work item already gives OSTERA TS 1007 on its page. The chairs need to change it to OSTERA TS 1001.") {
		t.Errorf("output is missing the note:\n%s", out.String())
	}
}

func TestAssignWhenTheWorkItemCannotBeFound(t *testing.T) {
	// Nothing can be reached: the short name is still used, and the
	// questions are asked with the usual suggestions.
	root := emptyLists(t)
	var out bytes.Buffer
	if err := AllocateNumber(root, "assigned", "jcs", strings.NewReader("\n2026-11-19\n\n"), &out, registryNow); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	for _, s := range []string{"so there are no suggestions", "[TS]", "OSTERA TS 1000"} {
		if !strings.Contains(out.String(), s) {
			t.Errorf("output is missing %q:\n%s", s, out.String())
		}
	}

	// A page that cannot be read gives the short name too.
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	out.Reset()
	if err := AllocateNumber(root, "reserved", srv.URL+"/jss/work-items/jes/", strings.NewReader("\n\n\n"), &out, registryNow); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "could not be read") || !numberIs(mustRegistry(t, root), 1001, "jes", "reserved", "2026-11-26") {
		t.Errorf("got:\n%s", out.String())
	}

	// The same short name on two sites must be given by its page.
	sites(t, testHub, nil, wgSites(t, map[string]map[string]any{"jss/two": jesItem(nil), "crgm/two": jesItem(nil)}))
	out.Reset()
	if _, _, err := findWorkItem("two", nil); err == nil || !strings.Contains(err.Error(), "more than one Working Group has a work item called two") {
		t.Errorf("got %v", err)
	}
}

func TestInsertEntry(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"a: 1\nnumbers: []\n", "a: 1\nnumbers:\n  - x\n"},
		{"numbers: [] # none yet\nb: 2\n", "numbers:\n  - x\nb: 2\n"},
		{"numbers:\n  - y\n  # a comment\n\n# Next\nb: 2\n", "numbers:\n  - y\n  # a comment\n  - x\n\n# Next\nb: 2\n"},
		{"numbers:\n  - y", "numbers:\n  - y\n  - x\n"},
	} {
		got, err := insertEntry([]byte(c.in), "numbers", "  - x\n", "[]")
		if err != nil || string(got) != c.want {
			t.Errorf("insertEntry(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
	if _, err := insertEntry([]byte("numbers: [1]\n"), "numbers", "  - x\n", "[]"); err == nil {
		t.Errorf("insertEntry accepted a list written on one line")
	}
}

func TestPublishAddsTheNextNumber(t *testing.T) {
	// The Secretariat gave the chairs number 1001 but did not record it:
	// publish adds it, because it is the next free number.
	root := gitFixture(t)
	pdf := []byte("%PDF jes 02\n")
	srv := wgSite(t, "jes", jesItem(pdf), map[string][]byte{"v1.0/draft-ts-jordan-jes-02.pdf": pdf})
	sites(t, testHub, nil, srv)
	var out bytes.Buffer
	// Only the short name is given, so publish finds the work item.
	if err := Publish(root, "", strings.NewReader("jes\n\n\nyes\n"), &out, publishNow); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "Found JSON Encryption Scheme (JES), in Working Group jss") {
		t.Errorf("publish did not find the work item:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "registry.yaml: number 1001 is added for jes, as active, allocated 2027-01-10") {
		t.Errorf("the summary does not show the new number:\n%s", out.String())
	}
	if !strings.HasSuffix(readText(t, root, "registry.yaml"), "  - number: 1001\n    shortname: jes\n    state: active\n    allocated: 2027-01-10\n") {
		t.Errorf("registry.yaml does not end with number 1001:\n%s", readText(t, root, "registry.yaml"))
	}
	p := validate(root)
	if err := checkHistory(root, "main", p); err != nil {
		t.Fatal(err)
	}
	if p.Len() != 0 {
		t.Fatalf("expected no problems, got %+v", p.list)
	}

	// Any other number must be assigned first.
	root = gitFixture(t)
	item := jesItem(pdf)
	item["number"] = "OSTERA TS 1005"
	srv = wgSite(t, "jes", item, map[string][]byte{"v1.0/draft-ts-jordan-jes-02.pdf": pdf})
	out.Reset()
	err := Publish(root, srv.URL+"/jss/work-items/jes/", strings.NewReader("\n\nyes\n"), &out, publishNow)
	if err == nil || !strings.Contains(out.String(), "it is not the next free number in the technical range, which is 1001; assign it first with make assign") {
		t.Errorf("got %v:\n%s", err, out.String())
	}
}
