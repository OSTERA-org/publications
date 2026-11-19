// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 OSTERA

package pub

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Public reviews live in docs/reviews/<shortname>/v<version>/<draft>/,
// where <version> is the version the draft is expected to become and
// <draft> is the draft number in two digits, the same number the draft
// file carries. The review of draft-ts-jcs-01.pdf for version 1.0 is in
// docs/reviews/jcs/v1.0/01/ and is served at
// https://pub.ostera.org/reviews/jcs/v1.0/01/. The version is part of the
// path because a Working Group may start its draft numbers again for each
// version, as its 03-Work-Items/<shortname>/v<version>/ folders allow.
// Each review folder holds review.yaml, the frozen copy of the draft in
// one file per format, SHA512SUMS, and its generated index.md.
const (
	reviewsDir   = docsDir + "/reviews"
	reviewsIndex = reviewsDir + "/index.md"
	reviewFile   = "review.yaml"

	// minReviewDays is the shortest public review the Working Methods
	// allow (Article 29): the closing date must be at least this many
	// days after the opening date.
	minReviewDays = 30

	reviewEmail = "public-review@ostera.org"
	wgOrgURL    = "https://github.com/OSTERA-org-Working-Groups"
)

// Review is docs/reviews/<shortname>/<draft>/review.yaml.
type Review struct {
	SchemaVersion int      `yaml:"schema_version"`
	Shortname     string   `yaml:"shortname"`
	Draft         int      `yaml:"draft"`
	Type          string   `yaml:"type"`
	Version       string   `yaml:"version"`
	Title         string   `yaml:"title"`
	WorkingGroup  string   `yaml:"working_group"`
	Opens         string   `yaml:"opens"`
	Closes        string   `yaml:"closes"`
	Formats       []string `yaml:"formats"`
	Summary       string   `yaml:"summary"`
	Changes       string   `yaml:"changes"`
	Withdrawn     string   `yaml:"withdrawn"`
	Disposition   string   `yaml:"disposition"`
}

// ReviewDir is one review folder, loaded.
type ReviewDir struct {
	Dir string // docs/reviews/<shortname>/<draft>, relative to the root
	R   *Review
}

func draftDir(n int) string { return fmt.Sprintf("%02d", n) }

// draftName is the file name of the draft in one format, for example
// draft-ts-jcs-01.pdf.
func draftName(r *Review, format string) string {
	return fmt.Sprintf("draft-%s-%s-%s.%s", strings.ToLower(r.Type), r.Shortname, draftDir(r.Draft), format)
}

// draftLabel is the draft name without a format, for example
// draft-ts-jcs-01.
func draftLabel(r *Review) string {
	return strings.TrimSuffix(draftName(r, "x"), ".x")
}

func expectedDraftFiles(r *Review) []string {
	var names []string
	for _, f := range r.Formats {
		names = append(names, draftName(r, f))
	}
	sort.Strings(names)
	return names
}

func reviewsIndexDir(short string) string { return reviewsDir + "/" + short }

// reviewsPath is the site path of the review history of a short name.
func reviewsPath(short string) string { return "/reviews/" + short }

func reviewPath(r *Review, parts ...string) string {
	return "/" + strings.Join(append([]string{"reviews", r.Shortname, versionDir(r.Version), draftDir(r.Draft)}, parts...), "/")
}

func parseReview(path string, data []byte, p *Problems) *Review {
	lintRaw(path, data, p)
	var r Review
	if !decodeStrict(path, data, &r, p) {
		return nil
	}
	return &r
}

// loadReviews reads every
// docs/reviews/<shortname>/v<version>/<draft>/review.yaml, oldest version
// and draft first within each short name. Misplaced files are reported by
// checkReviews.
func loadReviews(root string, p *Problems) []ReviewDir {
	shorts, err := os.ReadDir(filepath.Join(root, reviewsDir))
	if err != nil {
		return nil
	}
	var list []ReviewDir
	for _, s := range shorts {
		if !s.IsDir() {
			continue
		}
		versions, err := os.ReadDir(filepath.Join(root, reviewsDir, s.Name()))
		if err != nil {
			continue
		}
		for _, v := range versions {
			if !v.IsDir() {
				continue
			}
			drafts, err := os.ReadDir(filepath.Join(root, reviewsDir, s.Name(), v.Name()))
			if err != nil {
				continue
			}
			for _, d := range drafts {
				if !d.IsDir() {
					continue
				}
				rel := reviewsDir + "/" + s.Name() + "/" + v.Name() + "/" + d.Name()
				data, err := os.ReadFile(filepath.Join(root, rel, reviewFile))
				if err != nil {
					p.Addf(rel, "review directory has no %s", reviewFile)
					continue
				}
				if r := parseReview(rel+"/"+reviewFile, data, p); r != nil {
					list = append(list, ReviewDir{Dir: rel, R: r})
				}
			}
		}
	}
	sortReviews(list)
	return list
}

