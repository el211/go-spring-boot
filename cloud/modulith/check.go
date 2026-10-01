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

package modulith

import "fmt"

// Package is one package in the application's import graph — the input to
// [Modules.Check]. A loader (the gs-modulith tool, via `go list`) fills these;
// keeping Check graph-in/violations-out makes the rule engine testable with no
// toolchain or filesystem.
type Package struct {
	ImportPath string
	Imports    []string
}

// ViolationKind classifies a boundary breach.
type ViolationKind string

const (
	// DisallowedDependency: the importing module is not allowed to depend on
	// the imported module at all (it is not in AllowedDependencies).
	DisallowedDependency ViolationKind = "disallowed-dependency"
	// InternalAccess: the dependency is allowed, but the imported package is
	// internal to its module (not an exposed API package).
	InternalAccess ViolationKind = "internal-access"
)

// Violation is one broken boundary rule.
type Violation struct {
	Kind        ViolationKind
	FromModule  string
	FromPackage string
	ToModule    string
	ToImport    string
}

// Error renders a violation as a single diagnostic line.
func (v Violation) Error() string {
	switch v.Kind {
	case DisallowedDependency:
		return fmt.Sprintf("%s: module %q (%s) may not depend on module %q (imports %s)",
			v.Kind, v.FromModule, v.FromPackage, v.ToModule, v.ToImport)
	default:
		return fmt.Sprintf("%s: %s imports %s, which is internal to module %q",
			v.Kind, v.FromPackage, v.ToImport, v.ToModule)
	}
}

// Check walks the import graph and returns every boundary violation. A package
// not owned by any module, and any import not owned by any module, are ignored
// — only cross-module edges between declared modules are policed. Violations
// are returned in input order so output is deterministic.
func (m *Modules) Check(pkgs []Package) []Violation {
	var out []Violation
	for _, pkg := range pkgs {
		from, ok := m.Owner(pkg.ImportPath)
		if !ok {
			continue
		}
		for _, imp := range pkg.Imports {
			to, ok := m.Owner(imp)
			if !ok || to.Name == from.Name {
				continue
			}
			switch {
			case !m.Allows(from, to):
				out = append(out, Violation{
					Kind: DisallowedDependency, FromModule: from.Name,
					FromPackage: pkg.ImportPath, ToModule: to.Name, ToImport: imp,
				})
			case !m.Exposes(to, imp):
				out = append(out, Violation{
					Kind: InternalAccess, FromModule: from.Name,
					FromPackage: pkg.ImportPath, ToModule: to.Name, ToImport: imp,
				})
			}
		}
	}
	return out
}
