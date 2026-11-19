// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 OSTERA

// Command pubcheck validates and maintains the OSTERA publications
// repository. The work is done by the package in pub/; this file only
// reads the command line.
//
//	pubcheck [-root DIR] validate [-base REF]
//	pubcheck [-root DIR] generate
//	pubcheck [-root DIR] checksums DIR...
//
// validate checks registry.yaml, every metadata.yaml, every review.yaml,
// every published file, every review draft, every checksum, and every
// generated file. With -base, it also checks that nothing published or
// announced since REF was changed, removed, or renamed.
//
// generate writes the landing page (index.md) of each document and each
// version, the page of each public review and of each review history,
// docs/catalog.json, and the generated sections of docs/index.md,
// docs/about.md, and docs/reviews/index.md.
//
// checksums writes SHA512SUMS for each named directory: a version, a
// corrections folder, or a public review. Relative directories are taken
// from the repository root.
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
		if err := vf.Parse(rest); err != nil {
			return 2
		}
		p, err := pub.Validate(root, *base)
		if err != nil {
			fmt.Fprintln(os.Stderr, "pubcheck:", err)
			return 2
		}
		if p.Len() > 0 {
			p.Print(os.Stdout)
			fmt.Fprintf(os.Stdout, "\n%d problem(s) found\n", p.Len())
			return 1
		}
		fmt.Println("repository is valid")
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
		fmt.Println("generated the landing pages, the public review pages, catalog.json, and the sections of index.md, about.md, and reviews/index.md")
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
  pubcheck [-root DIR] validate [-base REF]
  pubcheck [-root DIR] generate
  pubcheck [-root DIR] checksums DIR...
`)
}
