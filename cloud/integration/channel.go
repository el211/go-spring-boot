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
	"errors"
	"sync"
)

// Handler consumes one message, the analog of a MessageHandler.
type Handler[T any] func(ctx context.Context, msg Message[T]) error

// Channel decouples producers from consumers, the analog of a MessageChannel.
type Channel[T any] interface {
	// Send delivers msg to the channel's subscribers.
	Send(ctx context.Context, msg Message[T]) error
	// Subscribe registers a handler to receive every sent message.
	Subscribe(h Handler[T])
}

// DirectChannel is a synchronous publish-subscribe channel: [DirectChannel.Send]
// invokes every subscriber inline, in subscription order, on the sender's
// goroutine — the analog of Spring's DirectChannel. Subscriber errors are
// joined and returned to the sender, so a flow surfaces failures at the source.
type DirectChannel[T any] struct {
	mu       sync.RWMutex
	handlers []Handler[T]
}

// NewDirectChannel returns an empty [DirectChannel].
func NewDirectChannel[T any]() *DirectChannel[T] {
	return &DirectChannel[T]{}
}

// Subscribe implements [Channel].
func (c *DirectChannel[T]) Subscribe(h Handler[T]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers = append(c.handlers, h)
}

// Send implements [Channel].
func (c *DirectChannel[T]) Send(ctx context.Context, msg Message[T]) error {
	c.mu.RLock()
	hs := make([]Handler[T], len(c.handlers))
	copy(hs, c.handlers)
	c.mu.RUnlock()

	var errs []error
	for _, h := range hs {
		if err := h(ctx, msg); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// SendPayload is a convenience that wraps payload in a [Message] and sends it.
func SendPayload[T any](ctx context.Context, c Channel[T], payload T) error {
	return c.Send(ctx, NewMessage(payload))
}

// Collect subscribes to c and appends every received payload to *dst, a handy
// terminal endpoint for tests and simple sinks. It is not safe for concurrent
// sends; guard dst yourself if the channel is driven from several goroutines.
func Collect[T any](c Channel[T], dst *[]T) {
	c.Subscribe(func(_ context.Context, msg Message[T]) error {
		*dst = append(*dst, msg.Payload)
		return nil
	})
}
