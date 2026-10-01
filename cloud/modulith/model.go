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

// Package modulith is the GoSpring port of Spring Modulith: an explicit module
// model over an application's packages, with allowed-dependency rules and an
// in-process module event bus. The model here is backend-neutral and does no
// I/O; the gs-modulith tool loads a real import graph and checks it against
// these rules (see [Modules.Check]).
package modulith

import (
	"sort"
	"strings"
)

// Module is one application module, the Go analog of a Spring @ApplicationModule.
// A module owns every package under BasePackage. Only its API packages
// (BasePackage itself plus any listed in Exposed) may be imported by other
// modules; everything deeper is internal, mirroring Spring's "sub-packages are
// internal by default" rule.
type Module struct {
	Name string
	// BasePackage is the import-path prefix the module owns, e.g.
	// "github.com/acme/shop/order".
	BasePackage string
	// AllowedDependencies whitelists the module names this module may depend
	// on. An empty slice means "open" — may depend on any module — matching
	// Spring's default before allowedDependencies is set.
	AllowedDependencies []string
	// Exposed lists API packages beyond BasePackage that other modules may
	// import. BasePackage is always exposed.
	Exposed []string
}

// Modules is a resolved set of [Module]s, indexed for ownership lookup.
type Modules struct {
	ordered []Module
}

// New builds a [Modules] from the given modules, sorted by descending
// BasePackage length so [Modules.Owner] resolves the most specific owner first
// (a nested module wins over an enclosing one).
func New(mods ...Module) *Modules {
	ordered := append([]Module(nil), mods...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return len(ordered[i].BasePackage) > len(ordered[j].BasePackage)
	})
	return &Modules{ordered: ordered}
}

// All returns the modules in declaration-independent (owner-resolution) order.
func (m *Modules) All() []Module { return m.ordered }

// Owner returns the module owning importPath — the one whose BasePackage is the
// longest matching prefix — or ok=false when no module claims it (e.g. a
// third-party or stdlib import, which is never a boundary violation).
func (m *Modules) Owner(importPath string) (Module, bool) {
	for _, mod := range m.ordered {
		if importPath == mod.BasePackage || strings.HasPrefix(importPath, mod.BasePackage+"/") {
			return mod, true
		}
	}
	return Module{}, false
}

// Allows reports whether module from may depend on module to. A module may
// always depend on itself; otherwise to must be whitelisted, unless from is
// open (no AllowedDependencies declared).
func (m *Modules) Allows(from, to Module) bool {
	if from.Name == to.Name {
		return true
	}
	if len(from.AllowedDependencies) == 0 {
		return true
	}
	for _, d := range from.AllowedDependencies {
		if d == to.Name {
			return true
		}
	}
	return false
}

// Exposes reports whether importPath is an API package of its owner mod — the
// base package itself or one listed in Exposed. A deeper package is internal
// and may not be imported from another module.
func (m *Modules) Exposes(mod Module, importPath string) bool {
	if importPath == mod.BasePackage {
		return true
	}
	for _, e := range mod.Exposed {
		if importPath == e || strings.HasPrefix(importPath, e+"/") {
			return true
		}
	}
	return false
}
