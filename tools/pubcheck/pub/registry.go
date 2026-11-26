// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 OSTERA

package pub

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The commands in this file make the everyday changes to registry.yaml, so
// that nobody needs to edit it by hand: adding a Working Group, and
// reserving or assigning a document number. Each one asks its questions
// with a suggested answer (press Enter to accept it), shows what it will
// change and asks before it changes anything, edits registry.yaml line by
// line so its comments are kept, reads it back to check the change, and
// then runs update and validate. None of them uses git; they print the git
// steps instead.

// wgConfigURL is where the name of a Working Group is read from when the
// list on wg.ostera.org does not have it: the short_title in the website
// settings of its repository.
var wgConfigURL = "https://raw.githubusercontent.com/OSTERA-org-Working-Groups/%s/main/06-Website/_config.yml"

var shortTitleRE = regexp.MustCompile(`(?m)^short_title:\s*"?([^"\n]+?)"?\s*$`)

// AddWorkingGroup adds a Working Group to working_groups in registry.yaml.
// id may be empty, and then it is asked for.
func AddWorkingGroup(root, id string, in io.Reader, out io.Writer) error {
	ask := newAsker(in, out)
	say := func(format string, args ...any) { fmt.Fprintf(out, format, args...) }
	reg, data, err := readRegistry(root)
	if err != nil {
		return err
	}

	if id == "" {
		id = ask.text("Working Group id, as in https://wg.ostera.org/<id>/, such as jss", "")
	}
	if err := checkWGID(reg, id); err != nil {
		return err
	}
	name, err := askWGName(ask, id)
	if err != nil {
		return err
	}

	entry := wgEntry(id, name)
	say("\nThis adds to working_groups in registry.yaml:\n\n%s\n", entry)
	if !ask.yesDefault("Add it?", true) {
		say("Nothing was changed.\n")
		return nil
	}
	updated, err := insertEntry(data, "working_groups", entry, "{}")
	if err != nil {
		return err
	}
	if err := writeRegistry(root, updated, func(r *Registry) bool { return wgIs(r, id, name) }); err != nil {
		return err
	}
	say("added Working Group %s to registry.yaml\n", id)
	if err := updateAndValidate(root, out); err != nil {
		return err
	}
	nextSteps(root, out, "add-wg-"+id, "Add Working Group "+strings.ToUpper(id))
	return nil
}

func checkWGID(reg *Registry, id string) error {
	switch {
	case id == "":
		return fmt.Errorf("no Working Group id was given; nothing was changed")
	case !idRE.MatchString(id):
		return fmt.Errorf("%q is not a Working Group id: use lowercase letters and digits, starting with a letter; nothing was changed", id)
	}
	if wg, ok := reg.WorkingGroups[id]; ok {
		return fmt.Errorf("Working Group %s is already in registry.yaml, as %s (%s); nothing was changed", id, wg.Name, wg.State)
	}
	return nil
}

// askWGName asks for the name of a Working Group, suggesting it from
// wg.ostera.org.
func askWGName(ask *asker, id string) (string, error) {
	suggested := wgNameFor(id)
	if suggested == "" {
		fmt.Fprintf(ask.out, "(The name of %s could not be found on wg.ostera.org or in the website settings of its repository, so there is no suggestion.)\n", id)
	}
	name := ask.text("Name of Working Group "+id+", such as \"JSON Signature Scheme (JSS)\"", suggested)
	if name == "" {
		return "", fmt.Errorf("no name was given; nothing was changed")
	}
	return name, nil
}

func wgEntry(id, name string) string {
	return fmt.Sprintf("  %s:\n    name: %s\n    state: active\n", id, yamlScalar(name))
}

func wgIs(r *Registry, id, name string) bool {
	wg, ok := r.WorkingGroups[id]
	return ok && wg.Name == name && wg.State == "active"
}

