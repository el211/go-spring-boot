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
	"reflect"
	"sync"
)

// Bus is an in-process, type-routed event bus — the GoSpring analog of Spring's
// ApplicationEventPublisher paired with @ApplicationModuleListener. Modules
// communicate by publishing typed events instead of calling each other, which
// is how Spring Modulith keeps modules decoupled while still collaborating.
//
// Delivery is synchronous and ordered: [Publish] invokes each subscriber for
// the event type in subscription order and returns their joined error. A Bus is
// safe for concurrent use.
type Bus struct {
	mu       sync.RWMutex
	handlers map[reflect.Type][]func(context.Context, any) error
}

// NewBus returns an empty [Bus].
func NewBus() *Bus {
	return &Bus{handlers: make(map[reflect.Type][]func(context.Context, any) error)}
}

// Subscribe registers handler for events of type E. Multiple handlers for the
// same type are invoked in registration order. The @ApplicationModuleListener
// analog: a module subscribes to events it reacts to.
func Subscribe[E any](b *Bus, handler func(context.Context, E) error) {
	var zero E
	key := reflect.TypeOf(&zero).Elem()
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[key] = append(b.handlers[key], func(ctx context.Context, e any) error {
		return handler(ctx, e.(E))
	})
}

// Publish delivers event to every subscriber of type E, in order, and returns
// their errors joined. With no subscribers it is a no-op returning nil, so a
// module may publish events nothing yet listens for.
func Publish[E any](b *Bus, ctx context.Context, event E) error {
	key := reflect.TypeOf(&event).Elem()
	b.mu.RLock()
	hs := b.handlers[key]
	b.mu.RUnlock()

	var errs []error
	for _, h := range hs {
		if err := h(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
