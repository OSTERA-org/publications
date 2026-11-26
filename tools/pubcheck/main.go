// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 OSTERA

// Command pubcheck validates and maintains the OSTERA publications
// repository. The work is done by the package in pub/; this file only
// reads the command line.
//
//	pubcheck [-root DIR] update
//	pubcheck [-root DIR] validate [-base REF [-force]]
//	pubcheck [-root DIR] generate
//	pubcheck [-root DIR] checksums DIR...
//	pubcheck [-root DIR] publish [SHORTNAME | URL]
//	pubcheck [-root DIR] add-wg [ID]
//	pubcheck [-root DIR] reserve [SHORTNAME | URL]
//	pubcheck [-root DIR] assign [SHORTNAME | URL]
//
// update deletes the files the operating system leaves in folders (such
// as .DS_Store), writes SHA512SUMS in every version and corrections folder
// where it is missing or out of date, and then does what generate
// does. It is the one command the publishing procedures use before
// validate.
//
// validate checks registry.yaml, every metadata.yaml, every published
// file, every checksum, and every generated file. With -base, it also
// checks that nothing published since REF was changed, removed, or renamed. With
// -force as well, those history problems are shown as warnings instead,
// for a deliberate exception; -force is refused in GitHub Actions.
//
// generate writes the landing page (index.md) of each document and each
// version, docs/catalog.json, and the generated sections of docs/index.md
// and docs/about.md.
//
// checksums writes SHA512SUMS for each named directory: a version, a
// corrections folder. Relative directories are taken
// from the repository root.
//
// publish publishes an approved draft from its Working Group site. It
// takes the short name of the work item, such as jcs, and finds it on
// wg.ostera.org, or the address of its page, such as
// https://wg.ostera.org/jss/work-items/jcs/, or of one of its draft pages.
// Without either, it asks for the short name.
// It checks that the draft is ready, asks for the publication date and the
// branch name, creates the branch, downloads the PDF and checks its
// SHA-512 sum, writes metadata.yaml and registry.yaml, runs update and
// validate, shows a summary, and when that is confirmed, commits only the
// files of this publication. It never pushes.
//
// add-wg adds a Working Group to registry.yaml, suggesting its name from
// wg.ostera.org. reserve and assign add the next free document number for
// a work item, as reserved or assigned. They find the work item on
// wg.ostera.org from its short name, and suggest its type, and today's
// date; when its Working Group is not in registry.yaml, they offer to add
// it too. assign also changes a reserved number to assigned. Each one asks
// before it changes anything, runs update and validate, and prints the git
// steps.
//
// Without -root, pubcheck looks for registry.yaml in the current directory
// and its parents. The usual way to run it is through the Makefile in the
// repository root, for example make validate.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/OSTERA-org/publications/tools/pubcheck/pub"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("pubcheck", flag.ContinueOnError)
	rootFlag := fs.String("root", "", "repository root (default: the nearest directory containing registry.yaml)")
	fs.Usage = usage
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		usage()
		return 2
	}
	root, err := pub.FindRoot(*rootFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pubcheck:", err)
		return 2
	}

	cmd, rest := fs.Arg(0), fs.Args()[1:]
	switch cmd {
	case "validate":
		vf := flag.NewFlagSet("validate", flag.ContinueOnError)
		base := vf.String("base", "", "git ref to compare against for the immutability checks")
		force := vf.Bool("force", false, "show the history problems as warnings instead of errors (not allowed in GitHub Actions)")
		if err := vf.Parse(rest); err != nil {
			return 2
		}
		p, overridden, err := pub.Validate(root, *base, *force)
		if err != nil {
			fmt.Fprintln(os.Stderr, "pubcheck:", err)
			return 2
		}
		if overridden.Len() > 0 {
			overridden.PrintWarnings(os.Stdout)
			fmt.Fprintf(os.Stdout, "\n%d history problem(s) overridden with -force; this change must be merged by someone on the bypass list of the main branch ruleset\n\n", overridden.Len())
		}
		if p.Len() > 0 {
			p.Print(os.Stdout)
			fmt.Fprintf(os.Stdout, "\n%d problem(s) found\n", p.Len())
			return 1
		}
		fmt.Println("repository is valid")
		return 0

	case "update":
		removed, err := pub.RemoveClutter(root)
		for _, f := range removed {
			fmt.Printf("deleted %s\n", f)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "pubcheck:", err)
			return 2
		}
		written, p, err := pub.Update(root)
		for _, dir := range written {
			fmt.Printf("wrote %s\n", filepath.Join(dir, pub.SumsFile))
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "pubcheck:", err)
			return 2
		}
		if p.Len() > 0 {
			p.Print(os.Stdout)
			fmt.Fprintf(os.Stdout, "\nnothing generated: fix the %d problem(s) above first\n", p.Len())
			return 1
		}
		if len(written) == 0 {
			fmt.Println("all checksums were already up to date")
		}
		fmt.Println("generated the landing pages, catalog.json, and the sections of index.md and about.md")
		return 0

	case "generate":
		p, err := pub.Generate(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, "pubcheck:", err)
			return 2
		}
		if p.Len() > 0 {
			p.Print(os.Stdout)
			fmt.Fprintf(os.Stdout, "\nnothing generated: fix the %d problem(s) above first\n", p.Len())
			return 1
		}
		fmt.Println("generated the landing pages, catalog.json, and the sections of index.md and about.md")
		return 0

	case "publish":
		if len(rest) > 1 {
			fmt.Fprintln(os.Stderr, "pubcheck: publish takes at most one argument, the short name of the work item, such as jcs, or its page")
			return 2
		}
		address := ""
		if len(rest) == 1 {
			address = rest[0]
		}
		if err := pub.Publish(root, address, os.Stdin, os.Stdout, time.Now()); err != nil {
			fmt.Fprintln(os.Stderr, "pubcheck:", err)
			return 1
		}
		return 0

	case "add-wg":
		if len(rest) > 1 {
			fmt.Fprintln(os.Stderr, "pubcheck: add-wg takes at most one argument, the Working Group id, such as jss")
			return 2
		}
		id := ""
		if len(rest) == 1 {
			id = rest[0]
		}
		if err := pub.AddWorkingGroup(root, id, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "pubcheck:", err)
			return 1
		}
		return 0

	case "reserve", "assign":
		if len(rest) > 1 {
			fmt.Fprintf(os.Stderr, "pubcheck: %s takes at most one argument, the short name of the work item, such as jcs, or its page\n", cmd)
			return 2
		}
		address := ""
		if len(rest) == 1 {
			address = rest[0]
		}
		state := map[string]string{"reserve": "reserved", "assign": "assigned"}[cmd]
		if err := pub.AllocateNumber(root, state, address, os.Stdin, os.Stdout, time.Now()); err != nil {
			fmt.Fprintln(os.Stderr, "pubcheck:", err)
			return 1
		}
		return 0

	case "checksums":
		if len(rest) == 0 {
			fmt.Fprintln(os.Stderr, "pubcheck: checksums needs at least one directory")
			return 2
		}
		for _, dir := range rest {
			if !filepath.IsAbs(dir) {
				dir = filepath.Join(root, dir)
			}
			if err := pub.WriteChecksums(dir); err != nil {
				fmt.Fprintln(os.Stderr, "pubcheck:", err)
				return 1
			}
			fmt.Printf("wrote %s\n", filepath.Join(dir, pub.SumsFile))
		}
		return 0
	}
	usage()
	return 2
}

func usage() {
	fmt.Fprint(os.Stderr, `usage:
  pubcheck [-root DIR] update
  pubcheck [-root DIR] validate [-base REF [-force]]
  pubcheck [-root DIR] generate
  pubcheck [-root DIR] checksums DIR...
  pubcheck [-root DIR] publish [SHORTNAME | URL]
  pubcheck [-root DIR] add-wg [ID]
  pubcheck [-root DIR] reserve [SHORTNAME | URL]
  pubcheck [-root DIR] assign [SHORTNAME | URL]
`)
}