func sortReviews(list []ReviewDir) {
	sort.Slice(list, func(i, j int) bool {
		a, b := list[i].R, list[j].R
		if a.Shortname != b.Shortname {
			return a.Shortname < b.Shortname
		}
		if a.Version != b.Version {
			return versionLess(a.Version, b.Version)
		}
		return a.Draft < b.Draft
	})
}

// reviewsOf lists the reviews of one short name, oldest draft first.
func reviewsOf(s *site, short string) []ReviewDir {
	var out []ReviewDir
	for _, rv := range s.reviews {
		if rv.R.Shortname == short {
			out = append(out, rv)
		}
	}
	return out
}

func reviewShortnames(s *site) []string {
	seen := map[string]bool{}
	var out []string
	for _, rv := range s.reviews {
		if !seen[rv.R.Shortname] {
			seen[rv.R.Shortname] = true
			out = append(out, rv.R.Shortname)
		}
	}
	return out
}

func documentFor(s *site, short string) *Document {
	for i := range s.docs {
		if s.docs[i].Meta.Shortname == short {
			return &s.docs[i]
		}
	}
	return nil
}

func parseDate(s string) (time.Time, bool) {
	if !dateRE.MatchString(s) {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", s)
	return t, err == nil
}

// ---------- validation ----------

// checkReviews checks every review.yaml. With files set, it also checks
// the files on disk, as checkDocuments does.
func checkReviews(root string, s *site, files bool, p *Problems) {
	for _, rv := range s.reviews {
		checkReview(s, rv, p)
	}
	if files {
		checkReviewTree(root, s, p)
	}
}

func checkReview(s *site, rv ReviewDir, p *Problems) {
	r, path := rv.R, rv.Dir+"/"+reviewFile
	parts := strings.Split(rv.Dir, "/")
	folderShort, folderVersion, folderDraft := parts[len(parts)-3], parts[len(parts)-2], parts[len(parts)-1]

	if r.SchemaVersion != 1 {
		p.Addf(path, "schema_version must be 1, not %d", r.SchemaVersion)
	}
	switch {
	case !idRE.MatchString(r.Shortname):
		p.Addf(path, "shortname %q must be lowercase letters and digits, starting with a letter", r.Shortname)
	case contains(reservedShortnames, r.Shortname):
		p.Addf(path, "shortname %q is reserved for the site; reserved: %s", r.Shortname, strings.Join(reservedShortnames, ", "))
	case r.Shortname != folderShort:
		p.Addf(path, "shortname is %s, but the review is in docs/reviews/%s/", r.Shortname, folderShort)
	}
	if r.Draft < 1 {
		p.Addf(path, "draft must be the draft number, 1 or more")
	} else if draftDir(r.Draft) != folderDraft {
		p.Addf(path, "draft %d belongs in a folder named %s, not %s", r.Draft, draftDir(r.Draft), folderDraft)
	}
	if _, ok := s.reg.Types[r.Type]; !ok {
		p.Addf(path, "type %q is not defined in registry.yaml", r.Type)
	}
	if !versionRE.MatchString(r.Version) {
		p.Addf(path, "version %q must be the version the draft is expected to become, such as \"1.0\"", r.Version)
	} else if versionDir(r.Version) != folderVersion {
		p.Addf(path, "version %s belongs in a folder named %s, not %s", r.Version, versionDir(r.Version), folderVersion)
	}
	if strings.TrimSpace(r.Title) == "" {
		p.Addf(path, "title is required")
	}
	if strings.TrimSpace(r.Summary) == "" {
		p.Addf(path, "summary is required")
	}
	if _, ok := s.reg.WorkingGroups[r.WorkingGroup]; !ok {
		p.Addf(path, "working_group %q is not defined in registry.yaml", r.WorkingGroup)
	}
	if d := documentFor(s, r.Shortname); d != nil && d.Meta.WorkingGroup != "" && d.Meta.WorkingGroup != r.WorkingGroup {
		p.Addf(path, "working_group is %s, but the published document %s belongs to %s", r.WorkingGroup, r.Shortname, d.Meta.WorkingGroup)
	}

	opens, okOpen := parseDate(r.Opens)
	closes, okClose := parseDate(r.Closes)
	if !okOpen {
		p.Addf(path, "opens %q must be a date in yyyy-mm-dd form", r.Opens)
	}
	if !okClose {
		p.Addf(path, "closes %q must be a date in yyyy-mm-dd form", r.Closes)
	}
	if okOpen && okClose && closes.Before(opens.AddDate(0, 0, minReviewDays)) {
		p.Addf(path, "closes must be at least %d days after opens (Working Methods, Article 29); the earliest allowed is %s", minReviewDays, opens.AddDate(0, 0, minReviewDays).Format("2006-01-02"))
	}
	if r.Withdrawn != "" {
		if _, ok := parseDate(r.Withdrawn); !ok {
			p.Addf(path, "withdrawn %q must be a date in yyyy-mm-dd form", r.Withdrawn)
		}
	}
	if r.Disposition != "" && !strings.HasPrefix(r.Disposition, "https://") {
		p.Addf(path, "disposition must be an https URL")
	}

	if len(r.Formats) == 0 {
		p.Addf(path, "formats must list at least one format, such as [pdf]")
	}
	seen := map[string]bool{}
	for _, f := range r.Formats {
		if !formatRE.MatchString(f) {
			p.Addf(path, "format %q must be lowercase letters and digits", f)
		}
		if seen[f] {
			p.Addf(path, "format %s is listed more than once", f)
		}
		seen[f] = true
	}
}

// checkReviewTree checks that docs/reviews holds only what belongs there:
// index.md at the top, an index.md and review folders in each short name
// folder, and in each review folder its review.yaml, index.md,
// SHA512SUMS, and exactly the drafts review.yaml lists.
func checkReviewTree(root string, s *site, p *Problems) {
	top, err := os.ReadDir(filepath.Join(root, reviewsDir))
	if err != nil {
		if len(s.reviews) > 0 {
			p.Addf(reviewsDir, "cannot read: %v", err)
		}
		return
	}
	byDir := map[string]*Review{}
	for _, rv := range s.reviews {
		byDir[rv.Dir] = rv.R
	}
	for _, e := range top {
		rel := reviewsDir + "/" + e.Name()
		switch {
		case !e.IsDir() && e.Name() == pageFile:
		case !e.IsDir():
			p.Addf(rel, "does not belong in docs/reviews")
		default:
			entries, _ := os.ReadDir(filepath.Join(root, rel))
			count := 0
			for _, v := range entries {
				vrel := rel + "/" + v.Name()
				switch {
				case !v.IsDir() && v.Name() == pageFile:
				case !v.IsDir() || !strings.HasPrefix(v.Name(), "v"):
					p.Addf(vrel, "does not belong in a review history folder; reviews go in v<version>/<draft>/")
				default:
					drafts, _ := os.ReadDir(filepath.Join(root, vrel))
					for _, f := range drafts {
						frel := vrel + "/" + f.Name()
						switch {
						case f.IsDir() && byDir[frel] != nil:
							count++
							checkArtifactDir(root, frel, expectedDraftFiles(byDir[frel]), []string{pageFile, reviewFile}, p)
						case f.IsDir():
							// Reported by loadReviews (no review.yaml) or by a parse error.
						default:
							p.Addf(frel, "does not belong in a version folder of a review history; reviews go in <draft>/")
						}
					}
				}
			}
			if count == 0 {
				p.Addf(rel, "holds no reviews")
			} else if _, err := os.Stat(filepath.Join(root, rel, pageFile)); err != nil {
				p.Addf(rel, "%s is missing; run pubcheck generate", pageFile)
			}
		}
	}
	for _, rv := range s.reviews {
		if _, err := os.Stat(filepath.Join(root, rv.Dir, pageFile)); err != nil {
			p.Addf(rv.Dir, "%s is missing; run pubcheck generate", pageFile)
		}
	}
}

// ---------- history ----------

// checkReviewPath enforces that a review folder never changes once it is
// in the base branch, except for its review.yaml (checked field by field
// by compareReviews) and its generated index.md.
func checkReviewPath(root, mb string, c change, p *Problems) {
	parts := strings.Split(c.path, "/")
	if len(parts) != 6 {
		return
	}
	dir, name := strings.Join(parts[:5], "/"), parts[5]
	if !gitExists(root, mb, dir+"/"+reviewFile) {
		return // a new review; nothing in it is announced yet
	}
	if name == pageFile || (name == reviewFile && c.status == "M") {
		return
	}
	p.Addf(c.path, "is part of a public review that has been announced, so it cannot be added, changed, or deleted; publish the new draft as a new review instead")
}

// compareReviews checks every review.yaml in the base branch against the
// working tree. A review is never removed. Its entry only changes by
// moving the closing date later, by recording the date it was withdrawn,
// or by adding the link to the disposition of comments.
func compareReviews(root, mb string, p *Problems) error {
	out, err := git(root, "ls-tree", "-r", "--name-only", mb, "--", reviewsDir)
	if err != nil {
		return err
	}
	for _, path := range strings.Split(strings.TrimSpace(out), "\n") {
		if !strings.HasSuffix(path, "/"+reviewFile) || strings.Count(path, "/") != 5 {
			continue
		}
		data, _ := gitShow(root, mb, path)
		old := parseReview(path, data, &Problems{})
		if old == nil {
			continue
		}
		cur, ok := readFile(root, path)
		if !ok {
			p.Addf(path, "was removed, but a public review is part of the record and is never removed")
			continue
		}
		n := parseReview(path, cur, &Problems{})
		if n == nil {
			continue
		}
		if n.Closes < old.Closes {
			p.Addf(path, "closes moved from %s to %s; a review can be extended, but never shortened", old.Closes, n.Closes)
		}
		if old.Withdrawn != "" && n.Withdrawn != old.Withdrawn {
			p.Addf(path, "withdrawn was %s and cannot change", old.Withdrawn)
		}
		if old.Disposition != "" && n.Disposition != old.Disposition {
			p.Addf(path, "disposition was %s and cannot change", old.Disposition)
		}
		a, b := *old, *n
		a.Closes, b.Closes = "", ""
		a.Withdrawn, b.Withdrawn = "", ""
		a.Disposition, b.Disposition = "", ""
		if !reflect.DeepEqual(a, b) {
			p.Addf(path, "was edited; once a review is announced, only closes (to a later date), withdrawn, and disposition may change")
		}
	}
	return nil
}

// ---------- pages ----------

// statusSpan is the status of one review. The text is what a reader sees
// before the theme's script (assets/js/ostera.js in ostera-pages-theme)
// replaces it with Upcoming, Open, Closed, or Withdrawn, using the
// reader's own date. That is how a review opens and closes on its dates
// without anyone changing the site. The generated pages never depend on
// today's date, so validate gives the same answer every day.
func statusSpan(r *Review) string {
	text := fmt.Sprintf("Open from %s through %s", r.Opens, r.Closes)
	if r.Withdrawn != "" {
		text = "Withdrawn on " + r.Withdrawn
	}
	return fmt.Sprintf(`<span class="review-status" data-opens="%s" data-closes="%s" data-withdrawn="%s">%s</span>`, r.Opens, r.Closes, r.Withdrawn, text)
}

func reviewLink(r *Review) string {
	return fmt.Sprintf(`<a href="%s/">%s</a>`, reviewPath(r), draftLabel(r))
}

func wgName(s *site, id string) string {
	if wg, ok := s.reg.WorkingGroups[id]; ok {
		return wg.Name
	}
	return id
}

// stagesHTML draws the progress of a short name through the process, one
// list for each version its drafts were reviewed for, newest first. The
// steps follow Articles 29 and 30 of the Working Methods.
func stagesHTML(s *site, short string) string {
	reviews := reviewsOf(s, short)
	byVersion := map[string][]*Review{}
	var versions []string
	for _, rv := range reviews {
		if _, ok := byVersion[rv.R.Version]; !ok {
			versions = append(versions, rv.R.Version)
		}
		byVersion[rv.R.Version] = append(byVersion[rv.R.Version], rv.R)
	}
	sort.Slice(versions, func(i, j int) bool { return versionLess(versions[j], versions[i]) })

	doc := documentFor(s, short)
	var b strings.Builder
	w := func(format string, args ...any) { fmt.Fprintf(&b, format, args...) }
	for _, ver := range versions {
		list := byVersion[ver]
		last := list[len(list)-1]
		var pub *Version
		if doc != nil {
			for i := range doc.Meta.Versions {
				if doc.Meta.Versions[i].Version == ver {
					pub = &doc.Meta.Versions[i]
				}
			}
		}
		review, final := "current", "todo"
		if pub != nil {
			review, final = "done", "done"
		}
		if len(versions) > 1 {
			w("<h3>Version %s</h3>\n", ver)
		}
		w("<ol class=\"stages\">\n")
		w("<li class=\"stages__step stages__step--done\"><strong>Working Group Draft</strong><br>Developed in the <a href=\"https://wg.ostera.org/%s/work-items/\">%s</a></li>\n", last.WorkingGroup, wgName(s, last.WorkingGroup))
		w("<li class=\"stages__step stages__step--done\"><strong>Working Group Last Call</strong><br>14 days</li>\n")
		w("<li class=\"stages__step stages__step--review stages__step--%s\"><strong>Public Review</strong>", review)
		for _, r := range list {
			w("<br>%s: %s", reviewLink(r), statusSpan(r))
		}
		w("</li>\n")
		if pub != nil {
			w("<li class=\"stages__step stages__step--%s\"><strong>Approved and Published</strong><br><a href=\"/%s/%s/\">Version %s</a>, published %s", final, short, versionDir(pub.Version), pub.Version, pub.Published)
			if pub.Status != "published" {
				w(" (%s)", pub.Status)
			}
			w("</li>\n")
		} else {
			w("<li class=\"stages__step stages__step--%s\"><strong>Approved and Published</strong><br>Not yet</li>\n", final)
		}
		w("</ol>\n")
	}
	return b.String()
}

// versionLess compares two version strings numerically.
func versionLess(a, b string) bool {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y int
		if i < len(pa) {
			x, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			y, _ = strconv.Atoi(pb[i])
		}
		if x != y {
			return x < y
		}
	}
	return false
}

