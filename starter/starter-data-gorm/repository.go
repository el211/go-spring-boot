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
	"errors"

	"go-spring.org/cloud/data"
	"gorm.io/gorm"
)

// Repository is the GORM-backed implementation of [data.CrudRepository] and
// [data.PagingAndSortingRepository] for entity T with primary key ID. A
// generated or hand-written business repository embeds *Repository to inherit
// the whole CRUD surface, then adds its derived-query methods on top (see
// Query* helpers).
//
// Construct one per entity type and expose it as a bean:
//
//	gs.Provide(gormdata.New[User, int64])
type Repository[T any, ID comparable] struct {
	db *gorm.DB
}

// New builds a [Repository] over db. db is the entity's *gorm.DB bean, so
// multi-datasource apps wire the matching instance per entity.
func New[T any, ID comparable](db *gorm.DB) *Repository[T, ID] {
	return &Repository[T, ID]{db: db}
}

// DB returns the underlying handle scoped to T's table, so derived-query
// helpers and ad-hoc callers share the model binding.
func (r *Repository[T, ID]) DB(ctx context.Context) *gorm.DB {
	var zero T
	return r.db.WithContext(ctx).Model(&zero)
}

func (r *Repository[T, ID]) Save(ctx context.Context, entity T) (T, error) {
	err := r.db.WithContext(ctx).Save(&entity).Error
	data.Observe(ctx, "Save", true, err)
	return entity, err
}

func (r *Repository[T, ID]) SaveAll(ctx context.Context, entities []T) ([]T, error) {
	if len(entities) == 0 {
		return entities, nil
	}
	err := r.db.WithContext(ctx).Save(&entities).Error
	data.Observe(ctx, "SaveAll", true, err)
	return entities, err
}

func (r *Repository[T, ID]) FindById(ctx context.Context, id ID) (T, bool, error) {
	var entity T
	err := r.db.WithContext(ctx).First(&entity, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		data.Observe(ctx, "FindById", false, nil)
		return entity, false, nil
	}
	data.Observe(ctx, "FindById", true, err)
	return entity, err == nil, err
}

func (r *Repository[T, ID]) ExistsById(ctx context.Context, id ID) (bool, error) {
	_, found, err := r.FindById(ctx, id)
	return found, err
}

func (r *Repository[T, ID]) FindAll(ctx context.Context) ([]T, error) {
	var out []T
	err := r.db.WithContext(ctx).Find(&out).Error
	data.Observe(ctx, "FindAll", len(out) > 0, err)
	return out, err
}

func (r *Repository[T, ID]) FindAllById(ctx context.Context, ids []ID) ([]T, error) {
	var out []T
	if len(ids) == 0 {
		return out, nil
	}
	err := r.db.WithContext(ctx).Find(&out, ids).Error
	data.Observe(ctx, "FindAllById", len(out) > 0, err)
	return out, err
}

func (r *Repository[T, ID]) Count(ctx context.Context) (int64, error) {
	var n int64
	var zero T
	err := r.db.WithContext(ctx).Model(&zero).Count(&n).Error
	data.Observe(ctx, "Count", true, err)
	return n, err
}

func (r *Repository[T, ID]) DeleteById(ctx context.Context, id ID) error {
	var zero T
	err := r.db.WithContext(ctx).Delete(&zero, id).Error
	data.Observe(ctx, "DeleteById", true, err)
	return err
}

func (r *Repository[T, ID]) Delete(ctx context.Context, entity T) error {
	err := r.db.WithContext(ctx).Delete(&entity).Error
	data.Observe(ctx, "Delete", true, err)
	return err
}

func (r *Repository[T, ID]) DeleteAllById(ctx context.Context, ids []ID) error {
	var zero T
	if len(ids) == 0 {
		return nil
	}
	err := r.db.WithContext(ctx).Delete(&zero, ids).Error
	data.Observe(ctx, "DeleteAllById", true, err)
	return err
}

func (r *Repository[T, ID]) DeleteAll(ctx context.Context) error {
	var zero T
	err := r.db.WithContext(ctx).Where("1 = 1").Delete(&zero).Error
	data.Observe(ctx, "DeleteAll", true, err)
	return err
}

var _ data.CrudRepository[struct{}, int64] = (*Repository[struct{}, int64])(nil)
