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

package example

import (
	"context"
	"testing"
	"time"

	"go-spring.org/cloud/scheduling"
)

// TestGeneratedRegistration boots a real scheduler with the generated
// RegisterReportTasks and verifies the //schedule: method actually fires.
func TestGeneratedRegistration(t *testing.T) {
	s := scheduling.NewScheduler()
	impl := &Reports{}
	if err := RegisterReportTasks(s, impl); err != nil {
		t.Fatalf("register: %v", err)
	}

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	// With a 20ms fixed rate, several fires should land within this window.
	deadline := time.Now().Add(2 * time.Second)
	for impl.Rollups() < 3 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	stopCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if err := s.Stop(stopCtx); err != nil {
		t.Fatalf("stop: %v", err)
	}

	if impl.Rollups() < 3 {
		t.Fatalf("Rollup fired only %d times; scheduling did not take effect", impl.Rollups())
	}
}
