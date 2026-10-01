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

package example

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestGeneratedRepository exercises the code emitted by gs-data-gen: the
// generated NewOrderRepository and its derived-query methods must compile and
// run against a real database, which is the whole point of build-time codegen.
func TestGeneratedRepository(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(&Order{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	ctx := context.Background()
	repo := NewOrderRepository(db)

	if _, err := repo.SaveAll(ctx, []Order{
		{Customer: "alice", Status: "paid", Total: 100},
		{Customer: "alice", Status: "open", Total: 50},
		{Customer: "bob", Status: "paid", Total: 300},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Derived: FindByCustomer
	alice, err := repo.FindByCustomer(ctx, "alice")
	if err != nil || len(alice) != 2 {
		t.Fatalf("FindByCustomer = %d, %v", len(alice), err)
	}

	// Derived: FindByStatusOrderByTotalDesc
	paid, err := repo.FindByStatusOrderByTotalDesc(ctx, "paid")
	if err != nil || len(paid) != 2 || paid[0].Total != 300 {
		t.Fatalf("FindByStatusOrderByTotalDesc = %+v, %v", paid, err)
	}

	// Derived: CountByStatus
	if n, err := repo.CountByStatus(ctx, "paid"); err != nil || n != 2 {
		t.Fatalf("CountByStatus = %d, %v", n, err)
	}

	// Derived: ExistsByCustomer
	if ok, err := repo.ExistsByCustomer(ctx, "bob"); err != nil || !ok {
		t.Fatalf("ExistsByCustomer = %v, %v", ok, err)
	}

	// Inherited CRUD: FindById
	first, found, err := repo.FindById(ctx, 1)
	if err != nil || !found || first.Customer != "alice" {
		t.Fatalf("FindById = %+v, %v, %v", first, found, err)
	}

	// Derived: DeleteByStatus
	if n, err := repo.DeleteByStatus(ctx, "open"); err != nil || n != 1 {
		t.Fatalf("DeleteByStatus = %d, %v", n, err)
	}
	if n, _ := repo.Count(ctx); n != 2 {
		t.Fatalf("Count after delete = %d", n)
	}
}
