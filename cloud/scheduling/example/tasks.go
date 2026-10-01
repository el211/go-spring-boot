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

// Package example shows the GoSpring @Scheduled workflow: declare a task
// interface whose methods carry //schedule: directives, then let gs-sched-gen
// emit the RegisterReportTasks function in tasks_sched.go. Only tasks.go is
// hand-written.
package example

import (
	"context"
	"sync/atomic"
)

// ReportTasks declares scheduled background work.
//
//go:generate go run go-spring.org/gs-sched-gen -in tasks.go
type ReportTasks interface {
	//schedule:fixedRate=20ms initialDelay=1ms
	Rollup(ctx context.Context) error
}

// Reports is a ReportTasks implementation that counts how often Rollup fired.
type Reports struct{ rollups atomic.Int64 }

// Rollups returns the number of times Rollup has run.
func (r *Reports) Rollups() int64 { return r.rollups.Load() }

func (r *Reports) Rollup(_ context.Context) error {
	r.rollups.Add(1)
	return nil
}
