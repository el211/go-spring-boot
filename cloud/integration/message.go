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

// Package integration is the GoSpring port of Spring Integration (Enterprise
// Integration Patterns): typed [Message]s flow through [Channel]s and are
// shaped by composable endpoints — filter, transform, split, route. The
// building blocks are generic and synchronous by default, so a flow is ordinary
// type-checked Go with no reflection and no runtime message dispatch table.
package integration

// Message is a payload plus headers, the analog of
// org.springframework.messaging.Message. Headers carry out-of-band metadata
// (correlation ids, routing keys) without polluting the payload type.
type Message[T any] struct {
	Payload T
	Headers map[string]any
}

// NewMessage wraps payload in a header-less [Message].
func NewMessage[T any](payload T) Message[T] {
	return Message[T]{Payload: payload}
}

// WithHeader returns a copy of m with key set to val. The header map is copied
// so the original message is never mutated — safe to fan out to several
// endpoints.
func (m Message[T]) WithHeader(key string, val any) Message[T] {
	h := make(map[string]any, len(m.Headers)+1)
	for k, v := range m.Headers {
		h[k] = v
	}
	h[key] = val
	return Message[T]{Payload: m.Payload, Headers: h}
}

// Header reads a header; ok is false when absent.
func (m Message[T]) Header(key string) (val any, ok bool) {
	val, ok = m.Headers[key]
	return
}

// mapHeaders carries a source message's headers onto a new payload of type R,
// used by endpoints that change the payload type (transform, split).
func mapHeaders[T, R any](src Message[T], payload R) Message[R] {
	return Message[R]{Payload: payload, Headers: src.Headers}
}
