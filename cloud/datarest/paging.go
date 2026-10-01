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
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"go-spring.org/cloud/data"
	"go-spring.org/cloud/hateoas"
)

// pageable returns the repository as a PagingAndSortingRepository when it is
// one, so the list endpoint can serve paged results.
func (r Resource[T, ID]) pageable() (data.PagingAndSortingRepository[T, ID], bool) {
	p, ok := r.Repo.(data.PagingAndSortingRepository[T, ID])
	return p, ok
}

// parsePageable reads Spring-style ?page=&size=&sort= query parameters. ok is
// false when no size is given, so an un-paged list request keeps its current
// behaviour. sort repeats as ?sort=field or ?sort=field,desc.
func parsePageable(q url.Values) (data.Pageable, bool) {
	if q.Get("size") == "" {
		return data.Pageable{}, false
	}
	size, err := strconv.Atoi(q.Get("size"))
	if err != nil || size <= 0 {
		return data.Pageable{}, false
	}
	page, _ := strconv.Atoi(q.Get("page"))
	return data.PageRequest(page, size, parseSort(q["sort"])), true
}

func parseSort(values []string) data.Sort {
	var s data.Sort
	for _, v := range values {
		field, dir, _ := strings.Cut(v, ",")
		if field == "" {
			continue
		}
		one := data.By(field)
		if strings.EqualFold(dir, "desc") {
			one = one.Descending()
		}
		s = s.And(one)
	}
	return s
}

// listPaged serves one page of entities as a HAL collection with self/next/prev
// navigation links, the analog of spring-data-rest's PagedResourcesAssembler.
func (r Resource[T, ID]) listPaged(w http.ResponseWriter, req *http.Request, repo data.PagingAndSortingRepository[T, ID], p data.Pageable) {
	page, err := repo.FindAllPaged(req.Context(), p)
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	models := make([]*hateoas.EntityModel[T], len(page.Content))
	for i, it := range page.Content {
		models[i] = r.entity(it)
	}
	links := []hateoas.Link{hateoas.Self(r.pageHref(p.PageNumber(), p.PageSize()))}
	if p.PageNumber() > 0 {
		links = append(links, hateoas.NewLink(hateoas.RelPrev, r.pageHref(p.PageNumber()-1, p.PageSize())))
	}
	if page.HasNext() {
		links = append(links, hateoas.NewLink(hateoas.RelNext, r.pageHref(p.PageNumber()+1, p.PageSize())))
	}
	w.Header().Set("X-Total-Count", strconv.FormatInt(page.TotalElements, 10))
	writeJSON(w, http.StatusOK, hateoas.Collection(models, links...))
}

func (r Resource[T, ID]) pageHref(page, size int) string {
	return fmt.Sprintf("/%s?page=%d&size=%d", r.Path, page, size)
}
