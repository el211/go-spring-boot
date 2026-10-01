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

// Package data is the GoSpring port of spring-data-commons: a backend-neutral
// repository abstraction over a persistence store.
//
// It mirrors the Spring Data surface — [CrudRepository] and
// [PagingAndSortingRepository], with [Sort], [Pageable] and [Page] as the
// paging vocabulary — reimagined in idiomatic Go generics. Spring returns an
// absent result as Optional<T>; the Go analog is the (T, bool, error) triple,
// where the bool is false on a miss and error is reserved for real failures.
//
// The package is container-free and imports no backend: a binding starter
// (starter-data-gorm, starter-data-mongo, …) provides a concrete
// [CrudRepository] bean per entity. Derived query methods — Spring's
// findByEmailAndStatus — are not runtime proxies here; a build-time generator
// (gospring-data codegen) emits the method bodies against this abstraction, so
// the developer-facing API matches Spring while the mechanism stays static.
package data

import (
	"context"
	"fmt"
)

// Direction is a sort direction, mirroring org.springframework.data.domain.Sort.Direction.
type Direction string

const (
	// Asc sorts ascending.
	Asc Direction = "ASC"
	// Desc sorts descending.
	Desc Direction = "DESC"
)

// Order is one property/direction pair within a [Sort], mirroring Sort.Order.
type Order struct {
	Property  string
	Direction Direction
}

// Sort is an ordered set of [Order]s. The zero value (no orders) means
// "unsorted", exactly like Sort.unsorted().
type Sort struct {
	orders []Order
}

// By builds an ascending [Sort] over the given properties, mirroring
// Sort.by(String...). Chain [Sort.Descending] to flip direction.
func By(properties ...string) Sort {
	orders := make([]Order, len(properties))
	for i, p := range properties {
		orders[i] = Order{Property: p, Direction: Asc}
	}
	return Sort{orders: orders}
}

// Ascending returns a copy of s with every order forced ascending.
func (s Sort) Ascending() Sort { return s.withDirection(Asc) }

// Descending returns a copy of s with every order forced descending.
func (s Sort) Descending() Sort { return s.withDirection(Desc) }

func (s Sort) withDirection(d Direction) Sort {
	orders := make([]Order, len(s.orders))
	for i, o := range s.orders {
		orders[i] = Order{Property: o.Property, Direction: d}
	}
	return Sort{orders: orders}
}

// And concatenates this sort with another, mirroring Sort.and(Sort).
func (s Sort) And(other Sort) Sort {
	orders := make([]Order, 0, len(s.orders)+len(other.orders))
	orders = append(orders, s.orders...)
	orders = append(orders, other.orders...)
	return Sort{orders: orders}
}

// Orders returns the orders in sort precedence; empty means unsorted.
func (s Sort) Orders() []Order { return s.orders }

// IsSorted reports whether any order is present, mirroring Sort.isSorted().
func (s Sort) IsSorted() bool { return len(s.orders) > 0 }

// Pageable describes a page request — zero-based page number, page size and
// [Sort] — mirroring org.springframework.data.domain.Pageable.
type Pageable struct {
	page int
	size int
	sort Sort
}

// PageRequest builds a [Pageable] for the given zero-based page and size,
// mirroring PageRequest.of(page, size). A size <= 0 is treated as unpaged by
// convention-respecting backends.
func PageRequest(page, size int, sort Sort) Pageable {
	return Pageable{page: page, size: size, sort: sort}
}

// PageNumber returns the zero-based page index.
func (p Pageable) PageNumber() int { return p.page }

// PageSize returns the page size.
func (p Pageable) PageSize() int { return p.size }

// Offset returns the row offset this page starts at (page * size).
func (p Pageable) Offset() int { return p.page * p.size }

// Sort returns the sort to apply to the page.
func (p Pageable) Sort() Sort { return p.sort }

// Next returns the Pageable for the following page.
func (p Pageable) Next() Pageable { return Pageable{page: p.page + 1, size: p.size, sort: p.sort} }

// Page is one slice of a larger result set plus the totals needed to navigate
// it, mirroring org.springframework.data.domain.Page<T>.
type Page[T any] struct {
	Content       []T
	Pageable      Pageable
	TotalElements int64
}

// NewPage assembles a [Page] from the content of one request and the total
// element count across all pages.
func NewPage[T any](content []T, pageable Pageable, total int64) Page[T] {
	return Page[T]{Content: content, Pageable: pageable, TotalElements: total}
}

// TotalPages returns the number of pages available, mirroring Page.getTotalPages().
func (p Page[T]) TotalPages() int {
	size := p.Pageable.PageSize()
	if size <= 0 {
		return 1
	}
	return int((p.TotalElements + int64(size) - 1) / int64(size))
}

// HasNext reports whether a following page exists, mirroring Page.hasNext().
func (p Page[T]) HasNext() bool { return p.Pageable.PageNumber()+1 < p.TotalPages() }

// Repository is the root marker interface, mirroring
// org.springframework.data.repository.Repository<T, ID>. It carries no methods;
// it exists so generated and hand-written repositories share a common root and
// so T/ID are documented at the type level. Being method-free, it is openly
// implementable by binding starters in any package.
type Repository[T any, ID comparable] interface{}

// CrudRepository is the standard CRUD surface, mirroring
// org.springframework.data.repository.CrudRepository<T, ID>. FindById returns
// (zero, false, nil) on a miss — the Go analog of Optional.empty() — reserving
// error for real backend failures.
type CrudRepository[T any, ID comparable] interface {
	// Save inserts or updates entity and returns the stored form (e.g. with a
	// generated id populated).
	Save(ctx context.Context, entity T) (T, error)
	// SaveAll saves every entity, returning them in input order.
	SaveAll(ctx context.Context, entities []T) ([]T, error)
	// FindById returns the entity with the given id; found is false on a miss.
	FindById(ctx context.Context, id ID) (entity T, found bool, err error)
	// ExistsById reports whether an entity with the given id exists.
	ExistsById(ctx context.Context, id ID) (bool, error)
	// FindAll returns every entity.
	FindAll(ctx context.Context) ([]T, error)
	// FindAllById returns the entities for the given ids, skipping misses.
	FindAllById(ctx context.Context, ids []ID) ([]T, error)
	// Count returns the number of entities.
	Count(ctx context.Context) (int64, error)
	// DeleteById deletes by id; deleting an absent id is not an error.
	DeleteById(ctx context.Context, id ID) error
	// Delete deletes the given entity.
	Delete(ctx context.Context, entity T) error
	// DeleteAllById deletes the entities for the given ids.
	DeleteAllById(ctx context.Context, ids []ID) error
	// DeleteAll deletes every entity.
	DeleteAll(ctx context.Context) error
}

// PagingAndSortingRepository extends [CrudRepository] with sorted and paged
// reads, mirroring org.springframework.data.repository.PagingAndSortingRepository.
type PagingAndSortingRepository[T any, ID comparable] interface {
	CrudRepository[T, ID]
	// FindAllSorted returns every entity in the given order.
	FindAllSorted(ctx context.Context, sort Sort) ([]T, error)
	// FindAllPaged returns one page of entities per the [Pageable].
	FindAllPaged(ctx context.Context, pageable Pageable) (Page[T], error)
}

// ErrOptimisticLock reports a version-check failure on save, mirroring
// Spring's OptimisticLockingFailureException. Backends that track a version
// column return this so callers can retry rather than clobber.
var ErrOptimisticLock = fmt.Errorf("data: optimistic locking failure")
