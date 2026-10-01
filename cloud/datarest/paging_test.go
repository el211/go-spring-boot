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
	"sort"
	"testing"

	"go-spring.org/cloud/data"
)

// The paging methods promote memRepo to a PagingAndSortingRepository. Content is
// ordered by id so the test is deterministic despite the backing map.
func (r *memRepo) FindAllSorted(ctx context.Context, _ data.Sort) ([]User, error) {
	return r.ordered(), nil
}

func (r *memRepo) FindAllPaged(_ context.Context, p data.Pageable) (data.Page[User], error) {
	all := r.ordered()
	start := p.Offset()
	if start > len(all) {
		start = len(all)
	}
	end := start + p.PageSize()
	if end > len(all) {
		end = len(all)
	}
	return data.NewPage(all[start:end], p, int64(len(all))), nil
}

func (r *memRepo) ordered() []User {
	out := make([]User, 0, len(r.m))
	for _, u := range r.m {
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func TestPagedList(t *testing.T) {
	srv, repo := server()
	defer srv.Close()
	for i := 0; i < 5; i++ {
		_, _ = repo.Save(context.Background(), User{Name: "u"})
	}

	// First page of 2.
	resp, _ := http.Get(srv.URL + "/users?page=0&size=2")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if tc := resp.Header.Get("X-Total-Count"); tc != "5" {
		t.Fatalf("X-Total-Count = %q", tc)
	}
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if got := len(body["content"].([]any)); got != 2 {
		t.Fatalf("page size = %d, want 2", got)
	}
	links := body["_links"].(map[string]any)
	if _, ok := links["next"]; !ok {
		t.Fatalf("first page should have next: %v", links)
	}
	if _, ok := links["prev"]; ok {
		t.Fatalf("first page should not have prev: %v", links)
	}

	// Last page: ids 5 only (page 2, size 2 -> items 5).
	resp, _ = http.Get(srv.URL + "/users?page=2&size=2")
	_ = json.NewDecoder(resp.Body).Decode(&body)
	links = body["_links"].(map[string]any)
	if _, ok := links["next"]; ok {
		t.Fatalf("last page should not have next: %v", links)
	}
	if _, ok := links["prev"]; !ok {
		t.Fatalf("last page should have prev: %v", links)
	}
}

func TestUnpagedListStillWorks(t *testing.T) {
	srv, repo := server()
	defer srv.Close()
	_, _ = repo.Save(context.Background(), User{Name: "a"})
	// No ?size= → full list path.
	resp, _ := http.Get(srv.URL + "/users")
	if resp.StatusCode != http.StatusOK || resp.Header.Get("X-Total-Count") != "" {
		t.Fatalf("unpaged should not set total count; status=%d", resp.StatusCode)
	}
}
