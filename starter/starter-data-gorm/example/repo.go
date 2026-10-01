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

// Package example shows the GoSpring Data workflow end to end: declare a
// repository interface embedding data.CrudRepository with derived-query
// methods, then let gs-data-gen emit the GORM-backed implementation in
// repo_gen.go. Only repo.go is hand-written.
package example

import (
	"context"

	"go-spring.org/cloud/data"
)

// Order is the persisted entity.
type Order struct {
	ID       int64 `gorm:"primaryKey"`
	Customer string
	Status   string
	Total    int
}

// OrderRepository is the business repository. The embedded CrudRepository
// supplies CRUD; the derived methods are filled in by the generator.
//
//go:generate go run go-spring.org/gs-data-gen -in repo.go
type OrderRepository interface {
	data.CrudRepository[Order, int64]

	FindByCustomer(ctx context.Context, customer string) ([]Order, error)
	FindByStatusOrderByTotalDesc(ctx context.Context, status string) ([]Order, error)
	CountByStatus(ctx context.Context, status string) (int64, error)
	ExistsByCustomer(ctx context.Context, customer string) (bool, error)
	DeleteByStatus(ctx context.Context, status string) (int64, error)
}
