// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 OSTERA

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// validate runs every check against the repository at root and returns
// the problems found. Generated file checks run only when the registry and
// all metadata loaded, since generation needs them.
func validate(root string) *Problems {
	p := &Problems{}
	st := loadSite(root, p)
	if st == nil {
		return p
	}
	checkRegistry(st.reg, p)
	checkDocuments(root, st.reg, st.docs, true, p)
	checkReviews(root, st, true, p)
	if p.Len() == 0 {
		checkGenerated(root, st, p)
	}
	return p
}

// ---------- registry.yaml ----------

func checkRegistry(r *Registry, p *Problems) {
	const path = "registry.yaml"
	if r.SchemaVersion != 1 {
		p.Addf(path, "schema_version must be 1, not %d", r.SchemaVersion)
	}

	for code, t := range r.Types {
		if !typeRE.MatchString(code) {
			p.Addf(path, "type code %q must be two or three uppercase letters", code)
		}
		if t.Name == "" {
			p.Addf(path, "type %s has no name", code)
		}
		if !contains(tracks, t.Track) {
			p.Addf(path, "type %s has track %q; allowed: %s", code, t.Track, strings.Join(tracks, ", "))
		}
	}

	// Ranges: types must be defined, each type in exactly one range, and
	// ranges must not overlap.
	typeRange := map[string]string{}
	names := sortedKeys(r.Ranges)
	for _, name := range names {
		rg := r.Ranges[name]
		if rg.First < 1 {
			p.Addf(path, "range %s: first must be 1 or more", name)
		}
		if rg.Last != nil && *rg.Last < rg.First {
			p.Addf(path, "range %s: last is below first", name)
		}
		for _, t := range rg.Types {
			if _, ok := r.Types[t]; !ok {
				p.Addf(path, "range %s lists type %s, which is not defined under types", name, t)
			}
			if other, dup := typeRange[t]; dup {
				p.Addf(path, "type %s is listed in both range %s and range %s", t, other, name)
			}
			typeRange[t] = name
		}
	}
	for code := range r.Types {
		if _, ok := typeRange[code]; !ok {
			p.Addf(path, "type %s is not allowed in any range", code)
		}
	}
	for i, a := range names {
		for _, b := range names[i+1:] {
			if rangesOverlap(r.Ranges[a], r.Ranges[b]) {
				p.Addf(path, "ranges %s and %s overlap", a, b)
			}
		}
	}

	for id, wg := range r.WorkingGroups {
		if !idRE.MatchString(id) {
			p.Addf(path, "Working Group id %q must be lowercase letters and digits, starting with a letter", id)
		}
		if wg.Name == "" {
			p.Addf(path, "Working Group %s has no name", id)
		}
		if !contains(wgStates, wg.State) {
			p.Addf(path, "Working Group %s has state %q; allowed: %s", id, wg.State, strings.Join(wgStates, ", "))
		}
	}

	seenNum := map[int]bool{}
	seenShort := map[string]bool{}
	for _, n := range r.Numbers {
		label := fmt.Sprintf("number %d", n.Number)
		if seenNum[n.Number] {
			p.Addf(path, "%s appears more than once", label)
		}
		seenNum[n.Number] = true
		if seenShort[n.Shortname] {
			p.Addf(path, "short name %q appears more than once", n.Shortname)
		}
		seenShort[n.Shortname] = true
		if !idRE.MatchString(n.Shortname) {
			p.Addf(path, "%s: short name %q must be lowercase letters and digits, starting with a letter", label, n.Shortname)
		}
		if contains(reservedShortnames, n.Shortname) {
			p.Addf(path, "%s: short name %q is reserved for the site; reserved: %s", label, n.Shortname, strings.Join(reservedShortnames, ", "))
		}
		if !contains(numberStates, n.State) {
			p.Addf(path, "%s: state %q; allowed: %s", label, n.State, strings.Join(numberStates, ", "))
		}
		if !dateRE.MatchString(n.Assigned) {
			p.Addf(path, "%s: assigned %q must be a date in yyyy-mm-dd form", label, n.Assigned)
		}
		if rangeFor(r, n.Number) == "" {
			p.Addf(path, "%s is not inside any range", label)
		}
	}
}

