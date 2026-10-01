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

// Package stomp is the GoSpring port of Spring's STOMP messaging (the
// spring-messaging layer that rides on top of WebSocket). It provides the wire
// [Frame] codec, a destination-based [Broker] (the SimpleBroker analog), a
// [Template] for pushing to subscribers (SimpMessagingTemplate) and a
// [Dispatcher] that routes inbound messages to @MessageMapping-style handlers.
//
// The transport (WebSocket) lives in the websocket starters; this package is
// the protocol and messaging model above it, so it is transport-agnostic and
// fully testable in memory.
package stomp

import (
	"bytes"
	"fmt"
	"strings"
)

// STOMP commands used by this package (STOMP 1.2).
const (
	CmdConnect     = "CONNECT"
	CmdConnected   = "CONNECTED"
	CmdSend        = "SEND"
	CmdSubscribe   = "SUBSCRIBE"
	CmdUnsubscribe = "UNSUBSCRIBE"
	CmdMessage     = "MESSAGE"
	CmdError       = "ERROR"
	CmdDisconnect  = "DISCONNECT"
)

// Header is one STOMP header. Headers are ordered because STOMP keeps the first
// occurrence of a repeated key as authoritative.
type Header struct {
	Key   string
	Value string
}

// Frame is a single STOMP frame: a command, ordered headers and a body.
type Frame struct {
	Command string
	Headers []Header
	Body    []byte
}

// Get returns the value of the first header with key, or "".
func (f Frame) Get(key string) string {
	for _, h := range f.Headers {
		if h.Key == key {
			return h.Value
		}
	}
	return ""
}

// Marshal renders the frame on the wire: COMMAND, headers, a blank line, the
// body and the terminating NUL, per STOMP 1.2. Header keys and values are
// escaped as the spec requires.
func (f Frame) Marshal() []byte {
	var b bytes.Buffer
	b.WriteString(f.Command)
	b.WriteByte('\n')
	for _, h := range f.Headers {
		b.WriteString(escape(h.Key))
		b.WriteByte(':')
		b.WriteString(escape(h.Value))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	b.Write(f.Body)
	b.WriteByte(0)
	return b.Bytes()
}

// ParseFrame parses one STOMP frame, with or without the trailing NUL.
func ParseFrame(data []byte) (Frame, error) {
	data = bytes.TrimSuffix(data, []byte{0})
	head, body, ok := bytes.Cut(data, []byte("\n\n"))
	if !ok {
		return Frame{}, fmt.Errorf("stomp: frame has no header/body separator")
	}
	lines := strings.Split(string(head), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return Frame{}, fmt.Errorf("stomp: frame has no command")
	}
	f := Frame{Command: lines[0], Body: body}
	seen := map[string]bool{}
	for _, ln := range lines[1:] {
		k, v, ok := strings.Cut(ln, ":")
		if !ok {
			return Frame{}, fmt.Errorf("stomp: malformed header %q", ln)
		}
		key := unescape(k)
		if seen[key] { // STOMP: first value of a repeated header wins
			continue
		}
		seen[key] = true
		f.Headers = append(f.Headers, Header{Key: key, Value: unescape(v)})
	}
	return f, nil
}

// escape applies STOMP 1.2 header escaping.
func escape(s string) string {
	r := strings.NewReplacer("\\", "\\\\", "\r", "\\r", "\n", "\\n", ":", "\\c")
	return r.Replace(s)
}

// unescape reverses [escape].
func unescape(s string) string {
	r := strings.NewReplacer("\\r", "\r", "\\n", "\n", "\\c", ":", "\\\\", "\\")
	return r.Replace(s)
}
