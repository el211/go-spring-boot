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

// Package gormdata is the GORM binding of cloud/data (GoSpring Data): it backs
// the backend-neutral CrudRepository / PagingAndSortingRepository abstraction
// with GORM, and translates a parsed derived-query [data.Query] into the
// equivalent GORM clauses so generated repository methods run as plain SQL.
package gormdata

import (
	"fmt"
	"strings"

	"go-spring.org/cloud/data"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// Translate applies a parsed [data.Query]'s predicate and ordering to db,
// binding args to the predicate placeholders in parse order. It returns the
// narrowed *gorm.DB ready for a terminal Find/First/Count/Delete; the caller
// chooses the terminal because it depends on the method's subject and return
// shape. A mismatch between the query's BindCount and len(args) is a generator
// bug and panics, exactly as a wrong proxy binding would fail fast in Spring.
func Translate(db *gorm.DB, q data.Query, args ...any) *gorm.DB {
	if q.BindCount() != len(args) {
		panic(fmt.Sprintf("gormdata: query binds %d args but %d supplied", q.BindCount(), len(args)))
	}
	if q.Distinct {
		db = db.Distinct()
	}
	at := 0
	for i, c := range q.Criteria {
		expr, vals := clause(db, c, args[at:at+c.Args])
		at += c.Args
		if i == 0 {
			db = db.Where(expr, vals...)
			continue
		}
		// Connectors[i-1] joins Criteria[i-1] and Criteria[i].
		if q.Connectors[i-1] == data.ConnectorOr {
			db = db.Or(expr, vals...)
		} else {
			db = db.Where(expr, vals...)
		}
	}
	for _, o := range q.Sort.Orders() {
		db = db.Order(column(db, o.Property) + " " + string(o.Direction))
	}
	if q.Limit > 0 {
		db = db.Limit(q.Limit)
	}
	return db
}

// clause renders one criterion into a GORM placeholder expression plus the
// argument values it consumes.
func clause(db *gorm.DB, c data.Criterion, args []any) (string, []any) {
	col := column(db, c.Property)
	switch c.Operator {
	case data.OpEquals:
		return col + " = ?", args
	case data.OpNotEquals:
		return col + " <> ?", args
	case data.OpLessThan:
		return col + " < ?", args
	case data.OpLessThanEqual:
		return col + " <= ?", args
	case data.OpGreaterThan:
		return col + " > ?", args
	case data.OpGreaterThanEqual:
		return col + " >= ?", args
	case data.OpLike:
		return col + " LIKE ?", args
	case data.OpNotLike:
		return col + " NOT LIKE ?", args
	case data.OpContaining:
		return col + " LIKE ?", []any{"%" + fmt.Sprint(args[0]) + "%"}
	case data.OpStartingWith:
		return col + " LIKE ?", []any{fmt.Sprint(args[0]) + "%"}
	case data.OpEndingWith:
		return col + " LIKE ?", []any{"%" + fmt.Sprint(args[0])}
	case data.OpIn:
		return col + " IN ?", args
	case data.OpNotIn:
		return col + " NOT IN ?", args
	case data.OpBetween:
		return col + " BETWEEN ? AND ?", args
	case data.OpIsNull:
		return col + " IS NULL", nil
	case data.OpIsNotNull:
		return col + " IS NOT NULL", nil
	case data.OpTrue:
		return col + " = ?", []any{true}
	case data.OpFalse:
		return col + " = ?", []any{false}
	default:
		panic(fmt.Sprintf("gormdata: unknown operator %v", c.Operator))
	}
}

// column maps an entity property (camelCase, as the parser lower-cases it) to
// the backing column name using GORM's configured naming strategy, so the
// translation honours whatever convention the model uses. It falls back to
// GORM's default snake_case when no strategy is configured.
func column(db *gorm.DB, property string) string {
	var ns schema.Namer = schema.NamingStrategy{}
	if db != nil && db.NamingStrategy != nil {
		ns = db.NamingStrategy
	}
	return ns.ColumnName("", strings.ToUpper(property[:1])+property[1:])
}
