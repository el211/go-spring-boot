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

// Package datarest is the GoSpring port of spring-data-rest: it exposes a
// cloud/data CrudRepository as a set of REST endpoints, with responses rendered
// as HAL via cloud/hateoas. It stitches together the two building blocks —
// GoSpring Data (the repository) and GoSpring HATEOAS (the representation) —
// into an out-of-the-box CRUD API over the standard net/http mux.
package datarest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"go-spring.org/cloud/data"
	"go-spring.org/cloud/hateoas"
)

// Resource exposes one entity repository at a URL path. T is the entity, ID its
// key. ParseID converts a path segment to an ID; IDOf extracts an entity's ID
// so self links can be built. Register wires it onto a net/http mux.
type Resource[T any, ID comparable] struct {
	// Path is the collection segment, e.g. "users" (no slashes).
	Path string
	// Repo is the backing repository (any cloud/data CrudRepository).
	Repo data.CrudRepository[T, ID]
	// ParseID parses the {id} path segment into an ID.
	ParseID func(string) (ID, error)
	// IDOf returns an entity's ID, used to build its self link.
	IDOf func(T) ID
}

// Int64ID parses a decimal int64 path segment, for repositories keyed by int64.
func Int64ID(s string) (int64, error) { return strconv.ParseInt(s, 10, 64) }

// StringID is the identity parser, for repositories keyed by string.
func StringID(s string) (string, error) { return s, nil }

// Register mounts the collection and item handlers on mux. With Go's method
// patterns the five Spring-data-rest routes map 1:1:
//
//	GET    /{path}        list
//	POST   /{path}        create
//	GET    /{path}/{id}   read
//	PUT    /{path}/{id}   update
//	DELETE /{path}/{id}   delete
func (r Resource[T, ID]) Register(mux *http.ServeMux) {
	base := "/" + r.Path
	mux.HandleFunc("GET "+base, r.list)
	mux.HandleFunc("POST "+base, r.create)
	mux.HandleFunc("GET "+base+"/{id}", r.read)
	mux.HandleFunc("PUT "+base+"/{id}", r.update)
	mux.HandleFunc("DELETE "+base+"/{id}", r.delete)
}

func (r Resource[T, ID]) selfHref(id ID) string {
	return "/" + r.Path + "/" + idString(id)
}

// entity wraps one record in an EntityModel with its self link.
func (r Resource[T, ID]) entity(v T) *hateoas.EntityModel[T] {
	return hateoas.Entity(v, hateoas.Self(r.selfHref(r.IDOf(v))))
}

func (r Resource[T, ID]) list(w http.ResponseWriter, req *http.Request) {
	// When the client asks for a page (?size=) and the repository supports
	// paging, serve a paged HAL collection instead of the full list.
	if p, ok := parsePageable(req.URL.Query()); ok {
		if repo, ok := r.pageable(); ok {
			r.listPaged(w, req, repo, p)
			return
		}
	}
	items, err := r.Repo.FindAll(req.Context())
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	models := make([]*hateoas.EntityModel[T], len(items))
	for i, it := range items {
		models[i] = r.entity(it)
	}
	writeJSON(w, http.StatusOK, hateoas.Collection(models, hateoas.Self("/"+r.Path)))
}

func (r Resource[T, ID]) read(w http.ResponseWriter, req *http.Request) {
	id, err := r.ParseID(req.PathValue("id"))
	if err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	v, found, err := r.Repo.FindById(req.Context(), id)
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	if !found {
		fail(w, http.StatusNotFound, errors.New("not found"))
		return
	}
	writeJSON(w, http.StatusOK, r.entity(v))
}

func (r Resource[T, ID]) create(w http.ResponseWriter, req *http.Request) {
	v, err := decode[T](req)
	if err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	saved, err := r.Repo.Save(req.Context(), v)
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Location", r.selfHref(r.IDOf(saved)))
	writeJSON(w, http.StatusCreated, r.entity(saved))
}

func (r Resource[T, ID]) update(w http.ResponseWriter, req *http.Request) {
	if _, err := r.ParseID(req.PathValue("id")); err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	v, err := decode[T](req)
	if err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	saved, err := r.Repo.Save(req.Context(), v)
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, r.entity(saved))
}

func (r Resource[T, ID]) delete(w http.ResponseWriter, req *http.Request) {
	id, err := r.ParseID(req.PathValue("id"))
	if err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	if err := r.Repo.DeleteById(req.Context(), id); err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decode[T any](req *http.Request) (T, error) {
	var v T
	err := json.NewDecoder(req.Body).Decode(&v)
	return v, err
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/hal+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func fail(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

// idString renders an ID for a URL path. It avoids fmt for the common scalar
// kinds so links are clean (no %!d surprises) and falls back to Sprint.
func idString(id any) string {
	switch v := id.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return fmt.Sprint(v)
	}
}
