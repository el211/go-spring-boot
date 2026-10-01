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

import (
	"context"
	"errors"
	"testing"
)

func fixture() *Modules {
	return New(
		Module{Name: "order", BasePackage: "app/order", AllowedDependencies: []string{"catalog", "shared"}},
		Module{Name: "catalog", BasePackage: "app/catalog", Exposed: []string{"app/catalog/api"}},
		Module{Name: "shared", BasePackage: "app/shared"},
		Module{Name: "billing", BasePackage: "app/billing"},
	)
}

func TestOwnerMostSpecific(t *testing.T) {
	m := New(
		Module{Name: "order", BasePackage: "app/order"},
		Module{Name: "orderitem", BasePackage: "app/order/item"},
	)
	o, _ := m.Owner("app/order/item/sub")
	if o.Name != "orderitem" {
		t.Fatalf("owner = %q, want orderitem", o.Name)
	}
	if _, ok := m.Owner("strings"); ok {
		t.Fatal("stdlib import should have no owner")
	}
}

func TestCheckDisallowedDependency(t *testing.T) {
	m := fixture()
	// order -> billing is not in order's AllowedDependencies.
	v := m.Check([]Package{{ImportPath: "app/order", Imports: []string{"app/billing"}}})
	if len(v) != 1 || v[0].Kind != DisallowedDependency || v[0].ToModule != "billing" {
		t.Fatalf("violations = %+v", v)
	}
}

func TestCheckInternalAccess(t *testing.T) {
	m := fixture()
	// order may depend on catalog, but app/catalog/internal is not exposed.
	v := m.Check([]Package{{ImportPath: "app/order", Imports: []string{"app/catalog/internal"}}})
	if len(v) != 1 || v[0].Kind != InternalAccess {
		t.Fatalf("violations = %+v", v)
	}
	// ...the exposed api package is fine.
	ok := m.Check([]Package{{ImportPath: "app/order", Imports: []string{"app/catalog/api"}}})
	if len(ok) != 0 {
		t.Fatalf("exposed import flagged: %+v", ok)
	}
}

func TestCheckCleanGraph(t *testing.T) {
	m := fixture()
	v := m.Check([]Package{
		{ImportPath: "app/order", Imports: []string{"app/shared", "app/catalog", "fmt"}},
		{ImportPath: "app/catalog", Imports: []string{"app/shared"}},
	})
	if len(v) != 0 {
		t.Fatalf("expected clean, got %+v", v)
	}
}

type OrderPlaced struct{ ID int }

func TestBus(t *testing.T) {
	b := NewBus()
	var got []int
	Subscribe(b, func(_ context.Context, e OrderPlaced) error { got = append(got, e.ID); return nil })
	Subscribe(b, func(_ context.Context, e OrderPlaced) error { got = append(got, e.ID*10); return nil })

	if err := Publish(b, context.Background(), OrderPlaced{ID: 2}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if len(got) != 2 || got[0] != 2 || got[1] != 20 {
		t.Fatalf("handlers ran wrong: %v", got)
	}

	// Error aggregation.
	Subscribe(b, func(_ context.Context, e OrderPlaced) error { return errors.New("boom") })
	if err := Publish(b, context.Background(), OrderPlaced{ID: 1}); err == nil {
		t.Fatal("expected joined error")
	}

	// Unrelated type with no subscribers is a no-op.
	type Other struct{}
	if err := Publish(b, context.Background(), Other{}); err != nil {
		t.Fatalf("no-subscriber publish: %v", err)
	}
}
