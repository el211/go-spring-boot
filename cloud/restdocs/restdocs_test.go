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

package restdocs

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// handler under documentation: echoes a user as JSON.
func userHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"id":1,"name":"Ada"}`))
}

func capture(t *testing.T) Capture {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/users/1", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	userHandler(rec, req)
	return FromRecorder(req, nil, rec)
}

func TestDocumentWritesSnippets(t *testing.T) {
	dir := t.TempDir()
	d := New(dir)
	err := d.Document("get-user", capture(t),
		WithResponseFields(
			Field("id", "The user id").WithType("Number"),
			Field("name", "The user name").WithType("String"),
		))
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range []string{"curl-request.md", "http-request.md", "http-response.md", "response-fields.md"} {
		if _, err := os.Stat(filepath.Join(dir, "get-user", f)); err != nil {
			t.Errorf("missing snippet %s: %v", f, err)
		}
	}

	curl := read(t, dir, "get-user", "curl-request.md")
	if !strings.Contains(curl, "curl '/users/1' -i -X GET") {
		t.Errorf("curl snippet wrong:\n%s", curl)
	}
	resp := read(t, dir, "get-user", "http-response.md")
	if !strings.Contains(resp, "HTTP/1.1 200 OK") || !strings.Contains(resp, `"name":"Ada"`) {
		t.Errorf("response snippet wrong:\n%s", resp)
	}
	fields := read(t, dir, "get-user", "response-fields.md")
	if !strings.Contains(fields, "| `id` | Number | The user id |") {
		t.Errorf("fields table wrong:\n%s", fields)
	}
}

func TestUndocumentedFieldFails(t *testing.T) {
	d := New(t.TempDir())
	err := d.Document("partial", capture(t),
		WithResponseFields(Field("id", "The id"))) // "name" left undocumented
	if err == nil || !strings.Contains(err.Error(), "undocumented") {
		t.Fatalf("expected undocumented-field error, got %v", err)
	}
}

func TestMissingDocumentedFieldFails(t *testing.T) {
	d := New(t.TempDir())
	err := d.Document("extra", capture(t),
		WithResponseFields(
			Field("id", "The id"),
			Field("name", "The name"),
			Field("email", "The email"), // not in payload and not optional
		))
	if err == nil || !strings.Contains(err.Error(), "not present") {
		t.Fatalf("expected missing-field error, got %v", err)
	}
}

func TestOptionalFieldExempt(t *testing.T) {
	d := New(t.TempDir())
	err := d.Document("opt", capture(t),
		WithResponseFields(
			Field("id", "The id"),
			Field("name", "The name"),
			Field("email", "The email").AsOptional(),
		))
	if err != nil {
		t.Fatalf("optional missing field should not fail: %v", err)
	}
}

func read(t *testing.T, parts ...string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(parts...))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
