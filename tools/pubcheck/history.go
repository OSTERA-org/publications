// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 OSTERA

package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"reflect"
	"strings"
)

// checkHistory enforces the rules that only make sense against an earlier
// state of the repository: published version files never change, numbers
// are never removed or renamed, and published version entries are only
// changed in the ways the process allows. It compares the merge base of
// base and HEAD with the working tree, so it works both in CI and locally
// before anything is committed.
func checkHistory(root, base string, p *Problems) error {
	mb, err := git(root, "merge-base", base, "HEAD")
	if err != nil {
		return fmt.Errorf("cannot find the merge base of %s and HEAD: %w", base, err)
	}
	mb = strings.TrimSpace(mb)

	changes, err := changedPaths(root, mb)
	if err != nil {
		return err
	}
	oldDocs, curDocs := map[string]*Metadata{}, map[string]*Metadata{}
	for _, c := range changes {
		short := strings.SplitN(strings.TrimPrefix(c.path, docsDir+"/"), "/", 2)[0]
		if _, seen := oldDocs[short]; seen {
			continue
		}
		path := docsDir + "/" + short + "/metadata.yaml"
		if data, ok := gitShow(root, mb, path); ok {
			oldDocs[short] = parseMetadata(path, data, &Problems{})
		} else {
			oldDocs[short] = nil
		}
		if data, ok := readFile(root, path); ok {
			curDocs[short] = parseMetadata(path, data, &Problems{})
		}
	}
	for _, c := range changes {
		checkChangedPath(root, mb, c, oldDocs, curDocs, p)
	}

	if data, ok := gitShow(root, mb, "registry.yaml"); ok {
		scratch := &Problems{}
		if old := parseRegistry("registry.yaml", data, scratch); old != nil {
			cur := loadRegistry(root, &Problems{})
			if cur != nil {
				compareRegistry(old, cur, p)
			}
		}
	}

	out, err := git(root, "ls-tree", "-r", "--name-only", mb, "--", docsDir)
	if err != nil {
		return err
	}
	for _, path := range strings.Split(strings.TrimSpace(out), "\n") {
		if !strings.HasSuffix(path, "/metadata.yaml") || strings.Count(path, "/") != 2 {
			continue
		}
		data, _ := gitShow(root, mb, path)
		old := parseMetadata(path, data, &Problems{})
		if old == nil {
			continue
		}
		cur, ok := readFile(root, path)
		if !ok {
			p.Addf(path, "was removed, but a published document can never be removed")
			continue
		}
		if m := parseMetadata(path, cur, &Problems{}); m != nil {
			compareVersions(path, old, m, p)
		}
	}
	return nil
}

type change struct{ status, path string }

// changedPaths lists every path that differs between mb and the working
// tree, including untracked files, which count as added.
func changedPaths(root, mb string) ([]change, error) {
	out, err := git(root, "diff", "--name-status", "--no-renames", mb, "--", docsDir)
	if err != nil {
		return nil, err
	}
	var list []change
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if f := strings.SplitN(line, "\t", 2); len(f) == 2 {
			list = append(list, change{status: f[0][:1], path: f[1]})
		}
	}
	out, err = git(root, "ls-files", "--others", "--exclude-standard", "--", docsDir)
	if err != nil {
		return nil, err
	}
	for _, path := range strings.Split(strings.TrimSpace(out), "\n") {
		if path != "" {
			list = append(list, change{status: "A", path: path})
		}
	}
	return list, nil
}

// checkChangedPath enforces immutability for one changed file. In a
// published version directory, the only changes allowed are adding the
// files for a format newly appended to that version's formats list, and
// adding their lines to SHA512SUMS. Existing files and existing checksum
// lines never change. The generated landing page (index.md) is not a
// published file; it is regenerated, and validate checks that it is
// current.
func checkChangedPath(root, mb string, c change, oldDocs, curDocs map[string]*Metadata, p *Problems) {
	parts := strings.Split(c.path, "/")
	if len(parts) != 4 || parts[0] != docsDir {
		return
	}
	short, dir, name := parts[1], parts[2], parts[3]
	verDir := strings.Join(parts[:3], "/")
	switch {
	case strings.HasPrefix(dir, "v"):
		if !gitExists(root, mb, verDir) {
			return // a new version directory; nothing in it is published yet
		}
		switch {
		case name == pageFile:
			// The generated landing page; validate checks that it is current.
		case name == sumsFile && c.status == "M":
			checkSumsAppendOnly(root, mb, c.path, p)
		case name != sumsFile && c.status == "A":
			if !isNewFormatFile(oldDocs[short], curDocs[short], dir, name) {
				p.Addf(c.path, "cannot be added to %s, which is already published; only the files for a format newly appended to that version's formats list may be added", verDir)
			}
		default:
			p.Addf(c.path, "is in a published version directory and cannot be changed or deleted; publish a correction or a new version instead")
		}
	case dir == "corrections":
		if name != sumsFile && c.status != "A" {
			p.Addf(c.path, "is a published correction and cannot be changed or deleted")
		}
	}
}

