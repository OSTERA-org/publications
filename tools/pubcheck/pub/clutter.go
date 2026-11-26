// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 OSTERA

package pub

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Operating systems leave files of their own in folders: macOS writes
// .DS_Store (folder view settings) and ._<name> (extra file data on drives
// that cannot hold it), and Windows writes Thumbs.db (thumbnails) and
// desktop.ini (folder settings). .gitignore keeps them out of git, but
// pubcheck reads the folders on disk, so it ignores them wherever it reads
// a folder, and update deletes them.

// isClutter reports whether name is a file the operating system made.
func isClutter(name string) bool {
	switch {
	case name == ".DS_Store", strings.HasPrefix(name, "._"):
		return true
	case strings.EqualFold(name, "Thumbs.db"), strings.EqualFold(name, "desktop.ini"):
		return true
	}
	return false
}

// readDir is os.ReadDir without the files the operating system made.
// Every check that reads a folder uses it.
func readDir(dir string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dir)
	out := entries[:0]
	for _, e := range entries {
		if !(e.Type().IsRegular() && isClutter(e.Name())) {
			out = append(out, e)
		}
	}
	return out, err
}

// RemoveClutter deletes every file the operating system made anywhere in
// the repository, except inside .git, and returns the paths it deleted,
// relative to root. update runs it first.
func RemoveClutter(root string) ([]string, error) {
	var removed []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if !d.Type().IsRegular() || !isClutter(d.Name()) {
			return nil
		}
		if err := os.Remove(path); err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		removed = append(removed, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(removed)
	return removed, err
}
