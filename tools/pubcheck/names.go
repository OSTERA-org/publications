// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 OSTERA

package main

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// filenameRE is the normative filename grammar from the publishing process.
var filenameRE = regexp.MustCompile(`^OSTERA-([A-Z]{2,3})-([0-9]{4,})-([0-9]{4})-([A-Z][A-Z0-9]*)-v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(\.(0|[1-9][0-9]*))?(-(ERRATUM|CORRIGENDUM)-([1-9][0-9]*))?-([a-z]{2})\.([a-z0-9]+)$`)

// draftRE is the naming rule for drafts in the Working Group
// repositories, which a public review keeps: draft-<type code, lowercase>-
// <short name>-<draft number, at least two digits>.<format>.
var draftRE = regexp.MustCompile(`^draft-([a-z]{2,3})-([a-z][a-z0-9]*)-([0-9]{2,})\.([a-z0-9]+)$`)

var (
	idRE      = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
	typeRE    = regexp.MustCompile(`^[A-Z]{2,3}$`)
	versionRE = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(\.(0|[1-9][0-9]*))?$`)
	langRE    = regexp.MustCompile(`^[a-z]{2}$`)
	formatRE  = regexp.MustCompile(`^[a-z0-9]+$`)
	dateRE    = regexp.MustCompile(`^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])$`)
)

const sumsFile = "SHA512SUMS"

// versionDir is the directory name for a version, for example "v1.0".
func versionDir(version string) string { return "v" + version }

// artifactName builds the filename for one published artifact. kind is
// empty for a version artifact, or "erratum" or "corrigendum" with seq.
func artifactName(m *Metadata, v *Version, kind string, seq int, lang, format string) string {
	year := v.Published
	if len(year) >= 4 {
		year = year[:4]
	}
	parts := []string{
		"OSTERA",
		v.Type,
		fmt.Sprintf("%04d", m.Number),
		year,
		strings.ToUpper(m.Shortname),
		"v" + v.Version,
	}
	if kind != "" {
		parts = append(parts, strings.ToUpper(kind), fmt.Sprint(seq))
	}
	parts = append(parts, lang)
	return strings.Join(parts, "-") + "." + format
}

// expectedVersionFiles lists the artifacts a version directory must hold.
func expectedVersionFiles(m *Metadata, v *Version) []string {
	var names []string
	for _, lang := range v.Languages {
		for _, f := range v.Formats {
			names = append(names, artifactName(m, v, "", 0, lang, f))
		}
	}
	sort.Strings(names)
	return names
}

// expectedCorrectionFiles lists every artifact the corrections directory
// must hold, across all versions.
func expectedCorrectionFiles(m *Metadata) []string {
	var names []string
	for i := range m.Versions {
		v := &m.Versions[i]
		for _, c := range v.Corrections {
			for _, lang := range c.Languages {
				for _, f := range c.Formats {
					names = append(names, artifactName(m, v, c.Kind, c.Sequence, lang, f))
				}
			}
		}
	}
	sort.Strings(names)
	return names
}

// sumsContent returns the SHA512SUMS text for the named files in dir:
// one line per file, digest, two spaces, bare filename, sorted by name in
// byte order. This is the format sha512sum writes and sha512sum -c reads.
func sumsContent(dir string, names []string) (string, error) {
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)
	var b strings.Builder
	for _, name := range sorted {
		f, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			return "", err
		}
		h := sha512.New()
		_, err = io.Copy(h, f)
		f.Close()
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "%s  %s\n", hex.EncodeToString(h.Sum(nil)), name)
	}
	return b.String(), nil
}

// artifactsIn lists the files in dir that look like published artifacts
// or review drafts.
func artifactsIn(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && (strings.HasPrefix(e.Name(), "OSTERA-") || strings.HasPrefix(e.Name(), "draft-")) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// writeSums writes SHA512SUMS for every artifact or review draft in dir.
func writeSums(dir string) error {
	names, err := artifactsIn(dir)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return fmt.Errorf("%s: no OSTERA- or draft- files found", dir)
	}
	for _, n := range names {
		if !filenameRE.MatchString(n) && !draftRE.MatchString(n) {
			return fmt.Errorf("%s: %s does not match the filename grammar or the draft naming rule", dir, n)
		}
	}
	content, err := sumsContent(dir, names)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, sumsFile), []byte(content), 0o644)
}