func rangesOverlap(a, b Range) bool {
	const max = int(^uint(0) >> 1)
	aLast, bLast := max, max
	if a.Last != nil {
		aLast = *a.Last
	}
	if b.Last != nil {
		bLast = *b.Last
	}
	return a.First <= bLast && b.First <= aLast
}

// rangeFor returns the name of the range containing number, or "".
func rangeFor(r *Registry, number int) string {
	for _, name := range sortedKeys(r.Ranges) {
		rg := r.Ranges[name]
		if number >= rg.First && (rg.Last == nil || number <= *rg.Last) {
			return name
		}
	}
	return ""
}

// ---------- documents ----------

// checkDocuments checks every metadata.yaml against the registry and the
// schema rules. With files set, it also checks the files on disk; generate
// leaves that out, because it runs before the landing pages exist.
func checkDocuments(root string, r *Registry, docs []Document, files bool, p *Problems) {
	byShort := map[string]NumberEntry{}
	for _, n := range r.Numbers {
		byShort[n.Shortname] = n
	}
	byNumber := map[int]*Metadata{}
	for _, d := range docs {
		byNumber[d.Meta.Number] = d.Meta
	}

	// Registry entries against directories.
	for _, n := range r.Numbers {
		_, err := os.Stat(filepath.Join(root, docsDir, n.Shortname, "metadata.yaml"))
		has := err == nil
		switch {
		case n.State == "active" && !has:
			p.Addf("registry.yaml", "number %d (%s) is active but docs/%s/metadata.yaml does not exist", n.Number, n.Shortname, n.Shortname)
		case n.State != "active" && has:
			p.Addf("registry.yaml", "number %d (%s) has a document directory, so its state must be active, not %s", n.Number, n.Shortname, n.State)
		}
	}

	for _, d := range docs {
		checkDocument(root, r, d, byShort, byNumber, files, p)
	}
}

