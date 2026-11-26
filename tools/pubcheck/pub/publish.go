// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 OSTERA

package pub

import (
	"bufio"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Publish publishes one approved draft from its Working Group site. Given
// the short name of a work item, such as jcs, or the address of its page,
// such as https://wg.ostera.org/jss/work-items/jcs/, or of one of its
// draft pages, such as https://wg.ostera.org/jss/work-items/jcs/v1.0/02/,
// or nothing, in which case it asks for the short name, it:
//
//  1. finds the work item on wg.ostera.org, and reads its registry.json,
//  2. checks that the draft is ready: it is the latest draft of its
//     version, the version had a Working Group Last Call of at least 14 days
//     and a Public Review of at least 30 days, each closed and not
//     withdrawn, the working group approved it on or after its release,
//     and its number is reserved or assigned to this work item in
//     registry.yaml (or, when registry.yaml does not have it yet, it is the
//     next free number in its range, and is added),
//  3. asks for the publication date (the date in the document, which is
//     the last time the Working Group approved it, suggesting the version's
//     approved date) and the branch name,
//  4. creates the branch, downloads the PDF, and checks its SHA-512 sum
//     against the working group site,
//  5. writes docs/<shortname>/metadata.yaml and the number's state in
//     registry.yaml (adding the number when it is new), and runs update
//     and validate,
//  6. shows a summary, and when it is confirmed, commits only the files of
//     this publication.
//
// Only the PDF is published. The other formats stay on the working group
// site.
func Publish(root, address string, in io.Reader, out io.Writer, now time.Time) error {
	ask := newAsker(in, out)
	say := func(format string, args ...any) { fmt.Fprintf(out, format, args...) }

	p := &Problems{}
	reg := loadRegistry(root, p)
	if reg == nil {
		return fmt.Errorf("cannot read registry.yaml: fix it first")
	}

	// 1. The work item, from its Working Group site: from its short name,
	// or its page.
	target, item, err := workItemFor(address, reg, ask, true)
	if err != nil {
		return err
	}

	// 2. Is the draft ready?
	plan := planPublication(root, reg, target, item, now, p)
	if p.Len() > 0 {
		say("This draft cannot be published yet:\n\n")
		p.Print(out)
		return fmt.Errorf("%d problem(s) found; nothing was changed", p.Len())
	}

	say("Publishing %s, Version %s, from %s\n", plan.identifier, plan.version.Version, plan.draft.Name)
	say("  %s\n\n", target.draftPage(plan.version.Version, plan.draft.Draft))

	// 3. The publication date and the branch.
	date := ask.text("Publication date, the date in the document, which is the last time the Working Group approved it (yyyy-mm-dd)", plan.date)
	if !dateRE.MatchString(date) {
		return fmt.Errorf("%q is not a date in yyyy-mm-dd form; nothing was changed", date)
	}
	plan.date = date
	branch := ask.text("Branch name", fmt.Sprintf("publish-%s-v%s", target.id, plan.version.Version))
	if branch == "" || strings.ContainsAny(branch, " ~^:?*[\\") {
		return fmt.Errorf("%q is not a usable branch name; nothing was changed", branch)
	}

	paths := plan.paths()
	if dirty, err := git(root, append([]string{"status", "--porcelain", "--untracked-files=all", "--"}, paths...)...); err != nil {
		return err
	} else if strings.TrimSpace(dirty) != "" {
		return fmt.Errorf("these files already have changes; commit or discard them first, and nothing was changed:\n%s", dirty)
	}
	if _, err := git(root, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		return fmt.Errorf("branch %s already exists; choose another name, and nothing was changed", branch)
	}

	// 4. The branch, and the PDF.
	if _, err := git(root, "switch", "-c", branch); err != nil {
		return fmt.Errorf("cannot create branch %s: %v", branch, err)
	}
	say("created branch %s\n", branch)
	if err := plan.apply(root); err != nil {
		return err
	}
	pdfURL := target.fileURL(plan.version.Version, plan.pdf.Name)
	data, err := download(pdfURL)
	if err != nil {
		return err
	}
	sum := sha512.Sum512(data)
	if got := hex.EncodeToString(sum[:]); got != plan.pdf.Sha512 {
		return fmt.Errorf("%s does not match its SHA-512 sum on the working group site; nothing was committed", pdfURL)
	}
	pdfPath := filepath.Join(root, filepath.FromSlash(plan.pdfPath()))
	if err := os.MkdirAll(filepath.Dir(pdfPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(pdfPath, data, 0o644); err != nil {
		return err
	}
	say("downloaded %s, and its SHA-512 sum matches\n", plan.pdf.Name)

	// 5. Metadata, registry, pages, and validation.
	if err := plan.write(root); err != nil {
		return err
	}
	if _, up, err := Update(root); err != nil {
		return err
	} else if up.Len() > 0 {
		up.Print(out)
		return fmt.Errorf("update found %d problem(s); nothing was committed", up.Len())
	}
	base := ""
	if _, err := git(root, "rev-parse", "--verify", "--quiet", "main"); err == nil {
		base = "main"
	}
	vp, _, err := Validate(root, base, false)
	if err != nil {
		return err
	}

	// 6. Summary, and commit.
	changed, err := git(root, append([]string{"status", "--porcelain", "--untracked-files=all", "--"}, paths...)...)
	if err != nil {
		return err
	}
	var files []string
	for _, line := range strings.Split(strings.TrimRight(changed, "\n"), "\n") {
		if len(line) > 3 {
			files = append(files, strings.TrimSpace(line[3:]))
		}
	}
	sort.Strings(files)
	say("\n%s", plan.summary(files))
	if vp.Len() > 0 {
		say("\nvalidate found problems:\n")
		vp.Print(out)
		return fmt.Errorf("%d problem(s) found; nothing was committed, and the changes are on branch %s", vp.Len(), branch)
	}
	say("\nvalidate: the repository is valid\n\n")
	if !ask.yes(fmt.Sprintf("Commit these %d files on branch %s?", len(files), branch)) {
		say("\nNothing was committed. The changes are on branch %s. To discard them:\n\n", branch)
		say("  git restore -- %s\n  git clean -f -- %s\n  git switch main\n  git branch -D %s\n", strings.Join(paths, " "), strings.Join(paths, " "), branch)
		return nil
	}
	if _, err := git(root, append([]string{"add", "--"}, files...)...); err != nil {
		return err
	}
	message := fmt.Sprintf("Publish %s, Version %s (%s)", plan.identifier, plan.version.Version, strings.ToUpper(target.id))
	if _, err := git(root, append([]string{"commit", "-q", "-m", message, "--"}, files...)...); err != nil {
		return err
	}
	say("\ncommitted on branch %s: %s\n\n", branch, message)
	say("Next:\n")
	say("  1. git push -u %s %s, and open a pull request\n", remoteName(root), branch)
	say("  2. after it is merged, ask the chairs of the Working Group to record it with\n")
	say("     make published, for version %s of %s, published %s\n", plan.version.Version, target.id, plan.date)
	return nil
}

// ------------------------------------------------------------------------
// The working group site
// ------------------------------------------------------------------------

// workItemURL is the address the Secretariat gives: a work item page, and
// optionally one of its drafts.
type workItemURL struct {
	site, wg, id, version, draft string
}

var workItemURLRE = regexp.MustCompile(`^(https?://[^/]+)/([a-z][a-z0-9]*)/work-items/([a-z][a-z0-9]*)(?:/v([0-9]+(?:\.[0-9]+){1,2})/([0-9]{2,}))?/?(?:index\.html|registry\.json)?$`)

func parseWorkItemURL(address string) (workItemURL, error) {
	m := workItemURLRE.FindStringSubmatch(strings.TrimSpace(address))
	if m == nil {
		return workItemURL{}, fmt.Errorf("%q is not the address of a work item or a draft on a working group site, such as https://wg.ostera.org/jss/work-items/jcs/", address)
	}
	return workItemURL{site: m[1], wg: m[2], id: m[3], version: m[4], draft: m[5]}, nil
}

func (u workItemURL) page() string                 { return fmt.Sprintf("%s/%s/work-items/%s/", u.site, u.wg, u.id) }
func (u workItemURL) registryURL() string          { return u.page() + "registry.json" }
func (u workItemURL) draftPage(v, d string) string { return fmt.Sprintf("%sv%s/%s/", u.page(), v, d) }
func (u workItemURL) fileURL(v, name string) string {
	return fmt.Sprintf("%sv%s/%s", u.page(), v, name)
}

// wgItem is the registry.json of a work item, as its working group site
// publishes it.
type wgItem struct {
	Name     string      `json:"name"`
	Number   string      `json:"number"`
	Type     string      `json:"type"`
	Summary  string      `json:"summary"`
	Versions []wgVersion `json:"versions"`
}

type wgVersion struct {
	Version   string      `json:"version"`
	Started   string      `json:"started"`
	Approved  *wgApproval `json:"approved"`
	Published string      `json:"published"`
	Drafts    []wgDraft   `json:"drafts"`
}

// wgApproval is the draft of a version the working group approved, and
// when.
type wgApproval struct {
	Draft string `json:"draft"`
	Date  string `json:"date"`
}

type wgDraft struct {
	Draft        string        `json:"draft"`
	Name         string        `json:"name"`
	Released     string        `json:"released"`
	Authors      []Person      `json:"authors"`
	Editors      []Person      `json:"editors"`
	SupplementTo *ExternalRef  `json:"supplement_to"`
	Updates      []ExternalRef `json:"updates"`
	Obsoletes    []ExternalRef `json:"obsoletes"`
	Citation     string        `json:"citation"`
	Keywords     []string      `json:"keywords"`
	LastCall     *wgStep       `json:"last_call"`
	PublicReview *wgStep       `json:"public_review"`
	Files        []wgFile      `json:"files"`
}

type wgStep struct {
	Opens     string `json:"opens"`
	Closes    string `json:"closes"`
	Withdrawn string `json:"withdrawn"`
}

type wgFile struct {
	Name   string `json:"name"`
	Format string `json:"format"`
	Sha512 string `json:"sha512"`
	Folder bool   `json:"folder"`
}

var httpClient = &http.Client{Timeout: 60 * time.Second}

func download(url string) ([]byte, error) { return downloadWith(httpClient, url) }

func downloadWith(client *http.Client, url string) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cannot read %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func fetchWorkItem(url string) (*wgItem, error) {
	data, err := download(url)
	if err != nil {
		return nil, fmt.Errorf("%v (is the working group site up to date, and is this the right address?)", err)
	}
	var item wgItem
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, fmt.Errorf("cannot read %s: %v", url, err)
	}
	return &item, nil
}

// ------------------------------------------------------------------------
// The plan
// ------------------------------------------------------------------------

type publication struct {
	target     workItemURL
	item       *wgItem
	version    wgVersion
	draft      wgDraft
	pdf        wgFile
	number     int
	typ        string
	identifier string
	date       string
	stateFrom  string    // the number's state before, or "" when it is new
	allocated  string    // the allocated date of a new number
	meta       *Metadata // the metadata.yaml before, or nil for a new document
	superseded []string  // versions changed from published to superseded
}

var numberIDRE = regexp.MustCompile(`^OSTERA ([A-Z]{2,3}) ([0-9]{4,})$`)

// planPublication finds the draft to publish and checks that it is ready.
// Every problem is added to p.
func planPublication(root string, reg *Registry, t workItemURL, item *wgItem, now time.Time, p *Problems) *publication {
	src := t.registryURL()
	pl := &publication{target: t, item: item}

	if len(item.Versions) == 0 {
		p.Addf(src, "the work item has no versions")
		return pl
	}
	vi := len(item.Versions) - 1
	if t.version != "" {
		vi = -1
		for i, v := range item.Versions {
			if v.Version == t.version {
				vi = i
			}
		}
		if vi < 0 {
			p.Addf(src, "the work item has no version %s", t.version)
			return pl
		}
	}
	pl.version = item.Versions[vi]
	v := pl.version
	if v.Published != "" {
		p.Addf(src, "version %s was already published on %s", v.Version, v.Published)
	}
	if len(v.Drafts) == 0 {
		p.Addf(src, "version %s has no drafts", v.Version)
		return pl
	}
	pl.draft = v.Drafts[len(v.Drafts)-1]
	if t.draft != "" && t.draft != pl.draft.Draft {
		p.Addf(src, "draft %s is not the latest draft of version %s; the latest is draft %s, and only the latest draft can be published", t.draft, v.Version, pl.draft.Draft)
	}
	d := pl.draft
	label := fmt.Sprintf("%s (version %s, draft %s)", d.Name, v.Version, d.Draft)
	today := now.Format("2006-01-02")

	// Working Methods: at least one Working Group Last Call of at least 14
	// days and at least one Public Review of at least 30 days, each closed
	// and not withdrawn, on any draft of the version. Shorter ones may come
	// before them, so they are not counted, but they are not a problem.
	hadLastCall, hadPublicReview := false, false
	for _, x := range v.Drafts {
		if fullStep(x.LastCall, 14, today) {
			hadLastCall = true
		}
		if fullStep(x.PublicReview, 30, today) {
			hadPublicReview = true
		}
	}
	if !hadLastCall {
		p.Addf(src, "version %s has no Working Group Last Call of at least 14 days that has closed and was not withdrawn", v.Version)
	}
	if !hadPublicReview {
		p.Addf(src, "version %s has no Public Review of at least 30 days that has closed and was not withdrawn", v.Version)
	}

	// The approval: the working group must have approved this draft, the
	// latest of its version, on or after the day it was released. The Last
	// Call, the Public Review, and the approval can be in any order.
	switch {
	case v.Approved == nil || v.Approved.Date == "":
		p.Addf(src, "version %s has not been approved by the working group (approved: is empty; the chairs record it with make approved)", v.Version)
	case v.Approved.Draft != d.Draft:
		p.Addf(src, "version %s: the working group approved draft %s, not %s; the working group must approve the latest draft", v.Version, v.Approved.Draft, label)
	case d.Released != "" && v.Approved.Date < d.Released:
		p.Addf(src, "version %s was approved on %s, before %s was released on %s", v.Version, v.Approved.Date, label, d.Released)
	}

	// The PDF.
	for _, f := range d.Files {
		if !f.Folder && strings.EqualFold(f.Format, "pdf") {
			pl.pdf = f
		}
	}
	if pl.pdf.Name == "" || pl.pdf.Sha512 == "" {
		p.Addf(src, "%s has no PDF with a SHA-512 sum", label)
	}

	// The cover page fields that a publication needs.
	if len(d.Authors) == 0 && len(d.Editors) == 0 {
		p.Addf(src, "%s lists no authors and no editors", label)
	}
	if strings.TrimSpace(d.Citation) == "" {
		p.Addf(src, "%s has no citation", label)
	}
	if len(d.Keywords) == 0 {
		p.Addf(src, "%s has no keywords", label)
	}
	for _, list := range [][]ExternalRef{d.Updates, d.Obsoletes} {
		for _, r := range list {
			if !strings.HasPrefix(r.URL, "https://") {
				p.Addf(src, "%s: %s needs a url starting with https:// to be published", label, r.ID)
			}
		}
	}

	// The number.
	m := numberIDRE.FindStringSubmatch(item.Number)
	if m == nil {
		p.Addf(src, "the work item has no document number; ask the chairs to add it to the registry")
		return pl
	}
	pl.typ = m[1]
	pl.number, _ = strconv.Atoi(m[2])
	pl.identifier = identifier(pl.typ, pl.number)
	if pl.typ != item.Type {
		p.Addf(src, "the number %s does not match the type %s", item.Number, item.Type)
	}
	found := false
	for _, e := range reg.Numbers {
		if e.Number == pl.number {
			found = true
			pl.stateFrom = e.State
			if e.Shortname != t.id {
				p.Addf("registry.yaml", "number %d belongs to %s, not to %s", pl.number, e.Shortname, t.id)
			}
			if e.State == "retired" {
				p.Addf("registry.yaml", "number %d is retired", pl.number)
			}
		} else if e.Shortname == t.id {
			p.Addf("registry.yaml", "%s has number %d in registry.yaml, but the work item gives %s", t.id, e.Number, item.Number)
		}
	}
	if !found {
		// A number the Secretariat gave the chairs but did not record. It
		// can be added now, but only when it is the next free number.
		switch rangeName, next, err := nextNumber(reg, pl.typ); {
		case err != nil:
			p.Addf("registry.yaml", "number %d is not in registry.yaml, and %v", pl.number, strings.TrimSuffix(err.Error(), "; nothing was changed"))
		case next != pl.number:
			p.Addf("registry.yaml", "number %d is not in registry.yaml, and it is not the next free number in the %s range, which is %d; assign it first with make assign", pl.number, rangeName, next)
		default:
			pl.allocated = today
		}
	}
	if _, ok := reg.WorkingGroups[t.wg]; !ok {
		p.Addf("registry.yaml", "Working Group %s is not listed under working_groups; add it first with make add-wg", t.wg)
	}
	if d.SupplementTo != nil {
		if numberIDRE.FindStringSubmatch(d.SupplementTo.ID) == nil {
			p.Addf(src, "%s: supplement_to must be an OSTERA document, such as \"OSTERA TS 1001\", to be published", label)
		}
	}

	// The document, if it was published before.
	mpath := docsDir + "/" + t.id + "/metadata.yaml"
	if data, err := os.ReadFile(filepath.Join(root, mpath)); err == nil {
		pl.meta = parseMetadata(mpath, data, p)
		if pl.meta != nil {
			for _, x := range pl.meta.Versions {
				if x.Version == v.Version {
					p.Addf(mpath, "version %s is already published", v.Version)
				}
			}
		}
	}

	// The publication date: the date the working group last approved it,
	// which is also the date in the document, or else the date in the
	// citation, or else today.
	if v.Approved != nil {
		pl.date = v.Approved.Date
	}
	if pl.date == "" {
		pl.date = citationDate(d.Citation)
	}
	if pl.date == "" {
		pl.date = today
	}
	return pl
}

// fullStep reports whether a Working Group Last Call or Public Review lasted
// at least days days, has closed before today, and was not withdrawn.
func fullStep(r *wgStep, days int, today string) bool {
	return r != nil && r.Withdrawn == "" && r.Closes != "" && r.Closes < today && daysBetween(r.Opens, r.Closes) >= days
}

func daysBetween(opens, closes string) int {
	a, err1 := time.Parse("2006-01-02", opens)
	b, err2 := time.Parse("2006-01-02", closes)
	if err1 != nil || err2 != nil {
		return 0
	}
	return int(b.Sub(a).Hours()/24) + 1
}

var citationDateRE = regexp.MustCompile(`\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Sept|Oct|Nov|Dec)[a-z]*\.? ([0-9]{1,2}), ([0-9]{4})\b`)

// citationDate reads a date such as "Nov. 24, 2026" from an IEEE citation.
func citationDate(citation string) string {
	m := citationDateRE.FindStringSubmatch(citation)
	if m == nil {
		return ""
	}
	month := m[1]
	if month == "Sept" {
		month = "Sep"
	}
	t, err := time.Parse("Jan 2 2006", month+" "+m[2]+" "+m[3])
	if err != nil {
		return ""
	}
	return t.Format("2006-01-02")
}

// newVersion is the version entry this publication adds.
func (pl *publication) newVersion() Version {
	d := pl.draft
	return Version{
		Version:     pl.version.Version,
		Type:        pl.typ,
		Status:      "published",
		Published:   pl.date,
		DevelopedBy: "working_group",
		Languages:   []string{"en"},
		Formats:     []string{"pdf"},
		Authors:     d.Authors,
		Editors:     d.Editors,
		Citation:    strings.Join(strings.Fields(d.Citation), " "),
		Corrections: []Correction{},
	}
}

// metadata returns the metadata.yaml after this publication.
func (pl *publication) metadata() *Metadata {
	d := pl.draft
	m := &Metadata{
		SchemaVersion:         1,
		AuthoritativeLanguage: "en",
		AuthoritativeFormat:   "pdf",
	}
	if pl.meta != nil {
		copied := *pl.meta
		m = &copied
		m.Versions = append([]Version{}, pl.meta.Versions...)
	}
	m.Number = pl.number
	m.Shortname = pl.target.id
	m.Title = pl.item.Name
	m.WorkingGroup = pl.target.wg
	m.Summary = strings.Join(strings.Fields(pl.item.Summary), " ")
	m.CanonicalURL = docURL(pl.target.id)
	m.Keywords = append([]string{}, d.Keywords...)
	sort.Slice(m.Keywords, func(i, j int) bool { return strings.ToLower(m.Keywords[i]) < strings.ToLower(m.Keywords[j]) })
	m.Obsoletes = d.Obsoletes
	m.Updates = d.Updates
	m.Supplements = nil
	if d.SupplementTo != nil {
		if x := numberIDRE.FindStringSubmatch(d.SupplementTo.ID); x != nil {
			n, _ := strconv.Atoi(x[2])
			m.Supplements = &n
		}
	}
	pl.superseded = nil
	for i := range m.Versions {
		if m.Versions[i].Status == "published" {
			m.Versions[i].Status = "superseded"
			pl.superseded = append(pl.superseded, m.Versions[i].Version)
		}
	}
	m.Versions = append(m.Versions, pl.newVersion())
	return m
}

func (pl *publication) pdfPath() string {
	v := pl.newVersion()
	return docsDir + "/" + pl.target.id + "/" + versionDir(v.Version) + "/" + artifactName(pl.metadata(), &v, "", 0, "en", "pdf")
}

// paths are the files and folders this publication may change. Before it
// starts, none of them may have changes; afterward, the changed ones are
// exactly what it commits.
func (pl *publication) paths() []string {
	return []string{"registry.yaml", docsDir + "/" + pl.target.id, docsDir + "/catalog.json", docsDir + "/index.md", docsDir + "/about.md"}
}

// apply makes the version folder, so the PDF can be written into it.
func (pl *publication) apply(root string) error {
	return os.MkdirAll(filepath.Join(root, docsDir, pl.target.id, versionDir(pl.version.Version)), 0o755)
}

// write writes metadata.yaml, and the number's state in registry.yaml.
func (pl *publication) write(root string) error {
	mpath := filepath.Join(root, docsDir, pl.target.id, "metadata.yaml")
	if err := os.WriteFile(mpath, []byte(formatMetadata(pl.metadata())), 0o644); err != nil {
		return err
	}
	if pl.stateFrom == "active" {
		return nil
	}
	rpath := filepath.Join(root, "registry.yaml")
	data, err := os.ReadFile(rpath)
	if err != nil {
		return err
	}
	var updated []byte
	if pl.stateFrom == "" {
		entry := fmt.Sprintf("  - number: %d\n    shortname: %s\n    state: active\n    allocated: %s\n", pl.number, pl.target.id, pl.allocated)
		updated, err = insertEntry(data, "numbers", entry, "[]")
	} else {
		updated, err = setNumberState(data, pl.number, pl.stateFrom, "active")
	}
	if err != nil {
		return err
	}
	return os.WriteFile(rpath, updated, 0o644)
}

// summary describes the publication, for the Secretariat to check before
// it is committed.
func (pl *publication) summary(files []string) string {
	var b strings.Builder
	w := func(format string, args ...any) { fmt.Fprintf(&b, format, args...) }
	m := pl.metadata()
	v := m.Versions[len(m.Versions)-1]
	w("Summary\n")
	w("  Document:      %s, %s\n", pl.identifier, m.Title)
	w("  Version:       %s, published %s\n", v.Version, v.Published)
	w("  From draft:    %s, released %s\n", pl.draft.Name, pl.draft.Released)
	w("  File:          %s\n", pl.pdfPath())
	if len(v.Authors) > 0 {
		w("  Authors:       %s\n", people(v.Authors))
	}
	if len(v.Editors) > 0 {
		w("  Editors:       %s\n", people(v.Editors))
	}
	if m.Supplements != nil {
		w("  Supplement To: %d\n", *m.Supplements)
	}
	for _, x := range []struct {
		label string
		refs  []ExternalRef
	}{{"Updates", m.Updates}, {"Obsoletes", m.Obsoletes}} {
		if len(x.refs) > 0 {
			var s []string
			for _, r := range x.refs {
				s = append(s, r.ID)
			}
			w("  %-14s %s\n", x.label+":", strings.Join(s, ", "))
		}
	}
	w("  Keywords:      %s\n", strings.Join(m.Keywords, ", "))
	w("  Citation:      %s\n", v.Citation)
	switch pl.stateFrom {
	case "active":
	case "":
		w("  registry.yaml: number %d is added for %s, as active, allocated %s\n", pl.number, pl.target.id, pl.allocated)
	default:
		w("  registry.yaml: number %d changes from %s to active\n", pl.number, pl.stateFrom)
	}
	for _, s := range pl.superseded {
		w("  Version %s changes from published to superseded\n", s)
	}
	if pl.meta == nil {
		w("  This is the first version of this document, so metadata.yaml is new.\n")
	}
	w("\nFiles to commit:\n")
	for _, f := range files {
		w("  %s\n", f)
	}
	return b.String()
}

// ------------------------------------------------------------------------
// Questions, and git
// ------------------------------------------------------------------------

type asker struct {
	in  *bufio.Reader
	out io.Writer
}

func newAsker(in io.Reader, out io.Writer) *asker { return &asker{bufio.NewReader(in), out} }

// text asks a question with a default, and returns the answer, or the
// default when the answer is empty.
func (a *asker) text(question, def string) string {
	if def == "" {
		fmt.Fprintf(a.out, "%s: ", question)
	} else {
		fmt.Fprintf(a.out, "%s [%s]: ", question, def)
	}
	line, _ := a.in.ReadString('\n')
	if line = strings.TrimSpace(line); line != "" {
		return line
	}
	return def
}

// yesDefault asks a yes or no question, with def as the answer when the
// answer is empty. Anything else that is not yes is no.
func (a *asker) yesDefault(question string, def bool) bool {
	d := "no"
	if def {
		d = "yes"
	}
	fmt.Fprintf(a.out, "%s (yes or no) [%s]: ", question, d)
	line, _ := a.in.ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	if line == "" {
		return def
	}
	return line == "y" || line == "yes"
}

// yes asks a yes or no question; anything but yes is no.
func (a *asker) yes(question string) bool {
	fmt.Fprintf(a.out, "%s (yes or no) [no]: ", question)
	line, _ := a.in.ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes"
}

// remoteName is the name of the git remote for the publications
// repository on GitHub, or "origin".
func remoteName(root string) string {
	out, err := git(root, "remote", "-v")
	if err != nil {
		return "origin"
	}
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 && strings.Contains(strings.ToLower(f[1]), "ostera-org/publications") {
			return f[0]
		}
	}
	return "origin"
}
