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

import "context"

// The endpoints below are the core Enterprise Integration Patterns. Each
// subscribes to an input [Channel] and forwards to an output [Channel], so a
// flow is assembled by wiring channels together. Payload-changing endpoints
// (transform, split) carry the source message's headers onto the new payload.

// Bridge forwards every message from src to dst unchanged (a pass-through
// connector), the analog of Spring's <bridge/>.
func Bridge[T any](src, dst Channel[T]) {
	src.Subscribe(func(ctx context.Context, msg Message[T]) error {
		return dst.Send(ctx, msg)
	})
}

// Filter forwards a message from src to dst only when pred(payload) is true,
// the analog of a MessageFilter. Rejected messages are silently dropped.
func Filter[T any](src, dst Channel[T], pred func(T) bool) {
	src.Subscribe(func(ctx context.Context, msg Message[T]) error {
		if pred(msg.Payload) {
			return dst.Send(ctx, msg)
		}
		return nil
	})
}

// Transform maps each src payload to a new payload on dst via fn, the analog of
// a Transformer. A fn error aborts that message and propagates to the sender.
func Transform[T, R any](src Channel[T], dst Channel[R], fn func(T) (R, error)) {
	src.Subscribe(func(ctx context.Context, msg Message[T]) error {
		out, err := fn(msg.Payload)
		if err != nil {
			return err
		}
		return dst.Send(ctx, mapHeaders(msg, out))
	})
}

// Split expands each src payload into zero or more dst payloads via fn, the
// analog of a Splitter. Each produced item becomes its own message, inheriting
// the source headers.
func Split[T, R any](src Channel[T], dst Channel[R], fn func(T) ([]R, error)) {
	src.Subscribe(func(ctx context.Context, msg Message[T]) error {
		items, err := fn(msg.Payload)
		if err != nil {
			return err
		}
		for _, it := range items {
			if err := dst.Send(ctx, mapHeaders(msg, it)); err != nil {
				return err
			}
		}
		return nil
	})
}

// Route sends each src message to the channel chosen by pick, the analog of a
// Router. A pick returning nil drops the message (no default channel), matching
// Spring's drop-on-no-channel option.
func Route[T any](src Channel[T], pick func(Message[T]) Channel[T]) {
	src.Subscribe(func(ctx context.Context, msg Message[T]) error {
		if dst := pick(msg); dst != nil {
			return dst.Send(ctx, msg)
		}
		return nil
	})
}

// Handle attaches a terminal message handler to src, the analog of a
// service-activator at the end of a flow.
func Handle[T any](src Channel[T], h Handler[T]) {
	src.Subscribe(h)
}