func checkDocument(root string, r *Registry, d Document, byShort map[string]NumberEntry, byNumber map[int]*Metadata, files bool, p *Problems) {
	m := d.Meta
	path := d.Dir + "/metadata.yaml"
	dirName := strings.TrimPrefix(d.Dir, docsDir+"/")

	// Identity.
	if m.SchemaVersion != 1 {
		p.Addf(path, "schema_version must be 1, not %d", m.SchemaVersion)
	}
	entry, ok := byShort[dirName]
	if !ok {
		p.Addf(path, "directory %s has no entry in registry.yaml", d.Dir)
	} else if entry.Number != m.Number {
		p.Addf(path, "number %d does not match registry.yaml, which assigns %d to %s", m.Number, entry.Number, dirName)
	}
	if m.Shortname != dirName {
		p.Addf(path, "shortname %q does not match its directory name %q", m.Shortname, dirName)
	}

	// Descriptive fields.
	if m.Title == "" {
		p.Addf(path, "title is required")
	}
	if m.Summary == "" {
		p.Addf(path, "summary is required")
	}
	if len(m.Keywords) == 0 {
		p.Addf(path, "keywords must list at least one keyword")
	}
	if !langRE.MatchString(m.AuthoritativeLanguage) {
		p.Addf(path, "authoritative_language %q must be a two letter ISO 639-1 code", m.AuthoritativeLanguage)
	}
	if !formatRE.MatchString(m.AuthoritativeFormat) {
		p.Addf(path, "authoritative_format %q must be a lowercase file extension, such as pdf", m.AuthoritativeFormat)
	}
	if want := docURL(m.Shortname); m.CanonicalURL != want {
		p.Addf(path, "canonical_url must be %s", want)
	}
	for _, o := range m.Obsoletes {
		if o.ID == "" || !strings.HasPrefix(o.URL, "https://") {
			p.Addf(path, "each obsoletes entry needs an id and an https url")
		}
	}
	for _, im := range m.Implementations {
		if im.Language == "" || !strings.HasPrefix(im.URL, "https://") {
			p.Addf(path, "each implementations entry needs a language and an https url")
		}
	}

	if len(m.Versions) == 0 {
		p.Addf(path, "versions must list at least one published version")
		return
	}

	// Versions.
	seenVersion := map[string]bool{}
	anyWG, anyPD, anySUP := false, false, false
	prevPublished := ""
	for i := range m.Versions {
		v := &m.Versions[i]
		label := fmt.Sprintf("version %q", v.Version)
		if !versionRE.MatchString(v.Version) {
			p.Addf(path, "%s must be two or three numbers separated by dots, with no leading zeros", label)
		}
		if seenVersion[v.Version] {
			p.Addf(path, "%s is listed more than once", label)
		}
		seenVersion[v.Version] = true

		if _, ok := r.Types[v.Type]; !ok {
			p.Addf(path, "%s: type %q is not defined in registry.yaml", label, v.Type)
		} else if rng := rangeFor(r, m.Number); rng != "" && !contains(r.Ranges[rng].Types, v.Type) {
			p.Addf(path, "%s: type %s is not allowed in the %s range that contains number %d", label, v.Type, rng, m.Number)
		}
		anyPD = anyPD || v.Type == "PD"
		anySUP = anySUP || v.Type == "SUP"

		if !contains(versionStatuses, v.Status) {
			p.Addf(path, "%s: status %q; allowed: %s", label, v.Status, strings.Join(versionStatuses, ", "))
		}
		if !dateRE.MatchString(v.Published) {
			p.Addf(path, "%s: published %q must be a date in yyyy-mm-dd form", label, v.Published)
		} else {
			if prevPublished != "" && v.Published < prevPublished {
				p.Addf(path, "%s is listed after a version published later; versions must be oldest first", label)
			}
			prevPublished = v.Published
		}
		if !contains(developedBys, v.DevelopedBy) {
			p.Addf(path, "%s: developed_by %q; allowed: %s", label, v.DevelopedBy, strings.Join(developedBys, ", "))
		}
		anyWG = anyWG || v.DevelopedBy == "working_group"
		if v.Type == "PD" && v.DevelopedBy != "board" && v.DevelopedBy != "secretariat" {
			p.Addf(path, "%s: a PD document must have developed_by board or secretariat", label)
		}
		checkList(path, label+": languages", v.Languages, langRE, "a two letter ISO 639-1 code", p)
		checkList(path, label+": formats", v.Formats, formatRE, "a lowercase file extension", p)
		if !contains(v.Languages, m.AuthoritativeLanguage) {
			p.Addf(path, "%s: languages must include the authoritative language %s", label, m.AuthoritativeLanguage)
		}
		if !contains(v.Formats, m.AuthoritativeFormat) {
			p.Addf(path, "%s: formats must include the authoritative format %s", label, m.AuthoritativeFormat)
		}
		for _, a := range append(append([]Person{}, v.Authors...), v.Editors...) {
			if a.Name == "" || a.Affiliation == "" {
				p.Addf(path, "%s: every author and editor needs a name and an affiliation", label)
			}
		}
		checkCorrections(path, label, v, p)
	}

	// Working Group and Policy/Process Document rules.
	if anyWG {
		if m.WorkingGroup == "" {
			p.Addf(path, "working_group is required because a version has developed_by: working_group")
		} else if _, ok := r.WorkingGroups[m.WorkingGroup]; !ok {
			p.Addf(path, "working_group %q is not defined in registry.yaml", m.WorkingGroup)
		}
	}
	if anyPD && m.WorkingGroup != "" {
		p.Addf(path, "a PD document is developed by the Association and must not have a working_group")
	}

	// Relationships between OSTERA documents.
	if anySUP {
		if m.Supplements == nil {
			p.Addf(path, "supplements must give the parent document number for a Supplement")
		} else if parent, ok := byNumber[*m.Supplements]; !ok {
			p.Addf(path, "supplements points to %d, which is not a published document", *m.Supplements)
		} else if !containsInt(parent.SupplementedBy, m.Number) {
			p.Addf(path, "parent document %d must list %d in its supplemented_by", *m.Supplements, m.Number)
		}
	} else if m.Supplements != nil {
		p.Addf(path, "supplements must be null, because this document is not a Supplement")
	}
	for _, n := range m.SupplementedBy {
		if child, ok := byNumber[n]; !ok {
			p.Addf(path, "supplemented_by lists %d, which is not a published document", n)
		} else if child.Supplements == nil || *child.Supplements != m.Number {
			p.Addf(path, "supplemented_by lists %d, but that document does not supplement %d", n, m.Number)
		}
	}
	for _, n := range m.Related {
		if _, ok := byNumber[n]; !ok {
			p.Addf(path, "related lists %d, which is not a published document", n)
		}
		if n == m.Number {
			p.Addf(path, "related must not list the document itself")
		}
	}

	if files {
		checkFiles(root, d, p)
	}
}

