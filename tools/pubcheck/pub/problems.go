// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 OSTERA

package pub

import (
	"fmt"
	"io"
	"os"
	"sort"
)

// Problem is one validation failure, tied to a file and optionally a line.
type Problem struct {
	Path    string
	Line    int
	Message string
}

// Problems collects every failure so a single run reports all of them.
type Problems struct {
	list []Problem
}

func (p *Problems) Add(path string, line int, msg string) {
	p.list = append(p.list, Problem{Path: path, Line: line, Message: msg})
}

func (p *Problems) Addf(path string, format string, args ...any) {
	p.Add(path, 0, fmt.Sprintf(format, args...))
}

func (p *Problems) Len() int { return len(p.list) }

// Print writes the problems sorted by file. Inside GitHub Actions it also
// emits annotations, so each problem appears on the pull request.
func (p *Problems) Print(w io.Writer) { p.print(w, "error", "") }

// PrintWarnings writes the problems as warnings, for the history problems
// that validate -force overrides.
func (p *Problems) PrintWarnings(w io.Writer) { p.print(w, "warning", "warning: ") }

func (p *Problems) print(w io.Writer, level, prefix string) {
	sort.SliceStable(p.list, func(i, j int) bool {
		if p.list[i].Path != p.list[j].Path {
			return p.list[i].Path < p.list[j].Path
		}
		return p.list[i].Line < p.list[j].Line
	})
	annotate := os.Getenv("GITHUB_ACTIONS") == "true"
	for _, x := range p.list {
		loc := x.Path
		if x.Line > 0 {
			loc = fmt.Sprintf("%s:%d", x.Path, x.Line)
		}
		fmt.Fprintf(w, "%s%s: %s\n", prefix, loc, x.Message)
		if annotate {
			if x.Line > 0 {
				fmt.Fprintf(w, "::%s file=%s,line=%d::%s\n", level, x.Path, x.Line, x.Message)
			} else {
				fmt.Fprintf(w, "::%s file=%s::%s\n", level, x.Path, x.Message)
			}
		}
	}
}
