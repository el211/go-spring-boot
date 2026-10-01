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

// Package example shows the GoSpring caching workflow (@Cacheable port): declare
// a service interface whose methods carry //cache: directives, then let
// gs-cache-gen emit the cache-aside decorator in service_cache.go. Only
// service.go is hand-written.
package example

import "context"

// Book is the cached value.
type Book struct {
	ISBN  string
	Title string
}

// BookService is the business interface. The decorator generated from the
// //cache: directives adds cache-aside behaviour around a plain implementation.
//
//go:generate go run go-spring.org/gs-cache-gen -in service.go
type BookService interface {
	//cache:cacheable name=books key=isbn
	FindBook(ctx context.Context, isbn string) (*Book, error)

	//cache:put name=books key=isbn
	SaveBook(ctx context.Context, isbn string, b *Book) (*Book, error)

	//cache:evict name=books key=isbn
	DeleteBook(ctx context.Context, isbn string) error
}

// store is a trivial BookService that counts how often FindBook hits the
// backing store, so a test can prove the cache short-circuits the second call.
type store struct {
	books map[string]*Book
	loads int
}

// NewStore returns a bare (uncached) BookService.
func NewStore() *store { return &store{books: map[string]*Book{}} }

// Loads reports how many times FindBook reached the store.
func (s *store) Loads() int { return s.loads }

func (s *store) FindBook(_ context.Context, isbn string) (*Book, error) {
	s.loads++
	return s.books[isbn], nil
}

func (s *store) SaveBook(_ context.Context, isbn string, b *Book) (*Book, error) {
	s.books[isbn] = b
	return b, nil
}

func (s *store) DeleteBook(_ context.Context, isbn string) error {
	delete(s.books, isbn)
	return nil
}
