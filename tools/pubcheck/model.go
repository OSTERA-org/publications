// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 OSTERA

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
)

// Allowed values. These must match the enums in schemas/*.schema.json;
// schema_test.go checks that they do.
var (
	tracks          = []string{"standards", "informative", "parent", "organizational"}
	wgStates        = []string{"active", "closed"}
	numberStates    = []string{"assigned", "active", "retired"}
	versionStatuses = []string{"published", "superseded", "withdrawn"}
	developedBys    = []string{"working_group", "board", "secretariat"}
	correctionKinds = []string{"erratum", "corrigendum"}
)

// Registry is registry.yaml.
type Registry struct {
	SchemaVersion int                     `yaml:"schema_version"`
	Ranges        map[string]Range        `yaml:"ranges"`
	Types         map[string]TypeDef      `yaml:"types"`
	WorkingGroups map[string]WorkingGroup `yaml:"working_groups"`
	Numbers       []NumberEntry           `yaml:"numbers"`
}

type Range struct {
	First int      `yaml:"first"`
	Last  *int     `yaml:"last"`
	Types []string `yaml:"types"`
}

type TypeDef struct {
	Name  string `yaml:"name"`
	Track string `yaml:"track"`
}

type WorkingGroup struct {
	Name  string `yaml:"name"`
	State string `yaml:"state"`
}

type NumberEntry struct {
	Number    int    `yaml:"number"`
	Shortname string `yaml:"shortname"`
	State     string `yaml:"state"`
	Assigned  string `yaml:"assigned"`
}

// Metadata is docs/<shortname>/metadata.yaml.
type Metadata struct {
	SchemaVersion         int              `yaml:"schema_version"`
	Number                int              `yaml:"number"`
	Shortname             string           `yaml:"shortname"`
	Title                 string           `yaml:"title"`
	WorkingGroup          string           `yaml:"working_group"`
	AuthoritativeLanguage string           `yaml:"authoritative_language"`
	AuthoritativeFormat   string           `yaml:"authoritative_format"`
	Keywords              []string         `yaml:"keywords"`
	Summary               string           `yaml:"summary"`
	CanonicalURL          string           `yaml:"canonical_url"`
	Obsoletes             []ExternalRef    `yaml:"obsoletes"`
	Supplements           *int             `yaml:"supplements"`
	SupplementedBy        []int            `yaml:"supplemented_by"`
	Related               []int            `yaml:"related"`
	Implementations       []Implementation `yaml:"implementations"`
	Versions              []Version        `yaml:"versions"`
}

type ExternalRef struct {
	ID  string `yaml:"id"`
	URL string `yaml:"url"`
}

type Implementation struct {
	Language string `yaml:"language"`
	URL      string `yaml:"url"`
}

type Version struct {
	Version      string       `yaml:"version"`
	Type         string       `yaml:"type"`
	Status       string       `yaml:"status"`
	Published    string       `yaml:"published"`
	DevelopedBy  string       `yaml:"developed_by"`
	Languages    []string     `yaml:"languages"`
	Formats      []string     `yaml:"formats"`
	Authors      []Person     `yaml:"authors"`
	Editors      []Person     `yaml:"editors"`
	Incorporates []string     `yaml:"incorporates"`
	Corrections  []Correction `yaml:"corrections"`
}

type Person struct {
	Name        string `yaml:"name"`
	Affiliation string `yaml:"affiliation"`
}

type Correction struct {
	Kind      string   `yaml:"kind"`
	Sequence  int      `yaml:"sequence"`
	Date      string   `yaml:"date"`
	Languages []string `yaml:"languages"`
	Formats   []string `yaml:"formats"`
}

// Document is one document directory, loaded.
type Document struct {
	Dir  string // docs/<shortname>, relative to the repository root
	Meta *Metadata
}