func checkCorrections(path, label string, v *Version, p *Problems) {
	next := map[string]int{"erratum": 1, "corrigendum": 1}
	for _, c := range v.Corrections {
		clabel := fmt.Sprintf("%s: %s %d", label, c.Kind, c.Sequence)
		if !contains(correctionKinds, c.Kind) {
			p.Addf(path, "%s: kind %q; allowed: %s", label, c.Kind, strings.Join(correctionKinds, ", "))
			continue
		}
		if c.Sequence != next[c.Kind] {
			p.Addf(path, "%s: expected sequence %d; each kind is numbered from 1 with no gaps", clabel, next[c.Kind])
		}
		next[c.Kind] = c.Sequence + 1
		if !dateRE.MatchString(c.Date) {
			p.Addf(path, "%s: date %q must be a date in yyyy-mm-dd form", clabel, c.Date)
		} else if dateRE.MatchString(v.Published) && c.Date < v.Published {
			p.Addf(path, "%s: dated before the version it corrects was published", clabel)
		}
		checkList(path, clabel+": languages", c.Languages, langRE, "a two letter ISO 639-1 code", p)
		checkList(path, clabel+": formats", c.Formats, formatRE, "a lowercase file extension", p)
	}
}

func checkList(path, label string, list []string, re interface{ MatchString(string) bool }, want string, p *Problems) {
	if len(list) == 0 {
		p.Addf(path, "%s must not be empty", label)
	}
	seen := map[string]bool{}
	for _, s := range list {
		if !re.MatchString(s) {
			p.Addf(path, "%s: %q must be %s", label, s, want)
		}
		if seen[s] {
			p.Addf(path, "%s: %q is listed twice", label, s)
		}
		seen[s] = true
	}
}

