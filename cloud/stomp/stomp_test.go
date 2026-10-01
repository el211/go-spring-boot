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
	"encoding/json"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	in := Frame{
		Command: CmdSend,
		Headers: []Header{
			{Key: "destination", Value: "/app/chat"},
			{Key: "content-type", Value: "application/json"},
		},
		Body: []byte(`{"text":"hi"}`),
	}
	out, err := ParseFrame(in.Marshal())
	if err != nil {
		t.Fatal(err)
	}
	if out.Command != CmdSend || out.Get("destination") != "/app/chat" {
		t.Fatalf("parsed = %+v", out)
	}
	if string(out.Body) != `{"text":"hi"}` {
		t.Fatalf("body = %q", out.Body)
	}
}

func TestHeaderEscaping(t *testing.T) {
	in := Frame{Command: CmdMessage, Headers: []Header{{Key: "k", Value: "a:b\nc"}}}
	out, err := ParseFrame(in.Marshal())
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Get("k"); got != "a:b\nc" {
		t.Fatalf("escaped header round-trip = %q", got)
	}
}

func TestMatchDestination(t *testing.T) {
	cases := []struct {
		pattern, dest string
		want          bool
	}{
		{"/topic/news", "/topic/news", true},
		{"/topic/*", "/topic/news", true},
		{"/topic/*", "/topic/news/sports", false},
		{"/topic/**", "/topic/news/sports", true},
		{"/topic/**", "/topic", true},
		{"/queue/a", "/queue/b", false},
	}
	for _, c := range cases {
		if got := matchDestination(c.pattern, c.dest); got != c.want {
			t.Errorf("match(%q,%q) = %v, want %v", c.pattern, c.dest, got, c.want)
		}
	}
}

func TestBrokerDelivery(t *testing.T) {
	b := NewBroker()
	var got []string
	unsub := b.Subscribe("/topic/**", func(m Message) { got = append(got, string(m.Body)) })
	b.Subscribe("/topic/news", func(m Message) { got = append(got, "news:"+string(m.Body)) })

	b.Send(Message{Destination: "/topic/news", Body: []byte("x")})
	// Both the wildcard and the exact subscription match.
	if len(got) != 2 {
		t.Fatalf("delivered %v", got)
	}

	unsub()
	got = nil
	b.Send(Message{Destination: "/topic/news", Body: []byte("y")})
	if len(got) != 1 || got[0] != "news:y" {
		t.Fatalf("after unsubscribe: %v", got)
	}
}

func TestConvertAndSend(t *testing.T) {
	b := NewBroker()
	var payload map[string]any
	b.Subscribe("/topic/p", func(m Message) { _ = json.Unmarshal(m.Body, &payload) })
	if err := b.ConvertAndSend("/topic/p", map[string]int{"n": 7}); err != nil {
		t.Fatal(err)
	}
	if payload["n"] != float64(7) {
		t.Fatalf("payload = %v", payload)
	}
}

func TestDispatchToBroker(t *testing.T) {
	ctx := context.Background()
	b := NewBroker()
	d := NewDispatcher(b)

	// @MessageMapping("/app/greet") @SendTo("/topic/greetings")
	d.Map("/app/greet", func(_ context.Context, msg Message) (any, error) {
		return map[string]string{"reply": "hello " + string(msg.Body)}, nil
	}, "/topic/greetings")

	var received string
	b.Subscribe("/topic/greetings", func(m Message) {
		var r map[string]string
		_ = json.Unmarshal(m.Body, &r)
		received = r["reply"]
	})

	// A client SEND to the app destination, as a STOMP frame.
	send := Frame{Command: CmdSend, Headers: []Header{{Key: "destination", Value: "/app/greet"}}, Body: []byte("Ada")}
	if err := d.DispatchFrame(ctx, send); err != nil {
		t.Fatal(err)
	}
	if received != "hello Ada" {
		t.Fatalf("broker relay = %q", received)
	}
}

func TestDispatchUnknownDestination(t *testing.T) {
	d := NewDispatcher(NewBroker())
	err := d.Dispatch(context.Background(), Message{Destination: "/app/none"})
	if err == nil {
		t.Fatal("expected error for unmapped destination")
	}
}
