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
	"strings"
)

type svc struct {
	Iface   string
	Methods []method
}

type method struct {
	Name    string
	Params  []param // includes the leading context parameter
	Results []string
	dir     directive
	hasDir  bool
}

type param struct {
	Name string
	Type string
}

type parsed struct {
	Pkg     string
	Svcs    []svc
	Imports []string
	usesTTL bool
}

// parse analyses src for interfaces that carry at least one //cache: directive.
func parse(fset *token.FileSet, src []byte, filename string) (*parsed, error) {
	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	out := &parsed{Pkg: f.Name.Name}
	used := map[string]bool{}

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
			s, err := buildSvc(fset, ts.Name.Name, iface, used, out)
			if err != nil {
				return nil, err
			}
			if s != nil {
				out.Svcs = append(out.Svcs, *s)
			}
		}
	}
	out.Imports = resolveImports(f, used)
	return out, nil
}

// buildSvc returns a svc only if the interface has a caching directive on at
// least one method; otherwise nil (interfaces with no directives are skipped).
func buildSvc(fset *token.FileSet, name string, iface *ast.InterfaceType, used map[string]bool, out *parsed) (*svc, error) {
	s := svc{Iface: name}
	any := false
	for _, field := range iface.Methods.List {
		if len(field.Names) == 0 {
			continue // embedded interface: not cached, pass through transparently
		}
		ft, ok := field.Type.(*ast.FuncType)
		if !ok {
			continue
		}
		m, err := buildMethod(fset, field, ft, used, out)
		if err != nil {
			return nil, err
		}
		if m.hasDir {
			any = true
		}
		s.Methods = append(s.Methods, m)
	}
	if !any {
		return nil, nil
	}
	return &s, nil
}

func buildMethod(fset *token.FileSet, field *ast.Field, ft *ast.FuncType, used map[string]bool, out *parsed) (method, error) {
	m := method{Name: field.Names[0].Name}

	idx := 0
	var paramFields []*ast.Field
	if ft.Params != nil {
		paramFields = ft.Params.List
	}
	for _, f := range paramFields {
		collectPkgs(f.Type, used)
		typ := printNode(fset, f.Type)
		names := f.Names
		if len(names) == 0 {
			m.Params = append(m.Params, param{Name: fmt.Sprintf("a%d", idx), Type: typ})
			idx++
			continue
		}
		for _, n := range names {
			m.Params = append(m.Params, param{Name: n.Name, Type: typ})
			idx++
		}
	}
	if ft.Results != nil {
		for _, r := range ft.Results.List {
			collectPkgs(r.Type, used)
			n := len(r.Names)
			if n == 0 {
				n = 1
			}
			for i := 0; i < n; i++ {
				m.Results = append(m.Results, printNode(fset, r.Type))
			}
		}
	}

	if field.Doc != nil {
		lines := make([]string, len(field.Doc.List))
		for i, c := range field.Doc.List {
			lines[i] = c.Text
		}
		d, ok, err := parseDirective(lines)
		if err != nil {
			return m, fmt.Errorf("%s: %w", m.Name, err)
		}
		m.dir, m.hasDir = d, ok
		if ok && d.ttl != "0" {
			out.usesTTL = true
		}
	}
	return m, nil
}

func collectPkgs(expr ast.Expr, used map[string]bool) {
	ast.Inspect(expr, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok {
				used[id.Name] = true
			}
		}
		return true
	})
}

func resolveImports(f *ast.File, used map[string]bool) []string {
	var out []string
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		name := path[strings.LastIndex(path, "/")+1:]
		if imp.Name != nil {
			name = imp.Name.Name
		}
		if used[name] {
			if imp.Name != nil {
				out = append(out, imp.Name.Name+" "+imp.Path.Value)
			} else {
				out = append(out, imp.Path.Value)
			}
		}
	}
	return out
}

func printNode(fset *token.FileSet, n ast.Node) string {
	var b bytes.Buffer
	_ = printer.Fprint(&b, fset, n)
	return b.String()
}
