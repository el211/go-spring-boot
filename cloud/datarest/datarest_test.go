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

package datarest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// memRepo is an in-memory data.CrudRepository[User, int64] for the test.
type memRepo struct {
	m   map[int64]User
	seq int64
}

func newRepo() *memRepo { return &memRepo{m: map[int64]User{}} }

func (r *memRepo) Save(_ context.Context, u User) (User, error) {
	if u.ID == 0 {
		r.seq++
		u.ID = r.seq
	}
	r.m[u.ID] = u
	return u, nil
}
func (r *memRepo) SaveAll(_ context.Context, us []User) ([]User, error) { return us, nil }
func (r *memRepo) FindById(_ context.Context, id int64) (User, bool, error) {
	u, ok := r.m[id]
	return u, ok, nil
}
func (r *memRepo) ExistsById(_ context.Context, id int64) (bool, error) {
	_, ok := r.m[id]
	return ok, nil
}
func (r *memRepo) FindAll(_ context.Context) ([]User, error) {
	out := make([]User, 0, len(r.m))
	for _, u := range r.m {
		out = append(out, u)
	}
	return out, nil
}
func (r *memRepo) FindAllById(_ context.Context, _ []int64) ([]User, error) { return nil, nil }
func (r *memRepo) Count(_ context.Context) (int64, error)                   { return int64(len(r.m)), nil }
func (r *memRepo) DeleteById(_ context.Context, id int64) error             { delete(r.m, id); return nil }
func (r *memRepo) Delete(_ context.Context, u User) error                   { delete(r.m, u.ID); return nil }
func (r *memRepo) DeleteAllById(_ context.Context, _ []int64) error         { return nil }
func (r *memRepo) DeleteAll(_ context.Context) error                        { clear(r.m); return nil }

func server() (*httptest.Server, *memRepo) {
	repo := newRepo()
	mux := http.NewServeMux()
	Resource[User, int64]{
		Path:    "users",
		Repo:    repo,
		ParseID: Int64ID,
		IDOf:    func(u User) int64 { return u.ID },
	}.Register(mux)
	return httptest.NewServer(mux), repo
}

func TestCreateReadDelete(t *testing.T) {
	srv, _ := server()
	defer srv.Close()

	// Create.
	resp, err := http.Post(srv.URL+"/users", "application/json", strings.NewReader(`{"name":"Ada"}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/users/1" {
		t.Fatalf("Location = %q", loc)
	}
	var created map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&created)
	if created["name"] != "Ada" {
		t.Fatalf("create body = %v", created)
	}
	self := created["_links"].(map[string]any)["self"].(map[string]any)["href"]
	if self != "/users/1" {
		t.Fatalf("self link = %v", self)
	}

	// Read.
	resp, _ = http.Get(srv.URL + "/users/1")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("read status = %d", resp.StatusCode)
	}

	// Read missing -> 404.
	resp, _ = http.Get(srv.URL + "/users/999")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing status = %d", resp.StatusCode)
	}

	// Delete.
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/users/1", nil)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d", resp.StatusCode)
	}
	resp, _ = http.Get(srv.URL + "/users/1")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("after delete status = %d", resp.StatusCode)
	}
}

func TestListHAL(t *testing.T) {
	srv, repo := server()
	defer srv.Close()
	_, _ = repo.Save(context.Background(), User{Name: "A"})
	_, _ = repo.Save(context.Background(), User{Name: "B"})

	resp, _ := http.Get(srv.URL + "/users")
	if ct := resp.Header.Get("Content-Type"); ct != "application/hal+json" {
		t.Fatalf("content-type = %q", ct)
	}
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	content := body["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("list size = %d", len(content))
	}
	// Each embedded entity carries its own self link.
	first := content[0].(map[string]any)
	if _, ok := first["_links"].(map[string]any)["self"]; !ok {
		t.Fatalf("embedded entity missing self link: %v", first)
	}
	if body["_links"].(map[string]any)["self"].(map[string]any)["href"] != "/users" {
		t.Fatalf("collection self link missing: %v", body["_links"])
	}
}

func TestBadIDRejected(t *testing.T) {
	srv, _ := server()
	defer srv.Close()
	resp, _ := http.Get(srv.URL + "/users/not-a-number")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad id status = %d", resp.StatusCode)
	}
}
