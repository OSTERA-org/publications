// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 OSTERA

package pub

import (
	"fmt"
	"regexp"
	"strings"
)

// formatMetadata writes a metadata.yaml in the house layout: every field,
// in the order of the schema, with the summary and each citation as folded
// text. The publish command uses it to write a new metadata.yaml, or to
// write one again with a new version added. Every field of Metadata is
// written, so nothing read from an existing file is lost.
func formatMetadata(m *Metadata) string {
	var b strings.Builder
	w := func(format string, args ...any) { fmt.Fprintf(&b, format, args...) }

	w("schema_version: %d\n", m.SchemaVersion)
	w("number: %d\n", m.Number)
	w("shortname: %s\n", m.Shortname)
	w("title: %s\n", yamlScalar(m.Title))
	if m.WorkingGroup != "" {
		w("working_group: %s\n", m.WorkingGroup)
	}
	w("authoritative_language: %s\n", m.AuthoritativeLanguage)
	w("authoritative_format: %s\n", m.AuthoritativeFormat)
	w("keywords:\n")
	for _, k := range m.Keywords {
		w("  - %s\n", yamlScalar(k))
	}
	w("summary: >\n%s", folded(m.Summary, "  "))
	w("canonical_url: %s\n", m.CanonicalURL)
	writeRefs(w, "obsoletes", m.Obsoletes)
	writeRefs(w, "updates", m.Updates)
	if m.Supplements == nil {
		w("supplements: null\n")
	} else {
		w("supplements: %d\n", *m.Supplements)
	}
	w("supplemented_by: %s\n", intList(m.SupplementedBy))
	w("related: %s\n", intList(m.Related))
	if len(m.Implementations) == 0 {
		w("implementations: []\n")
	} else {
		w("implementations:\n")
		for _, im := range m.Implementations {
			w("  - language: %s\n    url: %s\n", yamlScalar(im.Language), im.URL)
		}
	}
	w("versions:\n")
	for _, v := range m.Versions {
		w("  - version: %q\n", v.Version)
		w("    type: %s\n", v.Type)
		w("    status: %s\n", v.Status)
		w("    published: %s\n", v.Published)
		w("    developed_by: %s\n", v.DevelopedBy)
		w("    languages: [%s]\n", strings.Join(v.Languages, ", "))
		w("    formats: [%s]\n", strings.Join(v.Formats, ", "))
		writePeople(w, "authors", v.Authors)
		writePeople(w, "editors", v.Editors)
		if len(v.Incorporates) > 0 {
			w("    incorporates:\n")
			for _, x := range v.Incorporates {
				w("      - %s\n", yamlScalar(x))
			}
		}
		w("    citation: >\n%s", folded(v.Citation, "      "))
		if len(v.Corrections) == 0 {
			w("    corrections: []\n")
		} else {
			w("    corrections:\n")
			for _, c := range v.Corrections {
				w("      - kind: %s\n        sequence: %d\n        date: %s\n", c.Kind, c.Sequence, c.Date)
				w("        languages: [%s]\n        formats: [%s]\n", strings.Join(c.Languages, ", "), strings.Join(c.Formats, ", "))
			}
		}
	}
	return b.String()
}

func writeRefs(w func(string, ...any), name string, refs []ExternalRef) {
	if len(refs) == 0 {
		w("%s: []\n", name)
		return
	}
	w("%s:\n", name)
	for _, r := range refs {
		w("  - id: %s\n    url: %s\n", yamlScalar(r.ID), r.URL)
	}
}

func writePeople(w func(string, ...any), name string, list []Person) {
	if len(list) == 0 {
		return
	}
	w("    %s:\n", name)
	for _, x := range list {
		w("      - name: %s\n        affiliation: %s\n", yamlScalar(x.Name), yamlScalar(x.Affiliation))
	}
}

func intList(list []int) string {
	var s []string
	for _, n := range list {
		s = append(s, fmt.Sprint(n))
	}
	return "[" + strings.Join(s, ", ") + "]"
}

// plainRE matches text that YAML reads back exactly as it is, without
// quotes: it starts with a letter or digit, and holds no character that
// YAML treats specially.
var plainRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 .,()/&+_'-]*$`)

// yamlScalar returns s as plain text when that is safe, and otherwise in
// double quotes. Text that YAML would read as a number, a date, a boolean,
// or null is quoted too.
func yamlScalar(s string) string {
	if plainRE.MatchString(s) && !strings.HasSuffix(s, " ") && !looksTyped(s) {
		return s
	}
	return fmt.Sprintf("%q", s)
}

var typedRE = regexp.MustCompile(`^(?i:[0-9][0-9_.:+-]*|true|false|yes|no|on|off|null|y|n)$`)

func looksTyped(s string) bool { return typedRE.MatchString(s) }

// folded returns text as the lines of a folded block (>), wrapped at 76
// columns and indented, each ending in a newline.
func folded(text, indent string) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return indent + "\n"
	}
	var b strings.Builder
	line := indent
	for _, word := range words {
		if line != indent && len(line)+1+len(word) > 76 {
			b.WriteString(line + "\n")
			line = indent
		}
		if line != indent {
			line += " "
		}
		line += word
	}
	b.WriteString(line + "\n")
	return b.String()
}
