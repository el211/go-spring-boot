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

	fs := flag.NewFlagSet("gs-data-gen", flag.ExitOnError)
	in := fs.String("in", "", "input Go file declaring repository interfaces (required)")
	out := fs.String("out", "", "output file (default: <in> with _gen.go suffix)")
	bImport := fs.String("backend", "go-spring.org/starter-data-gorm", "backend import path")
	bAlias := fs.String("backend-alias", "gormdata", "backend package alias in generated code")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: gs-data-gen -in <file.go> [-out <file>] [-v|-vv]")
		fs.PrintDefaults()
	}
	_ = fs.Parse(rest)

	if verbosity >= 1 {
		infof("argv: %v", os.Args)
	}

	if *in == "" {
		fmt.Fprintln(os.Stderr, "gs-data-gen: -in is required")
		fs.Usage()
		os.Exit(2)
	}
	outPath := *out
	if outPath == "" {
		outPath = strings.TrimSuffix(*in, ".go") + "_gen.go"
	}

	if err := run(*in, outPath, backend{Import: *bImport, Alias: *bAlias}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(inPath, outPath string, b backend) error {
	if cwd, err := os.Getwd(); err == nil {
		detailf("cwd: %s", cwd)
	}
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
	if len(p.Repos) == 0 {
		return fmt.Errorf("gs-data-gen: no repository interface (embedding data.CrudRepository) found in %s", inPath)
	}
	infof("found %d repository interface(s) in package %s", len(p.Repos), p.Pkg)
	for _, r := range p.Repos {
		detailf("%s[%s, %s]: %d derived method(s)", r.Iface, r.Entity, r.ID, len(r.Methods))
		for _, m := range r.Methods {
			detailf("  %s -> %s", m.Name, m.Results)
		}
	}
	code, err := emit(p, b)
	if err != nil {
		return err
	}
	infof("writing %s", outPath)
	return os.WriteFile(outPath, code, 0o644)
}