// isNewFormatFile reports whether name is an artifact for a format that is
// in the current formats list of the version in dir but was not in it at
// the base commit.
func isNewFormatFile(old, cur *Metadata, dir, name string) bool {
	if old == nil || cur == nil {
		return false
	}
	var ov, cv *Version
	for i := range old.Versions {
		if versionDir(old.Versions[i].Version) == dir {
			ov = &old.Versions[i]
		}
	}
	for i := range cur.Versions {
		if versionDir(cur.Versions[i].Version) == dir {
			cv = &cur.Versions[i]
		}
	}
	if ov == nil || cv == nil {
		return false
	}
	for _, f := range cv.Formats {
		if contains(ov.Formats, f) {
			continue
		}
		for _, lang := range cv.Languages {
			if artifactName(cur, cv, "", 0, lang, f) == name {
				return true
			}
		}
	}
	return false
}

// checkSumsAppendOnly fails if any line of the published SHA512SUMS was
// changed or removed. New lines may be added.
func checkSumsAppendOnly(root, mb, path string, p *Problems) {
	old, _ := gitShow(root, mb, path)
	cur, ok := readFile(root, path)
	if !ok {
		p.Addf(path, "is in a published version directory and cannot be deleted")
		return
	}
	now := map[string]bool{}
	for _, line := range strings.Split(string(cur), "\n") {
		now[line] = true
	}
	for _, line := range strings.Split(strings.TrimRight(string(old), "\n"), "\n") {
		if !now[line] {
			p.Addf(path, "an existing checksum line was changed or removed: %s", line)
		}
	}
}

// compareRegistry checks that no number was removed or renamed, and that
// states only move forward: assigned to active, or assigned to retired.
func compareRegistry(old, cur *Registry, p *Problems) {
	const path = "registry.yaml"
	now := map[int]NumberEntry{}
	for _, n := range cur.Numbers {
		now[n.Number] = n
	}
	for _, o := range old.Numbers {
		n, ok := now[o.Number]
		if !ok {
			p.Addf(path, "number %d was removed; numbers are never removed (mark it retired instead)", o.Number)
			continue
		}
		if n.Shortname != o.Shortname {
			p.Addf(path, "number %d changed its short name from %s to %s; short names never change", o.Number, o.Shortname, n.Shortname)
		}
		if n.Assigned != o.Assigned {
			p.Addf(path, "number %d changed its assigned date", o.Number)
		}
		if n.State != o.State && !(o.State == "assigned" && (n.State == "active" || n.State == "retired")) {
			p.Addf(path, "number %d cannot change state from %s to %s", o.Number, o.State, n.State)
		}
	}
	for id := range old.WorkingGroups {
		if _, ok := cur.WorkingGroups[id]; !ok {
			p.Addf(path, "Working Group %s was removed; close it instead", id)
		}
	}
}

// compareVersions checks that every previously published version entry is
// still present, in place, and changed only in its status (forward), by
// appending formats, or by appending corrections.
func compareVersions(path string, old, cur *Metadata, p *Problems) {
	for i, o := range old.Versions {
		if i >= len(cur.Versions) || cur.Versions[i].Version != o.Version {
			p.Addf(path, "version %s was removed or moved; published versions stay in place, oldest first", o.Version)
			continue
		}
		n := cur.Versions[i]
		if n.Status != o.Status && !statusForward(o.Status, n.Status) {
			p.Addf(path, "version %s cannot change status from %s to %s", o.Version, o.Status, n.Status)
		}
		if len(n.Corrections) < len(o.Corrections) || !reflect.DeepEqual(n.Corrections[:len(o.Corrections)], o.Corrections) {
			p.Addf(path, "version %s: existing corrections were changed or removed; corrections may only be appended", o.Version)
		}
		if len(n.Formats) < len(o.Formats) || !reflect.DeepEqual(n.Formats[:len(o.Formats)], o.Formats) {
			p.Addf(path, "version %s: existing formats were changed, removed, or reordered; formats may only be appended", o.Version)
		}
		a, b := o, n
		a.Status, b.Status = "", ""
		a.Corrections, b.Corrections = nil, nil
		a.Formats, b.Formats = nil, nil
		if !reflect.DeepEqual(a, b) {
			p.Addf(path, "version %s was edited; a published entry may only change its status, append formats, or append corrections", o.Version)
		}
	}
}

func statusForward(from, to string) bool {
	return (from == "published" && (to == "superseded" || to == "withdrawn")) ||
		(from == "superseded" && to == "withdrawn")
}

func git(root string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}

func gitShow(root, rev, path string) ([]byte, bool) {
	out, err := git(root, "show", rev+":"+path)
	if err != nil {
		return nil, false
	}
	return []byte(out), true
}

func gitExists(root, rev, path string) bool {
	_, err := git(root, "cat-file", "-e", rev+":"+path)
	return err == nil
}