// Raw text checks catch YAML values that decode silently into the wrong
// thing: an unquoted version (1.10 becomes 1.1) and a number with a
// leading zero (0010 becomes 8, read as octal).
var (
	unquotedVersionRE = regexp.MustCompile(`(?m)^\s*(?:-\s+)?version:\s*[0-9]`)
	leadingZeroRE     = regexp.MustCompile(`(?m)^\s*(?:-\s+)?(?:number|sequence|supplements):\s*0[0-9]`)
	leadingZeroListRE = regexp.MustCompile(`(?m)^\s*(?:supplemented_by|related):\s*\[[^\]]*\b0[0-9]`)
)

func lintRaw(path string, data []byte, p *Problems) {
	for _, loc := range unquotedVersionRE.FindAllIndex(data, -1) {
		p.Add(path, lineOf(data, loc[0]), "version must be quoted (for example version: \"1.0\"), because YAML reads an unquoted 1.10 as 1.1")
	}
	for _, re := range []*regexp.Regexp{leadingZeroRE, leadingZeroListRE} {
		for _, loc := range re.FindAllIndex(data, -1) {
			p.Add(path, lineOf(data, loc[0]), "numbers must not have leading zeros, because YAML may read them as octal")
		}
	}
}

func lineOf(data []byte, offset int) int {
	line := 1
	for _, b := range data[:offset] {
		if b == '\n' {
			line++
		}
	}
	return line
}

func decodeStrict(path string, data []byte, v any, p *Problems) bool {
	if err := yaml.UnmarshalWithOptions(data, v, yaml.DisallowUnknownField()); err != nil {
		p.Add(path, 0, "cannot parse: "+yaml.FormatError(err, false, true))
		return false
	}
	return true
}

func loadRegistry(root string, p *Problems) *Registry {
	const path = "registry.yaml"
	data, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		p.Add(path, 0, "cannot read: "+err.Error())
		return nil
	}
	return parseRegistry(path, data, p)
}

func parseRegistry(path string, data []byte, p *Problems) *Registry {
	lintRaw(path, data, p)
	var r Registry
	if !decodeStrict(path, data, &r, p) {
		return nil
	}
	return &r
}

func parseMetadata(path string, data []byte, p *Problems) *Metadata {
	lintRaw(path, data, p)
	var m Metadata
	if !decodeStrict(path, data, &m, p) {
		return nil
	}
	return &m
}

// docsDir is the folder GitHub Pages publishes as https://pub.ostera.org/.
// It holds the site files and one folder per published document.
const docsDir = "docs"

// siteURL is the address of the published site, and repoURL is the
// repository it is built from.
const (
	siteURL = "https://pub.ostera.org"
	repoURL = "https://github.com/OSTERA-org/Publications"
)

// reservedShortnames cannot be used as short names, because each short
// name is also a top level path on the site and these are taken by site
// pages or are commonly used for site files.
var reservedShortnames = []string{"about", "assets", "catalog", "css", "fonts", "images", "index", "js"}

// loadDocuments reads every docs/<dir>/metadata.yaml. Files directly in
// docs/ are site files (such as index.md, about.md, 404.md, CNAME, and
// _config.yml), and folders starting with _ or . belong to Jekyll, so
// neither is a document. Any other folder is a document, and one with
// no metadata.yaml is reported.
func loadDocuments(root string, p *Problems) []Document {
	base := filepath.Join(root, docsDir)
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		p.Add(docsDir, 0, "cannot read: "+err.Error())
		return nil
	}
	var docs []Document
	for _, e := range entries {
		rel := docsDir + "/" + e.Name()
		if !e.IsDir() || strings.HasPrefix(e.Name(), "_") || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		mpath := rel + "/metadata.yaml"
		data, err := os.ReadFile(filepath.Join(root, mpath))
		if err != nil {
			p.Add(rel, 0, "document directory has no metadata.yaml")
			continue
		}
		m := parseMetadata(mpath, data, p)
		if m == nil {
			continue
		}
		docs = append(docs, Document{Dir: rel, Meta: m})
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].Meta.Number < docs[j].Meta.Number })
	return docs
}

// identifier returns the display identifier, for example "OSTERA TS 1000".
func identifier(typ string, number int) string {
	return fmt.Sprintf("OSTERA %s %04d", typ, number)
}

// docURL is the canonical address of a document's landing page.
func docURL(shortname string) string {
	return siteURL + "/" + shortname + "/"
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
