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

package gormdata

import (
	"context"
	"sync"

	"go-spring.org/cloud/data"
	"gorm.io/gorm"
)

// parseCache memoizes [data.ParseMethod] per method name: the grammar is fixed
// for a given name, so a generated method re-parses nothing after its first
// call. A bad name panics at first call — a generator bug surfaces in the first
// test rather than silently.
var parseCache sync.Map // map[string]data.Query

func parse(method string) data.Query {
	if q, ok := parseCache.Load(method); ok {
		return q.(data.Query)
	}
	q, err := data.ParseMethod(method)
	if err != nil {
		panic("gormdata: " + err.Error())
	}
	parseCache.Store(method, q)
	return q
}

// FindAllSorted returns every entity in the given order.
func (r *Repository[T, ID]) FindAllSorted(ctx context.Context, sort data.Sort) ([]T, error) {
	var out []T
	db := r.DB(ctx)
	for _, o := range sort.Orders() {
		db = db.Order(column(db, o.Property) + " " + string(o.Direction))
	}
	err := db.Find(&out).Error
	data.Observe(ctx, "FindAllSorted", len(out) > 0, err)
	return out, err
}

// FindAllPaged returns one page per the [data.Pageable], issuing a COUNT for
// the total so [data.Page.TotalPages] is accurate.
func (r *Repository[T, ID]) FindAllPaged(ctx context.Context, pageable data.Pageable) (data.Page[T], error) {
	var out []T
	var total int64
	if err := r.DB(ctx).Count(&total).Error; err != nil {
		data.Observe(ctx, "FindAllPaged", false, err)
		return data.Page[T]{}, err
	}
	db := r.DB(ctx)
	for _, o := range pageable.Sort().Orders() {
		db = db.Order(column(db, o.Property) + " " + string(o.Direction))
	}
	if pageable.PageSize() > 0 {
		db = db.Limit(pageable.PageSize()).Offset(pageable.Offset())
	}
	err := db.Find(&out).Error
	data.Observe(ctx, "FindAllPaged", len(out) > 0, err)
	return data.NewPage(out, pageable, total), err
}

var _ data.PagingAndSortingRepository[struct{}, int64] = (*Repository[struct{}, int64])(nil)

// applied parses method, narrows the model query and binds args.
func (r *Repository[T, ID]) applied(ctx context.Context, method string, args []any) (*gorm.DB, data.Query) {
	q := parse(method)
	return Translate(r.DB(ctx), q, args...), q
}

// QueryMany runs a derived finder returning a slice, e.g. FindByStatus.
func (r *Repository[T, ID]) QueryMany(ctx context.Context, method string, args ...any) ([]T, error) {
	var out []T
	db, _ := r.applied(ctx, method, args)
	err := db.Find(&out).Error
	data.Observe(ctx, method, len(out) > 0, err)
	return out, err
}

// QueryOne runs a derived finder returning at most one row, e.g. FindByEmail.
// found is false when nothing matched.
func (r *Repository[T, ID]) QueryOne(ctx context.Context, method string, args ...any) (T, bool, error) {
	var out T
	db, _ := r.applied(ctx, method, args)
	err := db.Limit(1).Find(&out).Error
	found := err == nil && db.RowsAffected > 0
	data.Observe(ctx, method, found, err)
	return out, found, err
}

// QueryCount runs a derived counter, e.g. CountByStatus.
func (r *Repository[T, ID]) QueryCount(ctx context.Context, method string, args ...any) (int64, error) {
	var n int64
	db, _ := r.applied(ctx, method, args)
	err := db.Count(&n).Error
	data.Observe(ctx, method, true, err)
	return n, err
}

// QueryExists runs a derived existence check, e.g. ExistsByEmail.
func (r *Repository[T, ID]) QueryExists(ctx context.Context, method string, args ...any) (bool, error) {
	n, err := r.QueryCount(ctx, method, args...)
	return n > 0, err
}

// QueryDelete runs a derived delete, e.g. DeleteByStatus, returning rows affected.
func (r *Repository[T, ID]) QueryDelete(ctx context.Context, method string, args ...any) (int64, error) {
	var zero T
	db, _ := r.applied(ctx, method, args)
	res := db.Delete(&zero)
	data.Observe(ctx, method, true, res.Error)
	return res.RowsAffected, res.Error
}