// AllocateNumber reserves or assigns a document number in registry.yaml.
// state is "reserved" or "assigned". arg is the short name of the work
// item, such as jcs, or the address of its page, or empty, and then the
// short name is asked for. Its type, and its Working Group, are suggested
// from wg.ostera.org. When the Working Group is not in registry.yaml yet,
// it offers to add it in the same change. When state is "assigned" and the
// work item already has a reserved number, that number is changed to
// assigned, and keeps its allocated date.
func AllocateNumber(root, state, arg string, in io.Reader, out io.Writer, now time.Time) error {
	ask := newAsker(in, out)
	say := func(format string, args ...any) { fmt.Fprintf(out, format, args...) }
	reg, data, err := readRegistry(root)
	if err != nil {
		return err
	}
	verb := map[string]string{"reserved": "Reserve", "assigned": "Assign"}[state]
	if verb == "" {
		return fmt.Errorf("unknown state %q", state)
	}
	today := now.Format("2006-01-02")

	// The work item, looked up on wg.ostera.org.
	target, item, err := workItemFor(arg, reg, ask, false)
	if err != nil {
		return err
	}
	short := target.id
	switch {
	case !idRE.MatchString(short):
		return fmt.Errorf("%q is not a short name: use lowercase letters and digits, starting with a letter; nothing was changed", short)
	case contains(reservedShortnames, short):
		return fmt.Errorf("%q is reserved for the site and cannot be a short name (reserved: %s); nothing was changed", short, strings.Join(reservedShortnames, ", "))
	}
	typ := ""
	if item != nil {
		typ = item.Type
	}

	// A number the work item already has.
	for _, e := range reg.Numbers {
		if e.Shortname != short {
			continue
		}
		if state == "assigned" && e.State == "reserved" {
			say("Number %d is reserved for %s, allocated %s.\n", e.Number, short, e.Allocated)
			if !ask.yesDefault(fmt.Sprintf("Change number %d to assigned? Its allocated date stays %s.", e.Number, e.Allocated), true) {
				say("Nothing was changed.\n")
				return nil
			}
			updated, err := setNumberState(data, e.Number, "reserved", "assigned")
			if err != nil {
				return err
			}
			if err := writeRegistry(root, updated, func(r *Registry) bool { return numberIs(r, e.Number, short, "assigned", e.Allocated) }); err != nil {
				return err
			}
			say("number %d is now assigned to %s\n", e.Number, short)
			return finishAllocation(root, out, state, short, e.Number, typ, "")
		}
		return fmt.Errorf("%s already has number %d, which is %s; nothing was changed", short, e.Number, e.State)
	}

	// The type, which decides the range the number comes from.
	codes := orderedTypes(reg)
	suggestedType := typ
	if _, ok := reg.Types[suggestedType]; !ok {
		suggestedType = "TS"
		if _, ok := reg.Types[suggestedType]; !ok && len(codes) > 0 {
			suggestedType = codes[0]
		}
	}
	typ = strings.ToUpper(ask.text("Type code ("+strings.Join(codes, ", ")+")", suggestedType))
	if _, ok := reg.Types[typ]; !ok {
		return fmt.Errorf("%q is not a type code in registry.yaml (%s); nothing was changed", typ, strings.Join(codes, ", "))
	}
	rangeName, number, err := nextNumber(reg, typ)
	if err != nil {
		return err
	}
	say("Number: %d, the next free number in the %s range\n", number, rangeName)
	if item != nil && item.Number != "" && item.Number != identifier(typ, number) {
		say("note: the work item already gives %s on its page. The chairs need to change it to %s.\n", item.Number, identifier(typ, number))
	}

	allocated := ask.text("Date allocated (yyyy-mm-dd)", today)
	switch {
	case !validDate(allocated):
		return fmt.Errorf("%q is not a date in yyyy-mm-dd form; nothing was changed", allocated)
	case allocated > today:
		return fmt.Errorf("%s is in the future; nothing was changed", allocated)
	}

	// Its Working Group, when registry.yaml does not have it yet.
	wgID, wgName := "", ""
	if target.wg != "" && item != nil {
		if _, ok := reg.WorkingGroups[target.wg]; !ok {
			say("Working Group %s is not in registry.yaml yet.\n", target.wg)
			if ask.yesDefault("Add it in the same change?", true) {
				if wgName, err = askWGName(ask, target.wg); err != nil {
					return err
				}
				wgID = target.wg
			}
		}
	}

	entry := fmt.Sprintf("  - number: %d\n    shortname: %s\n    state: %s\n    allocated: %s\n", number, short, state, allocated)
	say("\nThis adds to numbers in registry.yaml (%s):\n\n%s\n", identifier(typ, number), entry)
	if wgID != "" {
		say("and to working_groups:\n\n%s\n", wgEntry(wgID, wgName))
	}
	if !ask.yesDefault(fmt.Sprintf("%s number %d?", verb, number), true) {
		say("Nothing was changed.\n")
		return nil
	}
	updated := data
	if wgID != "" {
		if updated, err = insertEntry(updated, "working_groups", wgEntry(wgID, wgName), "{}"); err != nil {
			return err
		}
	}
	if updated, err = insertEntry(updated, "numbers", entry, "[]"); err != nil {
		return err
	}
	if err := writeRegistry(root, updated, func(r *Registry) bool {
		return numberIs(r, number, short, state, allocated) && (wgID == "" || wgIs(r, wgID, wgName))
	}); err != nil {
		return err
	}
	say("%s number %d for %s in registry.yaml\n", state, number, short)
	if wgID != "" {
		say("added Working Group %s to registry.yaml\n", wgID)
	}
	return finishAllocation(root, out, state, short, number, typ, wgID)
}

