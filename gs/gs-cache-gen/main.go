/*
 * Copyright 2025 The Go-Spring Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"flag"
	"fmt"
	"go/token"
	"os"
	"strings"
)

func main() {
	level, rest := splitVerbosity(os.Args[1:])
	verbosity = level

	fs := flag.NewFlagSet("gs-cache-gen", flag.ExitOnError)
	in := fs.String("in", "", "input Go file declaring cache-annotated interfaces (required)")
	out := fs.String("out", "", "output file (default: <in> with _cache.go suffix)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: gs-cache-gen -in <file.go> [-out <file>] [-v|-vv]")
		fs.PrintDefaults()
	}
	_ = fs.Parse(rest)

	if verbosity >= 1 {
		infof("argv: %v", os.Args)
	}
	if *in == "" {
		fmt.Fprintln(os.Stderr, "gs-cache-gen: -in is required")
		fs.Usage()
		os.Exit(2)
	}
	outPath := *out
	if outPath == "" {
		outPath = strings.TrimSuffix(*in, ".go") + "_cache.go"
	}
	if err := run(*in, outPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(inPath, outPath string) error {
	infof("parsing %s", inPath)
	src, err := os.ReadFile(inPath)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	p, err := parse(fset, src, inPath)
	if err != nil {
		return err
	}
	if len(p.Svcs) == 0 {
		return fmt.Errorf("gs-cache-gen: no interface with //cache: directives found in %s", inPath)
	}
	infof("found %d cached interface(s) in package %s", len(p.Svcs), p.Pkg)
	for _, s := range p.Svcs {
		detailf("%s: %d method(s)", s.Iface, len(s.Methods))
	}
	code, err := emit(p)
	if err != nil {
		return err
	}
	infof("writing %s", outPath)
	return os.WriteFile(outPath, code, 0o644)
}
