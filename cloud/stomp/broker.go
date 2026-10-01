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
	"encoding/json"
	"strings"
	"sync"
)

// Message is one application message flowing through the [Broker], decoupled
// from the wire [Frame]. Destination is the logical address (e.g. "/topic/news").
type Message struct {
	Destination string
	Headers     map[string]string
	Body        []byte
}

// Subscriber receives messages delivered to a destination it is subscribed to.
type Subscriber func(Message)

// Broker is an in-memory destination broker — the analog of Spring's
// SimpleBroker. Subscribers register for a destination pattern; [Broker.Send]
// delivers a message to every subscription whose pattern matches the message's
// destination. Safe for concurrent use.
type Broker struct {
	mu   sync.RWMutex
	seq  int
	subs map[int]subscription
}

type subscription struct {
	pattern string
	fn      Subscriber
}

// NewBroker returns an empty [Broker].
func NewBroker() *Broker { return &Broker{subs: map[int]subscription{}} }

// Subscribe registers fn for destination, which may be an exact destination or
// an ant-style pattern ("/topic/*", "/topic/**"). It returns an unsubscribe
// function.
func (b *Broker) Subscribe(destination string, fn Subscriber) (unsubscribe func()) {
	b.mu.Lock()
	id := b.seq
	b.seq++
	b.subs[id] = subscription{pattern: destination, fn: fn}
	b.mu.Unlock()
	return func() {
		b.mu.Lock()
		delete(b.subs, id)
		b.mu.Unlock()
	}
}

// Send delivers msg to every subscription whose pattern matches
// msg.Destination, in no particular order.
func (b *Broker) Send(msg Message) {
	b.mu.RLock()
	var matched []Subscriber
	for _, s := range b.subs {
		if matchDestination(s.pattern, msg.Destination) {
			matched = append(matched, s.fn)
		}
	}
	b.mu.RUnlock()
	for _, fn := range matched {
		fn(msg)
	}
}

// ConvertAndSend JSON-encodes payload and sends it to destination, the analog
// of SimpMessagingTemplate.convertAndSend.
func (b *Broker) ConvertAndSend(destination string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	b.Send(Message{
		Destination: destination,
		Headers:     map[string]string{"content-type": "application/json"},
		Body:        body,
	})
	return nil
}

// matchDestination reports whether an ant-style pattern matches a destination.
// "*" matches exactly one path segment; "**" matches any number of trailing
// segments; otherwise segments must be equal. This mirrors the default
// AntPathMatcher Spring's SimpleBroker uses.
func matchDestination(pattern, dest string) bool {
	if pattern == dest {
		return true
	}
	p := strings.Split(strings.Trim(pattern, "/"), "/")
	d := strings.Split(strings.Trim(dest, "/"), "/")
	for i := 0; i < len(p); i++ {
		if p[i] == "**" {
			return true // matches the rest, including zero segments
		}
		if i >= len(d) {
			return false
		}
		if p[i] != "*" && p[i] != d[i] {
			return false
		}
	}
	return len(p) == len(d)
}
