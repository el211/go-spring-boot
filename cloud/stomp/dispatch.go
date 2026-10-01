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

package stomp

import (
	"context"
	"fmt"
)

// MappingHandler handles a message addressed to an application destination, the
// analog of an @MessageMapping method. A non-nil return value is published to
// the handler's SendTo destination (see [Dispatcher.Map]); return (nil, nil) to
// publish nothing.
type MappingHandler func(ctx context.Context, msg Message) (any, error)

// Dispatcher routes inbound application messages to handlers and relays their
// results to the broker — the analog of Spring's SimpAnnotationMethodMessage
// handler feeding a SimpleBroker. Application destinations (where clients SEND)
// are handled here; broker destinations (/topic, /queue — where clients
// SUBSCRIBE) are served by the [Broker].
type Dispatcher struct {
	broker   *Broker
	mappings map[string]mapping
}

type mapping struct {
	handler MappingHandler
	sendTo  string
}

// NewDispatcher returns a dispatcher that relays handler results to broker.
func NewDispatcher(broker *Broker) *Dispatcher {
	return &Dispatcher{broker: broker, mappings: map[string]mapping{}}
}

// Map registers handler for an application destination, the analog of
// @MessageMapping(destination). An optional sendTo destination (the @SendTo
// analog) is where a non-nil handler result is published; without it, the
// result is discarded.
func (d *Dispatcher) Map(destination string, handler MappingHandler, sendTo ...string) {
	m := mapping{handler: handler}
	if len(sendTo) > 0 {
		m.sendTo = sendTo[0]
	}
	d.mappings[destination] = m
}

// Dispatch routes msg to the handler registered for its destination. It returns
// an error when no mapping exists (an unroutable client SEND) or the handler
// fails. A non-nil result is JSON-published to the mapping's SendTo.
func (d *Dispatcher) Dispatch(ctx context.Context, msg Message) error {
	m, ok := d.mappings[msg.Destination]
	if !ok {
		return fmt.Errorf("stomp: no @MessageMapping for destination %q", msg.Destination)
	}
	result, err := m.handler(ctx, msg)
	if err != nil {
		return err
	}
	if result != nil && m.sendTo != "" {
		return d.broker.ConvertAndSend(m.sendTo, result)
	}
	return nil
}

// DispatchFrame decodes a STOMP SEND frame into a [Message] and dispatches it,
// so a WebSocket transport can hand raw frames straight to the messaging layer.
func (d *Dispatcher) DispatchFrame(ctx context.Context, f Frame) error {
	if f.Command != CmdSend {
		return fmt.Errorf("stomp: expected %s frame, got %s", CmdSend, f.Command)
	}
	headers := make(map[string]string, len(f.Headers))
	for _, h := range f.Headers {
		headers[h.Key] = h.Value
	}
	return d.Dispatch(ctx, Message{
		Destination: f.Get("destination"),
		Headers:     headers,
		Body:        f.Body,
	})
}
