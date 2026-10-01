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

package gql

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/graphql-go/graphql"
)

type user struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

var userType = graphql.NewObject(graphql.ObjectConfig{
	Name: "User",
	Fields: graphql.Fields{
		"id":   &graphql.Field{Type: graphql.Int},
		"name": &graphql.Field{Type: graphql.String},
	},
})

func builder() *Builder {
	store := map[int]user{1: {ID: 1, Name: "Ada"}}
	b := New()
	b.Query(Field{
		Name: "user",
		Type: userType,
		Args: graphql.FieldConfigArgument{"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)}},
		Resolve: func(_ context.Context, args map[string]any, _ any) (any, error) {
			u, ok := store[args["id"].(int)]
			if !ok {
				return nil, nil
			}
			return u, nil
		},
	})
	b.Mutation(Field{
		Name: "addUser",
		Type: userType,
		Args: graphql.FieldConfigArgument{"name": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)}},
		Resolve: func(_ context.Context, args map[string]any, _ any) (any, error) {
			u := user{ID: len(store) + 1, Name: args["name"].(string)}
			store[u.ID] = u
			return u, nil
		},
	})
	return b
}

func exec(t *testing.T, h http.Handler, query string) map[string]any {
	t.Helper()
	body, _ := json.Marshal(request{Query: query})
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(string(body)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v (%s)", err, rec.Body.String())
	}
	return out
}

func TestQuery(t *testing.T) {
	h, err := builder().Handler()
	if err != nil {
		t.Fatal(err)
	}
	out := exec(t, h, `{ user(id: 1) { id name } }`)
	if out["errors"] != nil {
		t.Fatalf("errors: %v", out["errors"])
	}
	u := out["data"].(map[string]any)["user"].(map[string]any)
	if u["name"] != "Ada" {
		t.Fatalf("user = %v", u)
	}
}

func TestMutation(t *testing.T) {
	h, _ := builder().Handler()
	out := exec(t, h, `mutation { addUser(name: "Al") { id name } }`)
	if out["errors"] != nil {
		t.Fatalf("errors: %v", out["errors"])
	}
	u := out["data"].(map[string]any)["addUser"].(map[string]any)
	if u["name"] != "Al" {
		t.Fatalf("addUser = %v", u)
	}
}

func TestValidationErrorInBody(t *testing.T) {
	h, _ := builder().Handler()
	// Unknown field → GraphQL validation error, still HTTP 200 with errors.
	out := exec(t, h, `{ user(id: 1) { nope } }`)
	if out["errors"] == nil {
		t.Fatalf("expected validation errors, got %v", out)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	h, _ := builder().Handler()
	req := httptest.NewRequest(http.MethodGet, "/graphql", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status = %d, want 405", rec.Code)
	}
}

func TestSchemaRequiresQueryOrMutation(t *testing.T) {
	if _, err := New().Handler(); err == nil {
		t.Fatal("empty builder should fail to build a schema")
	}
}