// reviewPage is the page of one review.
func reviewPage(s *site, rv ReviewDir) string {
	r := rv.R
	var b strings.Builder
	w := func(format string, args ...any) { fmt.Fprintf(&b, format, args...) }

	w("%s", frontMatter("Public Review of "+draftLabel(r), r.Title+", Version "+r.Version))
	w("<p class=\"review-banner\"><strong>Status:</strong> %s</p>\n\n", statusSpan(r))
	w("%s\n\n", strings.TrimSpace(r.Summary))

	w("* **Draft:** %s\n", draftLabel(r))
	w("* **Working group:** [%s](https://wg.ostera.org/%s/)\n", wgName(s, r.WorkingGroup), r.WorkingGroup)
	w("* **Expected document type:** %s\n", typeName(s.reg, r.Type))
	w("* **Expected version:** %s\n", r.Version)
	w("* **Review period:** %s through %s\n", r.Opens, r.Closes)
	if r.Withdrawn != "" {
		w("* **Withdrawn:** %s\n", r.Withdrawn)
	}
	list := reviewsOf(s, r.Shortname)
	for i, x := range list {
		if x.R == r && i > 0 {
			w("* **Previous review:** [%s](%s/)\n", draftLabel(list[i-1].R), reviewPath(list[i-1].R))
		}
		if x.R == r && i < len(list)-1 {
			w("* **Next review:** [%s](%s/)\n", draftLabel(list[i+1].R), reviewPath(list[i+1].R))
		}
	}
	if d := documentFor(s, r.Shortname); d != nil {
		w("* **Published document:** [%s](%s/)\n", d.Meta.Title, path(d.Meta))
	}
	w("* **All reviews of this work item:** [Review history](%s/)\n", reviewsPath(r.Shortname))

	w("\n## Progress\n\n")
	w("%s", stagesHTML(s, r.Shortname))

	if c := strings.TrimSpace(r.Changes); c != "" {
		w("\n## Changes Since the Previous Review\n\n%s\n", c)
	}

	w("\n## Files\n\n")
	for _, f := range r.Formats {
		name := draftName(r, f)
		w("* [%s](%s): `%s`\n", fileLabel("en", f), reviewPath(r, name), name)
	}
	w("* [%s](%s): checksums for the files above\n", sumsFile, reviewPath(r, sumsFile))
	w("\nThis draft is a work in progress. It has not been approved by OSTERA and must not be cited as an OSTERA publication. The files of a review never change; a revised draft gets a new review.\n")

	w("\n## How to Comment\n\n")
	w("Anyone may comment, whether or not they are an OSTERA Member. Comments are accepted from the opening date through the closing date. Either:\n\n")
	w("* open an issue in the [%s repository](%s/%s/issues/new/choose), or\n", wgName(s, r.WorkingGroup), wgOrgURL, r.WorkingGroup)
	w("* send an email to [%s](mailto:%s?subject=%s%%20for%%20version%%20%s).\n\n", reviewEmail, reviewEmail, draftLabel(r), r.Version)
	w("Please include the draft name and version (%s for version %s) and the section your comment is about. Anything you submit is a contribution under the [OSTERA IPR and Copyright Policy](https://ostera.org/policies/). By submitting it, you agree to the terms of that policy (Article 3). Do not submit anything confidential (Article 4).\n", draftLabel(r), r.Version)

	w("\n## Outcome\n\n")
	if r.Disposition != "" {
		w("The working group's response to the comments is at <%s>.\n", r.Disposition)
	} else {
		w("After the review closes, the working group addresses every comment before it can approve the work item (Working Methods, Article 30). The response to the comments will be linked here.\n")
	}

	w("\n## Verifying a File\n\n")
	w("Compute the SHA512 checksum of the downloaded file and compare it with the matching line in `%s`:\n\n", sumsFile)
	w("```\nshasum -a 512 %s\n```\n\n", draftName(r, r.Formats[0]))
	w("%s\n", githubLink(rv.Dir))
	return b.String()
}

