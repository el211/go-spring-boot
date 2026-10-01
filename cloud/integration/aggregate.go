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

package integration

import (
	"context"
	"sync"
)

// Aggregator collects correlated messages and releases a single combined
// message once a group is complete — the analog of Spring Integration's
// Aggregator (correlation strategy + release strategy + aggregation function).
//
// correlationOf groups messages (Spring's CorrelationStrategy); a group is
// released when it reaches releaseSize messages (a size-based ReleaseStrategy);
// combine folds the group's payloads into the released payload. It is safe for
// concurrent sends.
type Aggregator[T, R any] struct {
	dst           Channel[R]
	correlationOf func(Message[T]) string
	releaseSize   int
	combine       func([]T) R

	mu     sync.Mutex
	groups map[string][]T
}

// Aggregate wires an [Aggregator] from src to dst and returns it (handy for
// inspecting pending groups in tests). Each released group is sent to dst as one
// message whose correlation id is carried in the "correlationId" header.
func Aggregate[T, R any](src Channel[T], dst Channel[R], correlationOf func(Message[T]) string, releaseSize int, combine func([]T) R) *Aggregator[T, R] {
	a := &Aggregator[T, R]{
		dst:           dst,
		correlationOf: correlationOf,
		releaseSize:   releaseSize,
		combine:       combine,
		groups:        map[string][]T{},
	}
	src.Subscribe(a.receive)
	return a
}

func (a *Aggregator[T, R]) receive(ctx context.Context, msg Message[T]) error {
	key := a.correlationOf(msg)

	a.mu.Lock()
	a.groups[key] = append(a.groups[key], msg.Payload)
	if len(a.groups[key]) < a.releaseSize {
		a.mu.Unlock()
		return nil
	}
	group := a.groups[key]
	delete(a.groups, key)
	a.mu.Unlock()

	// Release outside the lock so a downstream send cannot deadlock the group.
	out := NewMessage(a.combine(group)).WithHeader("correlationId", key)
	return a.dst.Send(ctx, out)
}

// Pending returns the number of not-yet-released messages buffered for key,
// exposing partial-group state for diagnostics and tests.
func (a *Aggregator[T, R]) Pending(key string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.groups[key])
}