// checkFiles compares the files on disk with what metadata.yaml says
// must be there, and verifies every checksum.
func checkFiles(root string, d Document, p *Problems) {
	m := d.Meta
	abs := filepath.Join(root, d.Dir)
	entries, err := os.ReadDir(abs)
	if err != nil {
		p.Addf(d.Dir, "cannot read: %v", err)
		return
	}
	wantVersions := map[string]*Version{}
	for i := range m.Versions {
		wantVersions[versionDir(m.Versions[i].Version)] = &m.Versions[i]
	}
	hasCorrections := len(expectedCorrectionFiles(m)) > 0
	sawCorrections := false
	for _, e := range entries {
		name := e.Name()
		switch {
		case name == pageFile || name == "metadata.yaml":
			if e.IsDir() {
				p.Addf(d.Dir+"/"+name, "must be a file")
			}
		case name == "corrections" && e.IsDir():
			sawCorrections = true
		case e.IsDir() && wantVersions[name] != nil:
			// checked below
		case e.IsDir() && strings.HasPrefix(name, "v"):
			p.Addf(d.Dir+"/"+name, "version directory is not listed in metadata.yaml")
		default:
			p.Addf(d.Dir+"/"+name, "does not belong in a document directory")
		}
	}
	if _, err := os.Stat(filepath.Join(abs, pageFile)); err != nil {
		p.Addf(d.Dir, "%s is missing; run pubcheck generate", pageFile)
	}

	for dir, v := range wantVersions {
		rel := d.Dir + "/" + dir
		checkArtifactDir(root, rel, expectedVersionFiles(m, v), []string{pageFile}, p)
	}
	switch {
	case hasCorrections && !sawCorrections:
		p.Addf(d.Dir, "corrections/ is missing, but metadata.yaml lists corrections")
	case !hasCorrections && sawCorrections:
		p.Addf(d.Dir+"/corrections", "exists, but metadata.yaml lists no corrections")
	case hasCorrections:
		checkArtifactDir(root, d.Dir+"/corrections", expectedCorrectionFiles(m), nil, p)
	}
}

// checkArtifactDir checks that rel holds exactly the expected artifacts
// plus SHA512SUMS, and that SHA512SUMS is exactly right. A version
// directory also holds its generated landing page, and a review directory
// its page and review.yaml (other); these are not artifacts and are not
// listed in SHA512SUMS.
func checkArtifactDir(root, rel string, want []string, other []string, p *Problems) {
	abs := filepath.Join(root, rel)
	entries, err := os.ReadDir(abs)
	if err != nil {
		p.Addf(rel, "directory is missing")
		return
	}
	have := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if name == sumsFile || contains(other, name) {
			continue
		}
		if e.IsDir() {
			p.Addf(rel+"/"+name, "subdirectories are not allowed here")
			continue
		}
		have[name] = true
		if !contains(want, name) {
			if !filenameRE.MatchString(name) && !draftRE.MatchString(name) {
				p.Addf(rel+"/"+name, "does not match the filename grammar or the draft naming rule")
			} else {
				p.Addf(rel+"/"+name, "is not expected from metadata.yaml (check type, number, year, short name, version, language, and format)")
			}
		}
	}
	missing := false
	for _, w := range want {
		if !have[w] {
			p.Addf(rel, "missing %s", w)
			missing = true
		}
	}
	if missing {
		return
	}

	got, err := os.ReadFile(filepath.Join(abs, sumsFile))
	if err != nil {
		p.Addf(rel, "%s is missing; run pubcheck checksums %s", sumsFile, rel)
		return
	}
	expect, err := sumsContent(abs, want)
	if err != nil {
		p.Addf(rel, "cannot compute checksums: %v", err)
		return
	}
	if string(got) != expect {
		p.Addf(rel+"/"+sumsFile, "%s", describeSumsMismatch(string(got), expect))
	}
}

func describeSumsMismatch(got, want string) string {
	parse := func(s string) map[string]string {
		m := map[string]string{}
		for _, line := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
			if f := strings.SplitN(line, "  ", 2); len(f) == 2 {
				m[f[1]] = f[0]
			}
		}
		return m
	}
	g, w := parse(got), parse(want)
	var issues []string
	for name, sum := range w {
		switch gs, ok := g[name]; {
		case !ok:
			issues = append(issues, name+" is not listed")
		case gs != sum:
			issues = append(issues, name+" has the wrong digest")
		}
	}
	for name := range g {
		if _, ok := w[name]; !ok {
			issues = append(issues, name+" is listed but is not an expected artifact")
		}
	}
	sort.Strings(issues)
	if len(issues) == 0 {
		return "is not in canonical form (sorted by filename, digest, two spaces, filename); run pubcheck checksums"
	}
	return strings.Join(issues, "; ")
}

func containsInt(list []int, n int) bool {
	for _, x := range list {
		if x == n {
			return true
		}
	}
	return false
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
