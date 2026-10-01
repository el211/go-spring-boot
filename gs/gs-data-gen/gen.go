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

// Command gs-data-gen is the GoSpring Data generator: the build-time
// counterpart to Spring Data's runtime repository proxy. It reads a Go file
// declaring repository interfaces that embed data.CrudRepository[T, ID] (or
// PagingAndSortingRepository) and emits, for each, a backend-bound struct whose
// derived-query methods delegate to the backend's Query* executors — so a
// method named FindByEmailAndStatus turns into real code, no reflection.
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"

	"go-spring.org/cloud/data"
)

// repoSpec is one repository interface slated for generation.
type repoSpec struct {
	Iface   string // interface name, e.g. "UserRepository"
	Entity  string // entity type expression, e.g. "User"
	ID      string // id type expression, e.g. "int64"
	Methods []methodSpec
}

// methodSpec is one derived-query method declared on the interface.
type methodSpec struct {
	Name    string
	Params  []param // every parameter after the leading context
	Results string  // printed result list, e.g. "(User, bool, error)"
	subject data.Subject
	slice   bool // first result (excluding error) is a slice type
	nResult int  // total number of results, including the trailing error
}

type param struct {
	Name string
	Type string
}

// parsed is the whole-file result: the package name, the repository specs and
// the set of source imports the generated signatures reference.
type parsed struct {
	Pkg     string
	Repos   []repoSpec
	Imports []string // import spec lines, e.g. `"time"` or `uuid "github.com/..."`
}

// baseInterfaces are the embedded abstractions that mark an interface for
// generation; both expose the same Query* executors through the backend.
var baseInterfaces = map[string]bool{
	"CrudRepository":             true,
	"PagingAndSortingRepository": true,
}

// parse analyses src and returns every repository interface found.
func parse(fset *token.FileSet, src []byte, filename string) (*parsed, error) {
	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	out := &parsed{Pkg: f.Name.Name}
	usedPkgs := map[string]bool{}

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
			rs, ok := buildRepo(fset, ts.Name.Name, iface, usedPkgs)
			if ok {
				out.Repos = append(out.Repos, rs)
			}
		}
	}
	out.Imports = resolveImports(f, usedPkgs)
	return out, nil
}

// buildRepo extracts an entity/id pair and derived methods from one interface,
// returning ok=false when it embeds no GoSpring Data base interface.
func buildRepo(fset *token.FileSet, name string, iface *ast.InterfaceType, used map[string]bool) (repoSpec, bool) {
	rs := repoSpec{Iface: name}
	found := false
	for _, field := range iface.Methods.List {
		if len(field.Names) == 0 { // embedded interface
			if e, id, ok := baseTypeArgs(fset, field.Type, used); ok {
				rs.Entity, rs.ID, found = e, id, true
			}
			continue
		}
		ft := field.Type.(*ast.FuncType)
		m, ok := buildMethod(fset, field.Names[0].Name, ft, used)
		if ok {
			rs.Methods = append(rs.Methods, m)
		}
	}
	return rs, found
}

// baseTypeArgs recognises data.CrudRepository[T, ID] (and the paging variant)
// and returns the printed T and ID type expressions.
func baseTypeArgs(fset *token.FileSet, expr ast.Expr, used map[string]bool) (entity, id string, ok bool) {
	il, isList := expr.(*ast.IndexListExpr)
	if !isList {
		return "", "", false
	}
	sel, isSel := il.X.(*ast.SelectorExpr)
	if !isSel || !baseInterfaces[sel.Sel.Name] {
		return "", "", false
	}
	if len(il.Indices) != 2 {
		return "", "", false
	}
	collectPkgs(il.Indices[0], used)
	collectPkgs(il.Indices[1], used)
	return printNode(fset, il.Indices[0]), printNode(fset, il.Indices[1]), true
}

// buildMethod turns one interface method into a methodSpec, synthesising
// parameter names and classifying the return cardinality.
func buildMethod(fset *token.FileSet, name string, ft *ast.FuncType, used map[string]bool) (methodSpec, bool) {
	q, err := data.ParseMethod(name)
	if err != nil {
		return methodSpec{}, false // not a derived query; skip
	}
	m := methodSpec{Name: name, subject: q.Subject}

	flat := flattenParams(ft.Params)
	for i, p := range flat {
		collectPkgs(p.typ, used)
		if i == 0 {
			continue // the leading context parameter, named ctx below
		}
		m.Params = append(m.Params, param{Name: fmt.Sprintf("a%d", i), Type: printNode(fset, p.typ)})
	}

	if ft.Results != nil {
		flatR := flattenParams(ft.Results)
		m.nResult = len(flatR)
		parts := make([]string, len(flatR))
		for i, r := range flatR {
			collectPkgs(r.typ, used)
			parts[i] = printNode(fset, r.typ)
		}
		// go/printer cannot render a bare *ast.FieldList, so assemble the
		// result list from the individual types. A single result needs no
		// parentheses; zero or many do.
		if len(parts) == 1 {
			m.Results = parts[0]
		} else if len(parts) > 1 {
			m.Results = "(" + strings.Join(parts, ", ") + ")"
		}
		if len(flatR) > 0 {
			_, m.slice = flatR[0].typ.(*ast.ArrayType)
		}
	}
	return m, true
}

type flatParam struct {
	typ ast.Expr
}

func flattenParams(fl *ast.FieldList) []flatParam {
	var out []flatParam
	if fl == nil {
		return out
	}
	for _, f := range fl.List {
		n := len(f.Names)
		if n == 0 {
			n = 1
		}
		for i := 0; i < n; i++ {
			out = append(out, flatParam{typ: f.Type})
		}
	}
	return out
}

// collectPkgs records the package identifier of any selector type so the
// generated file re-imports exactly the packages its signatures reference.
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

// resolveImports maps the used package identifiers back to the source file's
// import specs, dropping any the generated code does not reference.
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