// finishAllocation runs update and validate, and prints what comes next.
func finishAllocation(root string, out io.Writer, state, short string, number int, typ, wgID string) error {
	say := func(format string, args ...any) { fmt.Fprintf(out, format, args...) }
	if err := updateAndValidate(root, out); err != nil {
		return err
	}
	also := ""
	if wgID != "" {
		also = ", and add Working Group " + strings.ToUpper(wgID)
	}
	if state == "assigned" {
		id := fmt.Sprintf("OSTERA <type> %d", number)
		if typ != "" {
			id = identifier(typ, number)
		}
		say("\nAfter it is merged, tell the chairs the number, %s, so they can put it on the final draft. make add-draft asks for it.\n", id)
		nextSteps(root, out, "assign-"+short+"-number", fmt.Sprintf("Assign number %d to %s%s", number, strings.ToUpper(short), also))
		return nil
	}
	nextSteps(root, out, "reserve-"+short+"-number", fmt.Sprintf("Reserve number %d for %s%s", number, strings.ToUpper(short), also))
	return nil
}

// nextNumber returns the range a type belongs to, and the next number in
// it: the highest number ever allocated in the range, including retired
// numbers, plus one, or the first number of the range.
func nextNumber(reg *Registry, typ string) (string, int, error) {
	for _, name := range sortedKeys(reg.Ranges) {
		rg := reg.Ranges[name]
		if !contains(rg.Types, typ) {
			continue
		}
		next := rg.First
		for _, e := range reg.Numbers {
			if e.Number >= rg.First && (rg.Last == nil || e.Number <= *rg.Last) && e.Number >= next {
				next = e.Number + 1
			}
		}
		if rg.Last != nil && next > *rg.Last {
			return name, 0, fmt.Errorf("the %s range is full (%d to %d); nothing was changed", name, rg.First, *rg.Last)
		}
		return name, next, nil
	}
	return "", 0, fmt.Errorf("type %s is not in any range in registry.yaml; nothing was changed", typ)
}

func validDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil && dateRE.MatchString(s)
}

func numberIs(r *Registry, number int, short, state, allocated string) bool {
	for _, e := range r.Numbers {
		if e.Number == number {
			return e.Shortname == short && e.State == state && e.Allocated == allocated
		}
	}
	return false
}

// ------------------------------------------------------------------------
// Editing registry.yaml
// ------------------------------------------------------------------------

func readRegistry(root string) (*Registry, []byte, error) {
	data, err := os.ReadFile(filepath.Join(root, "registry.yaml"))
	if err != nil {
		return nil, nil, err
	}
	p := &Problems{}
	reg := parseRegistry("registry.yaml", data, p)
	if reg == nil || p.Len() > 0 {
		var b strings.Builder
		p.Print(&b)
		return nil, nil, fmt.Errorf("registry.yaml has problems; fix them first, and nothing was changed:\n%s", b.String())
	}
	return reg, data, nil
}

// writeRegistry writes registry.yaml, reads it back, and checks the change
// with ok. If the check fails, the old file is put back.
func writeRegistry(root string, data []byte, ok func(*Registry) bool) error {
	path := filepath.Join(root, "registry.yaml")
	old, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	p := &Problems{}
	if reg := parseRegistry("registry.yaml", data, p); reg == nil || p.Len() > 0 || !ok(reg) {
		os.WriteFile(path, old, 0o644)
		return fmt.Errorf("the change to registry.yaml could not be checked, so it was undone; nothing was changed")
	}
	return nil
}

var topKeyRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*:`)

// insertEntry adds entry at the end of the top level key in data. When the
// key's value is empty (empty, such as "{}" or "[]"), entry replaces it.
// Comments in the file are kept.
func insertEntry(data []byte, key, entry, empty string) ([]byte, error) {
	lines := strings.SplitAfter(string(data), "\n")
	start := -1
	for i, l := range lines {
		if strings.HasPrefix(l, key+":") {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("registry.yaml has no %s: key; nothing was changed", key)
	}
	value := strings.TrimSpace(strings.TrimPrefix(lines[start], key+":"))
	if i := strings.Index(value, " #"); i >= 0 {
		value = strings.TrimSpace(value[:i])
	}
	switch value {
	case empty, "":
		if value == empty {
			lines[start] = key + ":\n"
		}
	default:
		return nil, fmt.Errorf("%s: in registry.yaml is written on one line (%s); write it as a block, one entry per line, first, and nothing was changed", key, value)
	}
	// The block ends before the next top level key. The entry goes after
	// its last indented line, so comments that follow the block stay put.
	last := start
	for i := start + 1; i < len(lines); i++ {
		l := lines[i]
		if topKeyRE.MatchString(l) {
			break
		}
		if strings.HasPrefix(l, " ") && strings.TrimSpace(l) != "" {
			last = i
		}
	}
	var b strings.Builder
	for _, l := range lines[:last+1] {
		b.WriteString(l)
	}
	if !strings.HasSuffix(b.String(), "\n") {
		b.WriteString("\n")
	}
	b.WriteString(entry)
	for _, l := range lines[last+1:] {
		b.WriteString(l)
	}
	return []byte(b.String()), nil
}

// setNumberState changes the state of one number in registry.yaml from
// one state to another.
func setNumberState(data []byte, number int, from, to string) ([]byte, error) {
	re := regexp.MustCompile(`(?m)^(\s*-\s+number:\s*` + strconv.Itoa(number) + `\s*\n(?:\s+[a-z_]+:.*\n)*?\s+state:\s*)` + regexp.QuoteMeta(from) + `\b`)
	loc := re.FindSubmatchIndex(data)
	if loc == nil {
		return nil, fmt.Errorf("cannot find the state of number %d in registry.yaml", number)
	}
	return []byte(string(data[:loc[3]]) + to + string(data[loc[1]:])), nil
}

// updateAndValidate runs update and validate, as a person would after a
// change.
func updateAndValidate(root string, out io.Writer) error {
	if _, up, err := Update(root); err != nil {
		return err
	} else if up.Len() > 0 {
		up.Print(out)
		return fmt.Errorf("update found %d problem(s)", up.Len())
	}
	vp, _, err := Validate(root, "", false)
	if err != nil {
		return err
	}
	if vp.Len() > 0 {
		vp.Print(out)
		return fmt.Errorf("validate found %d problem(s)", vp.Len())
	}
	fmt.Fprintf(out, "ran update and validate: the repository is valid\n")
	return nil
}

// nextSteps prints the git steps that follow a change, suggesting a branch
// name when the change was made on main.
func nextSteps(root string, out io.Writer, branch, message string) {
	say := func(format string, args ...any) { fmt.Fprintf(out, format, args...) }
	current, _ := git(root, "branch", "--show-current")
	current = strings.TrimSpace(current)
	changed, _ := git(root, "status", "--porcelain", "--untracked-files=all", "--", "registry.yaml", docsDir)
	var files []string
	for _, line := range strings.Split(strings.TrimRight(changed, "\n"), "\n") {
		if len(line) > 3 {
			files = append(files, strings.TrimSpace(line[3:]))
		}
	}
	sort.Strings(files)
	say("\nNext:\n")
	n := 1
	if current == "" || current == "main" || current == "master" {
		say("  %d. git switch -c %s\n", n, branch)
		n++
	} else {
		branch = current
	}
	if len(files) == 0 {
		files = []string{"registry.yaml"}
	}
	say("  %d. git add %s\n", n, strings.Join(files, " "))
	say("  %d. git commit -m %q\n", n+1, message)
	say("  %d. git push -u %s %s, and open a pull request\n", n+2, remoteName(root), branch)
}
