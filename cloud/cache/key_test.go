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

package cache

import "testing"

func TestBuildKey(t *testing.T) {
	cases := []struct {
		name  string
		parts []any
		want  string
	}{
		{"catalog", nil, "catalog"},
		{"books", []any{"978-0135"}, "books::978-0135"},
		{"books", []any{"Tolkien", 1954}, "books::Tolkien::1954"},
	}
	for _, c := range cases {
		if got := BuildKey(c.name, c.parts...); got != c.want {
			t.Errorf("BuildKey(%q, %v) = %q, want %q", c.name, c.parts, got, c.want)
		}
	}
}
