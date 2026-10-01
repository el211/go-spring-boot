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

package data

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// This file carries the observability seam for the repository abstraction.
// A binding starter wraps the raw query it issues in [Observe], which records
// the logical repository operation — Save, FindById, derived queries — that
// the underlying SQL/driver layer cannot name: at the db client all it sees is
// a SELECT, only this layer knows it was FindByEmailAndStatus. The driver stays
// responsible for duration and db.system; this layer owns the logical result.
//
// The statuses are exclusive, so data.operation.total summed over status is the
// number of repository operations executed. A read distinguishes empty (no row
// matched — the Optional.empty() analog) from ok and error, so a miss is never
// silently counted as a success; writes have ok/error only. Property values and
// arguments never appear in a label: they are unbounded and would explode the
// label space, exactly as key is kept off the cache metric.

const (
	statusOK    = "ok"
	statusEmpty = "empty"
	statusError = "error"
)

// meter and the operation counter are built once; repository calls are frequent
// enough that a per-call meter lookup would show up in profiles. As with the
// cache seam this relies on the OTel global provider being installed before the
// first call — under the framework it is, after RefreshPrepare.
var operationTotal, _ = otel.Meter("go-spring.org/cloud/data").
	Int64Counter("data.operation.total",
		metric.WithDescription("Repository operations executed, by operation and status"),
		metric.WithUnit("{operation}"))

// Observe records one repository operation named op, deriving status from the
// outcome: err != nil is error; otherwise empty when the read matched nothing
// (found == false) and ok in every other case. A binding calls it in a defer:
//
//	func (r *userRepo) FindById(ctx context.Context, id int64) (User, bool, error) {
//	    var u User; var found bool; var err error
//	    defer func() { data.Observe(ctx, "FindById", found, err) }()
//	    ...
//	}
//
// For write operations pass found = true so a successful write is counted ok.
func Observe(ctx context.Context, op string, found bool, err error) {
	status := statusOK
	switch {
	case err != nil:
		status = statusError
	case !found:
		status = statusEmpty
	}
	operationTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("operation", op),
		attribute.String("status", status),
	))
}
