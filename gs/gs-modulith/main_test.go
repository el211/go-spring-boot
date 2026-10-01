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
	"os"
	"path/filepath"
	"testing"
)

// writeModule lays down a tiny two-module app in dir: order imports billing,
// which the config forbids. Returns the config path.
func writeModule(t *testing.T, dir string, forbid bool) string {
	t.Helper()
	write := func(rel, content string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module app\n\ngo 1.26\n")
	write("billing/billing.go", "package billing\n\nfunc Charge() {}\n")
	write("order/order.go", "package order\n\nimport \"app/billing\"\n\nfunc Place() { billing.Charge() }\n")

	// A non-empty whitelist that excludes billing forbids the dependency; an
	// empty/undeclared list means "open" (may depend on any module).
	allowed := `["billing"]`
	if forbid {
		allowed = `["catalog"]`
	}
	cfg := `{"modules":[
		{"name":"order","basePackage":"app/order","allowedDependencies":` + allowed + `},
		{"name":"billing","basePackage":"app/billing"}
	]}`
	write("modulith.json", cfg)
	return filepath.Join(dir, "modulith.json")
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := writeModule(t, dir, false)
	mods, err := loadConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if o, ok := mods.Owner("app/order/sub"); !ok || o.Name != "order" {
		t.Fatalf("owner = %+v ok=%v", o, ok)
	}
}

func TestRunDetectsViolation(t *testing.T) {
	dir := t.TempDir()
	cfg := writeModule(t, dir, true) // order forbidden from billing

	err := run(cfg, dir)
	if err == nil {
		t.Fatal("expected a boundary violation error")
	}
}

func TestRunCleanWhenAllowed(t *testing.T) {
	dir := t.TempDir()
	cfg := writeModule(t, dir, false) // order allowed to use billing

	if err := run(cfg, dir); err != nil {
		t.Fatalf("expected clean, got %v", err)
	}
}
