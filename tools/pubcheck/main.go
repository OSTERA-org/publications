// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 OSTERA

// Command pubcheck validates and maintains the OSTERA Publications
// repository.
//
//	pubcheck [-root DIR] validate [-base REF]
//	pubcheck [-root DIR] generate
//	pubcheck [-root DIR] checksums DIR...
//
// validate checks registry.yaml, every metadata.yaml, every published file,
// every checksum, and every generated file. With -base, it also checks that
// nothing published since REF was changed, removed, or renamed.
//
// generate writes the landing page (index.md) of each document and each
// version, docs/catalog.json, and the generated sections of docs/index.md
// and docs/about.md.
//
// checksums writes SHA512SUMS for each named directory.
//
// Without -root, pubcheck looks for registry.yaml in the current directory
// and its parents.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
	root, err := findRoot(*rootFlag)
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
		p := validate(root)
		if *base != "" {
			if err := checkHistory(root, *base, p); err != nil {
				fmt.Fprintln(os.Stderr, "pubcheck:", err)
				return 2
			}
		}
		return report(p, "repository is valid")

	case "generate":
		p, err := generate(root)
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

	case "checksums":
		if len(rest) == 0 {
			fmt.Fprintln(os.Stderr, "pubcheck: checksums needs at least one directory")
			return 2
		}
		for _, dir := range rest {
			// Relative directories are relative to the repository root, so
			// the command works the same from any working directory.
			if !filepath.IsAbs(dir) {
				dir = filepath.Join(root, dir)
			}
			if err := writeSums(dir); err != nil {
				fmt.Fprintln(os.Stderr, "pubcheck:", err)
				return 1
			}
			fmt.Printf("wrote %s\n", filepath.Join(dir, sumsFile))
		}
		return 0
	}
	usage()
	return 2
}

func report(p *Problems, ok string) int {
	if p.Len() == 0 {
		fmt.Println(ok)
		return 0
	}
	p.Print(os.Stdout)
	fmt.Fprintf(os.Stdout, "\n%d problem(s) found\n", p.Len())
	return 1
}

func findRoot(flagValue string) (string, error) {
	if flagValue != "" {
		if _, err := os.Stat(filepath.Join(flagValue, "registry.yaml")); err != nil {
			return "", fmt.Errorf("%s does not contain registry.yaml", flagValue)
		}
		return flagValue, nil
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

func usage() {
	fmt.Fprint(os.Stderr, `usage:
  pubcheck [-root DIR] validate [-base REF]
  pubcheck [-root DIR] generate
  pubcheck [-root DIR] checksums DIR...
`)
}
