// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 OSTERA

// Package pub validates and maintains the OSTERA publications repository:
// registry.yaml, the published documents in docs/<shortname>/, and the
// public reviews in docs/reviews/. The pubcheck command (../main.go) is a
// thin command line wrapper around the four functions in this file.
package pub

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SumsFile is the name of the checksum file in every version, corrections,
// and review folder.
const SumsFile = sumsFile

// Validate checks registry.yaml, every metadata.yaml, every review.yaml,
// every published file, every review draft, every checksum, and every
// generated file. When base is not empty, it also checks that nothing
// published or announced in the git commit base was changed, removed, or
// renamed. The error is for problems running git, not for problems in the
// repository, which are returned in Problems.
func Validate(root, base string) (*Problems, error) {
	p := validate(root)
	if base != "" {
		if err := checkHistory(root, base, p); err != nil {
			return p, err
		}
	}
	return p, nil
}

// Generate writes the landing page (index.md) of each document and each
// version, the page of each public review and of each review history,
// docs/catalog.json, and the generated sections of docs/index.md,
// docs/about.md, and docs/reviews/index.md. It writes nothing, and returns
// the problems, if registry.yaml, a metadata.yaml, or a review.yaml is
// wrong.
func Generate(root string) (*Problems, error) {
	return generate(root)
}

// WriteChecksums writes SumsFile in dir for every published artifact or
// review draft in it.
func WriteChecksums(dir string) error {
	return writeSums(dir)
}

// Update brings every SumsFile up to date and then runs Generate. It looks
// at every version folder (docs/<shortname>/v<version>/), corrections
// folder (docs/<shortname>/corrections/), and review folder
// (docs/reviews/<shortname>/v<version>/<draft>/), and rewrites SumsFile
// only where it is missing or does not match the files. It returns the
// folders it wrote, relative to root.
//
// Update never decides whether a change is allowed. If a published file
// was changed by mistake, Update records the new checksum like any other,
// and Validate with a base still reports the changed file, because it
// compares the files themselves with the base.
func Update(root string) ([]string, *Problems, error) {
	p := &Problems{}
	var written []string
	var dirs []string
	for _, pattern := range []string{
		docsDir + "/*/v*",
		docsDir + "/*/corrections",
		reviewsDir + "/*/v*/*",
	} {
		matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(pattern)))
		if err != nil {
			return nil, p, err
		}
		dirs = append(dirs, matches...)
	}
	sort.Strings(dirs)
	for _, dir := range dirs {
		rel := filepath.ToSlash(strings.TrimPrefix(dir, root+string(filepath.Separator)))
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			continue
		}
		if strings.HasPrefix(rel, reviewsDir+"/") != (strings.Count(rel, "/") == 4) {
			continue // docs/reviews/<shortname>/v<version> itself is not a file folder
		}
		names, err := artifactsIn(dir)
		if err != nil {
			return nil, p, err
		}
		if len(names) == 0 {
			continue
		}
		bad := false
		for _, n := range names {
			if !filenameRE.MatchString(n) && !draftRE.MatchString(n) {
				p.Addf(rel+"/"+n, "does not match the filename grammar or the draft naming rule, so no checksums were written for this folder")
				bad = true
			}
		}
		if bad {
			continue
		}
		want, err := sumsContent(dir, names)
		if err != nil {
			return nil, p, err
		}
		if have, err := os.ReadFile(filepath.Join(dir, sumsFile)); err == nil && string(have) == want {
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, sumsFile), []byte(want), 0o644); err != nil {
			return nil, p, err
		}
		written = append(written, rel)
	}
	if p.Len() > 0 {
		return written, p, nil
	}
	gp, err := generate(root)
	return written, gp, err
}

// FindRoot returns dir if it contains registry.yaml. When dir is empty,
// it looks for registry.yaml in the current directory and its parents.
func FindRoot(dir string) (string, error) {
	if dir != "" {
		if _, err := os.Stat(filepath.Join(dir, "registry.yaml")); err != nil {
			return "", fmt.Errorf("%s does not contain registry.yaml", dir)
		}
		return dir, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "registry.yaml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("cannot find registry.yaml in this directory or any parent; use -root")
		}
		dir = parent
	}
}

func readFile(root, rel string) ([]byte, bool) {
	data, err := os.ReadFile(filepath.Join(root, rel))
	return data, err == nil
}
