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

// Command gs-modulith is the GoSpring Modulith verifier: the build-time
// counterpart to Spring Modulith's ApplicationModules.verify(). It loads a
// declared module model (modulith.json) and the application's real import graph
// (via `go list`), then fails if any package crosses a module boundary it may
// not — a disallowed dependency, or reaching into another module's internals.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	level, rest := splitVerbosity(os.Args[1:])
	verbosity = level

	fs := flag.NewFlagSet("gs-modulith", flag.ExitOnError)
	cfg := fs.String("config", "modulith.json", "module declaration file")
	dir := fs.String("dir", ".", "module root to analyse (go list ./... runs here)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: gs-modulith [-config modulith.json] [-dir .] [-v|-vv]")
		fs.PrintDefaults()
	}
	_ = fs.Parse(rest)

	if verbosity >= 1 {
		infof("argv: %v", os.Args)
	}
	if err := run(*cfg, *dir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cfgPath, dir string) error {
	detailf("dir: %s", dir)
	infof("loading module model from %s", cfgPath)
	mods, err := loadConfig(cfgPath)
	if err != nil {
		return err
	}
	for _, m := range mods.All() {
		detailf("module %q owns %s", m.Name, m.BasePackage)
	}

	infof("loading import graph via go list")
	graph, err := loadGraph(dir)
	if err != nil {
		return err
	}
	infof("checking %d packages against %d modules", len(graph), len(mods.All()))

	violations := mods.Check(graph)
	if len(violations) == 0 {
		infof("OK: no module boundary violations")
		return nil
	}
	for _, v := range violations {
		fmt.Fprintf(os.Stderr, "VIOLATION %s\n", v.Error())
	}
	return fmt.Errorf("gs-modulith: %d boundary violation(s)", len(violations))
}
