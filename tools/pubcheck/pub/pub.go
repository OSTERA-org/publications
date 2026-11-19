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
