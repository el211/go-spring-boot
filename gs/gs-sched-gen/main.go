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

	fs := flag.NewFlagSet("gs-sched-gen", flag.ExitOnError)
	in := fs.String("in", "", "input Go file declaring @schedule-annotated interfaces (required)")
	out := fs.String("out", "", "output file (default: <in> with _sched.go suffix)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: gs-sched-gen -in <file.go> [-out <file>] [-v|-vv]")
		fs.PrintDefaults()
	}
	_ = fs.Parse(rest)

	if verbosity >= 1 {
		infof("argv: %v", os.Args)
	}
	if *in == "" {
		fmt.Fprintln(os.Stderr, "gs-sched-gen: -in is required")
		fs.Usage()
		os.Exit(2)
	}
	outPath := *out
	if outPath == "" {
		outPath = strings.TrimSuffix(*in, ".go") + "_sched.go"
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
		return fmt.Errorf("gs-sched-gen: no interface with //schedule: directives found in %s", inPath)
	}
	for _, s := range p.Svcs {
		infof("scheduling %d job(s) for %s", len(s.Jobs), s.Iface)
		for _, j := range s.Jobs {
			detailf("%s -> %s", j.Method, j.sc.kind)
		}
	}
	code, err := emit(p)
	if err != nil {
		return err
	}
	infof("writing %s", outPath)
	return os.WriteFile(outPath, code, 0o644)
}
