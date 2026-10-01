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

package hateoas

import (
	"encoding/json"
	"testing"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func decode(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal %s: %v", b, err)
	}
	return m
}

func TestEntityHoistsContentAndLinks(t *testing.T) {
	e := Entity(User{ID: 1, Name: "Ada"},
		Self("/users/1"),
		NewLink(RelNext, "/users/2"))

	m := decode(t, e)
	if m["id"] != float64(1) || m["name"] != "Ada" {
		t.Fatalf("content not hoisted: %v", m)
	}
	links, ok := m["_links"].(map[string]any)
	if !ok {
		t.Fatalf("_links missing: %v", m)
	}
	self := links["self"].(map[string]any)
	if self["href"] != "/users/1" {
		t.Fatalf("self href = %v", self["href"])
	}
	if links["next"].(map[string]any)["href"] != "/users/2" {
		t.Fatalf("next link wrong: %v", links["next"])
	}
}

func TestMultipleLinksSameRelBecomeArray(t *testing.T) {
	e := Entity(User{ID: 1, Name: "Ada"},
		NewLink(RelItem, "/a"),
		NewLink(RelItem, "/b"))
	m := decode(t, e)
	items, ok := m["_links"].(map[string]any)["item"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("expected item array of 2, got %v", m["_links"])
	}
}

func TestCollectionModel(t *testing.T) {
	c := Collection([]User{{ID: 1, Name: "Ada"}, {ID: 2, Name: "Al"}},
		Self("/users"))
	m := decode(t, c)
	content, ok := m["content"].([]any)
	if !ok || len(content) != 2 {
		t.Fatalf("content = %v", m["content"])
	}
	if m["_links"].(map[string]any)["self"].(map[string]any)["href"] != "/users" {
		t.Fatalf("collection self link missing: %v", m)
	}
}

func TestNoLinksOmitsLinksKey(t *testing.T) {
	m := decode(t, Entity(User{ID: 1, Name: "Ada"}))
	if _, present := m["_links"]; present {
		t.Fatalf("_links should be omitted when empty: %v", m)
	}
}

func TestScalarContentNested(t *testing.T) {
	e := Entity("hello", Self("/greeting"))
	m := decode(t, e)
	if m["content"] != "hello" {
		t.Fatalf("scalar content not nested: %v", m)
	}
	if _, ok := m["_links"]; !ok {
		t.Fatalf("_links missing for scalar entity: %v", m)
	}
}

func TestGetLinkAndTemplated(t *testing.T) {
	e := Entity(User{ID: 1}, NewLink("search", "/users{?q}").AsTemplated())
	l, ok := e.GetLink("search")
	if !ok || !l.Templated {
		t.Fatalf("templated link not found: %+v", l)
	}
}
