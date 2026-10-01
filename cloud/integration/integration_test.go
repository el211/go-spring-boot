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
	"strings"
	"testing"
)

func TestFilterTransformFlow(t *testing.T) {
	ctx := context.Background()
	in := NewDirectChannel[int]()
	evens := NewDirectChannel[int]()
	out := NewDirectChannel[string]()

	Filter(in, evens, func(n int) bool { return n%2 == 0 })
	Transform(evens, out, func(n int) (string, error) { return strings.Repeat("*", n), nil })

	var got []string
	Collect(out, &got)

	for i := 0; i < 5; i++ {
		if err := SendPayload(ctx, in, i); err != nil {
			t.Fatal(err)
		}
	}
	// Only 0, 2, 4 pass the filter.
	want := []string{"", "**", "****"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSplitCarriesHeaders(t *testing.T) {
	ctx := context.Background()
	in := NewDirectChannel[string]()
	words := NewDirectChannel[string]()

	Split(in, words, func(s string) ([]string, error) {
		return strings.Fields(s), nil
	})

	var headers []any
	words.Subscribe(func(_ context.Context, m Message[string]) error {
		if v, ok := m.Header("trace"); ok {
			headers = append(headers, v)
		}
		return nil
	})

	if err := in.Send(ctx, NewMessage("a b c").WithHeader("trace", "xyz")); err != nil {
		t.Fatal(err)
	}
	if len(headers) != 3 {
		t.Fatalf("expected header on each split message, got %v", headers)
	}
}

func TestRoute(t *testing.T) {
	ctx := context.Background()
	in := NewDirectChannel[int]()
	pos := NewDirectChannel[int]()
	neg := NewDirectChannel[int]()

	Route(in, func(m Message[int]) Channel[int] {
		switch {
		case m.Payload > 0:
			return pos
		case m.Payload < 0:
			return neg
		default:
			return nil // drop zero
		}
	})

	var posN, negN int
	Handle(pos, func(context.Context, Message[int]) error { posN++; return nil })
	Handle(neg, func(context.Context, Message[int]) error { negN++; return nil })

	for _, n := range []int{-2, -1, 0, 3, 4} {
		_ = SendPayload(ctx, in, n)
	}
	if posN != 2 || negN != 2 {
		t.Fatalf("routed pos=%d neg=%d (zero should drop)", posN, negN)
	}
}

func TestTransformErrorPropagates(t *testing.T) {
	ctx := context.Background()
	in := NewDirectChannel[int]()
	out := NewDirectChannel[int]()
	Transform(in, out, func(int) (int, error) { return 0, errors.New("bad") })

	if err := SendPayload(ctx, in, 1); err == nil {
		t.Fatal("expected transform error to propagate to sender")
	}
}
