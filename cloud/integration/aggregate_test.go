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
	"sort"
	"strings"
	"testing"
)

type part struct {
	order string
	item  string
}

func TestAggregatorReleasesOnSize(t *testing.T) {
	ctx := context.Background()
	in := NewDirectChannel[part]()
	out := NewDirectChannel[string]()

	var released []string
	Collect(out, &released)

	// Group parts by order; release once 2 parts of an order have arrived,
	// combining their items into a sorted, comma-joined summary.
	agg := Aggregate(in, out,
		func(m Message[part]) string { return m.Payload.order },
		2,
		func(parts []part) string {
			items := make([]string, len(parts))
			for i, p := range parts {
				items[i] = p.item
			}
			sort.Strings(items)
			return strings.Join(items, ",")
		})

	_ = SendPayload(ctx, in, part{order: "A", item: "pen"})
	if agg.Pending("A") != 1 {
		t.Fatalf("pending A = %d, want 1", agg.Pending("A"))
	}
	if len(released) != 0 {
		t.Fatalf("released early: %v", released)
	}

	// A second order interleaves; it must not trigger A's release.
	_ = SendPayload(ctx, in, part{order: "B", item: "ink"})
	_ = SendPayload(ctx, in, part{order: "A", item: "cup"})

	if len(released) != 1 || released[0] != "cup,pen" {
		t.Fatalf("released = %v, want [cup,pen]", released)
	}
	if agg.Pending("A") != 0 {
		t.Fatalf("group A not cleared after release: %d", agg.Pending("A"))
	}
	if agg.Pending("B") != 1 {
		t.Fatalf("group B pending = %d, want 1", agg.Pending("B"))
	}
}

func TestAggregatorCorrelationHeader(t *testing.T) {
	ctx := context.Background()
	in := NewDirectChannel[int]()
	out := NewDirectChannel[int]()

	var gotKey string
	out.Subscribe(func(_ context.Context, m Message[int]) error {
		if v, ok := m.Header("correlationId"); ok {
			gotKey = v.(string)
		}
		return nil
	})

	Aggregate(in, out, func(Message[int]) string { return "k" }, 2, func(xs []int) int {
		sum := 0
		for _, x := range xs {
			sum += x
		}
		return sum
	})

	_ = SendPayload(ctx, in, 2)
	_ = SendPayload(ctx, in, 3)
	if gotKey != "k" {
		t.Fatalf("correlationId header = %q, want k", gotKey)
	}
}
