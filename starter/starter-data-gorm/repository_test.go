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

package gormdata

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"go-spring.org/cloud/data"
	"gorm.io/gorm"
)

type User struct {
	ID     int64 `gorm:"primaryKey"`
	Email  string
	Status string
	Age    int
}

func newRepo(t *testing.T) *Repository[User, int64] {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return New[User, int64](db)
}

func seed(t *testing.T, r *Repository[User, int64]) {
	t.Helper()
	ctx := context.Background()
	_, err := r.SaveAll(ctx, []User{
		{Email: "a@x.io", Status: "active", Age: 30},
		{Email: "b@x.io", Status: "active", Age: 40},
		{Email: "c@x.io", Status: "banned", Age: 50},
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func TestCrud(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)
	seed(t, r)

	n, err := r.Count(ctx)
	if err != nil || n != 3 {
		t.Fatalf("count = %d, %v", n, err)
	}

	u, found, err := r.FindById(ctx, 1)
	if err != nil || !found || u.Email != "a@x.io" {
		t.Fatalf("findById = %+v found=%v err=%v", u, found, err)
	}

	_, found, _ = r.FindById(ctx, 999)
	if found {
		t.Fatal("findById(999) should miss")
	}

	if err := r.DeleteById(ctx, 3); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if n, _ := r.Count(ctx); n != 2 {
		t.Fatalf("count after delete = %d", n)
	}
}

func TestDerivedQueries(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)
	seed(t, r)

	// FindByEmail -> single
	u, found, err := r.QueryOne(ctx, "FindByEmail", "b@x.io")
	if err != nil || !found || u.Age != 40 {
		t.Fatalf("FindByEmail = %+v found=%v err=%v", u, found, err)
	}

	// FindByStatus -> many
	active, err := r.QueryMany(ctx, "FindByStatus", "active")
	if err != nil || len(active) != 2 {
		t.Fatalf("FindByStatus = %d, %v", len(active), err)
	}

	// FindByAgeGreaterThanEqualOrderByAgeDesc
	older, err := r.QueryMany(ctx, "FindByAgeGreaterThanEqualOrderByAgeDesc", 40)
	if err != nil || len(older) != 2 || older[0].Age != 50 {
		t.Fatalf("GreaterThanEqual+Order = %+v, %v", older, err)
	}

	// CountByStatus
	c, err := r.QueryCount(ctx, "CountByStatus", "active")
	if err != nil || c != 2 {
		t.Fatalf("CountByStatus = %d, %v", c, err)
	}

	// ExistsByEmail
	ex, err := r.QueryExists(ctx, "ExistsByEmail", "c@x.io")
	if err != nil || !ex {
		t.Fatalf("ExistsByEmail = %v, %v", ex, err)
	}

	// DeleteByStatus
	del, err := r.QueryDelete(ctx, "DeleteByStatus", "banned")
	if err != nil || del != 1 {
		t.Fatalf("DeleteByStatus = %d, %v", del, err)
	}
}

func TestPaging(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)
	seed(t, r)

	page, err := r.FindAllPaged(ctx, data.PageRequest(0, 2, data.By("age")))
	if err != nil {
		t.Fatalf("paged: %v", err)
	}
	if page.TotalElements != 3 || page.TotalPages() != 2 || len(page.Content) != 2 {
		t.Fatalf("page = %+v totalPages=%d", page, page.TotalPages())
	}
	if !page.HasNext() {
		t.Fatal("expected HasNext")
	}
	if page.Content[0].Age != 30 {
		t.Fatalf("sort wrong: %+v", page.Content)
	}
}
