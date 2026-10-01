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
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
)

type svc struct {
	Iface string
	Jobs  []job
}

type job struct {
	Method string
	sc     schedule
}

type parsed struct {
	Pkg      string
	Svcs     []svc
	usesTime bool
}

// parse finds interfaces with at least one //schedule: method.
func parse(fset *token.FileSet, src []byte, filename string) (*parsed, error) {
	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	out := &parsed{Pkg: f.Name.Name}
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts := spec.(*ast.TypeSpec)
			iface, ok := ts.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}
			s, err := buildSvc(fset, ts.Name.Name, iface, out)
			if err != nil {
				return nil, err
			}
			if s != nil {
				out.Svcs = append(out.Svcs, *s)
			}
		}
	}
	return out, nil
}

func buildSvc(fset *token.FileSet, name string, iface *ast.InterfaceType, out *parsed) (*svc, error) {
	s := svc{Iface: name}
	for _, field := range iface.Methods.List {
		if len(field.Names) == 0 || field.Doc == nil {
			continue
		}
		ft, ok := field.Type.(*ast.FuncType)
		if !ok {
			continue
		}
		lines := docLines(field.Doc)
		sc, ok, err := parseSchedule(lines)
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %w", name, field.Names[0].Name, err)
		}
		if !ok {
			continue
		}
		if err := checkSignature(fset, ft); err != nil {
			return nil, fmt.Errorf("%s.%s: %w", name, field.Names[0].Name, err)
		}
		s.Jobs = append(s.Jobs, job{Method: field.Names[0].Name, sc: sc})
		if sc.rate != "" || sc.initialDelay != "" || sc.jitter != "" || sc.timeout != "" {
			out.usesTime = true
		}
	}
	if len(s.Jobs) == 0 {
		return nil, nil
	}
	return &s, nil
}

// checkSignature enforces that a scheduled method is func(context.Context) error,
// the exact shape cloud/scheduling.NewJob accepts as its run function.
func checkSignature(fset *token.FileSet, ft *ast.FuncType) error {
	params := fieldTypes(fset, ft.Params)
	results := fieldTypes(fset, ft.Results)
	if len(params) != 1 || params[0] != "context.Context" {
		return fmt.Errorf("scheduled method must take exactly (context.Context), got (%v)", params)
	}
	if len(results) != 1 || results[0] != "error" {
		return fmt.Errorf("scheduled method must return exactly error, got (%v)", results)
	}
	return nil
}

func fieldTypes(fset *token.FileSet, fl *ast.FieldList) []string {
	var out []string
	if fl == nil {
		return out
	}
	for _, f := range fl.List {
		n := len(f.Names)
		if n == 0 {
			n = 1
		}
		for i := 0; i < n; i++ {
			out = append(out, printNode(fset, f.Type))
		}
	}
	return out
}

func docLines(cg *ast.CommentGroup) []string {
	lines := make([]string, len(cg.List))
	for i, c := range cg.List {
		lines[i] = c.Text
	}
	return lines
}

func printNode(fset *token.FileSet, n ast.Node) string {
	var b bytes.Buffer
	_ = printer.Fprint(&b, fset, n)
	return b.String()
}