// workItemPage is the review history of one short name.
func workItemPage(s *site, short string) string {
	list := reviewsOf(s, short)
	last := list[len(list)-1].R
	var b strings.Builder
	w := func(format string, args ...any) { fmt.Fprintf(&b, format, args...) }

	w("%s", frontMatter(last.Title, "Public review history"))
	w("* **Working group:** [%s](https://wg.ostera.org/%s/)\n", wgName(s, last.WorkingGroup), last.WorkingGroup)
	w("* **Latest draft reviewed:** [%s](%s/)\n", draftLabel(last), reviewPath(last))
	if d := documentFor(s, short); d != nil {
		w("* **Published document:** [%s](%s/)\n", d.Meta.Title, path(d.Meta))
	}

	w("\n## Progress\n\n")
	w("%s", stagesHTML(s, short))

	w("\n## Reviews\n\n")
	w("| Draft | Version | Review Period | Status |\n| --- | --- | --- | --- |\n")
	for i := len(list) - 1; i >= 0; i-- {
		r := list[i].R
		w("| [%s](%s/) | %s | %s through %s | %s |\n", draftLabel(r), reviewPath(r), r.Version, r.Opens, r.Closes, statusSpan(r))
	}
	w("\n[All public reviews](/reviews/)\n")
	return b.String()
}

// reviewsSection is the generated list on docs/reviews/index.md, newest
// first by opening date.
func reviewsSection(s *site) string {
	var b strings.Builder
	w := func(format string, args ...any) { fmt.Fprintf(&b, format, args...) }
	w("## All Public Reviews\n\n")
	if len(s.reviews) == 0 {
		w("No public reviews have been announced yet.\n")
		return b.String()
	}
	list := append([]ReviewDir(nil), s.reviews...)
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].R.Opens != list[j].R.Opens {
			return list[i].R.Opens > list[j].R.Opens
		}
		return draftLabel(list[i].R) < draftLabel(list[j].R)
	})
	w("| Draft | Title | Version | Working Group | Status |\n| --- | --- | --- | --- | --- |\n")
	for _, rv := range list {
		r := rv.R
		w("| [%s](%s/) | %s | %s | %s | %s |\n", draftLabel(r), reviewPath(r), cell(r.Title), r.Version, cell(wgName(s, r.WorkingGroup)), statusSpan(r))
	}
	return b.String()
}
