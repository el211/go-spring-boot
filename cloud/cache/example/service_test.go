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
	"sync"
	"time"

	"testing"

	"go-spring.org/cloud/cache"
)

// memCache is a minimal in-memory [cache.ByteCache] for the test.
type memCache struct {
	mu sync.Mutex
	m  map[string][]byte
}

func newMem() *memCache { return &memCache{m: map[string][]byte{}} }

func (c *memCache) GetBytes(_ context.Context, key string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	b, ok := c.m[key]
	if !ok {
		return nil, cache.ErrMiss
	}
	return b, nil
}

func (c *memCache) SetBytes(_ context.Context, key string, val []byte, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = val
	return nil
}

func (c *memCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, key)
	return nil
}

func TestGeneratedCacheDecorator(t *testing.T) {
	ctx := context.Background()
	backing := NewStore()
	svc := NewBookServiceCache(backing, cache.New(newMem()))

	if _, err := svc.SaveBook(ctx, "1", &Book{ISBN: "1", Title: "Go"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	// First read: @CachePut on save already populated the cache, so the store
	// is not hit at all.
	b, err := svc.FindBook(ctx, "1")
	if err != nil || b == nil || b.Title != "Go" {
		t.Fatalf("find = %+v, %v", b, err)
	}
	// Second read: served from cache.
	if _, err := svc.FindBook(ctx, "1"); err != nil {
		t.Fatalf("find2: %v", err)
	}
	if backing.Loads() != 0 {
		t.Fatalf("store hit %d times; cache/put should have prevented loads", backing.Loads())
	}

	// A cache miss falls through to the store exactly once, then is cached.
	_, _ = svc.FindBook(ctx, "miss")
	_, _ = svc.FindBook(ctx, "miss")
	if backing.Loads() != 1 {
		t.Fatalf("expected one store load for the miss, got %d", backing.Loads())
	}

	// Evict removes the entry, so the next read reloads from the store.
	if err := svc.DeleteBook(ctx, "1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, _ = svc.FindBook(ctx, "1")
	if backing.Loads() != 2 {
		t.Fatalf("expected reload after evict, got %d loads", backing.Loads())
	}
}
