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

package scheduling

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSubmitGetsValue(t *testing.T) {
	f := Submit(context.Background(), func(context.Context) (int, error) {
		return 42, nil
	})
	got, err := f.Get(context.Background())
	if err != nil || got != 42 {
		t.Fatalf("Get = %d, %v", got, err)
	}
	// Second Get returns the cached result.
	if g2, _ := f.Get(context.Background()); g2 != 42 {
		t.Fatalf("cached Get = %d", g2)
	}
}

func TestSubmitPropagatesError(t *testing.T) {
	boom := errors.New("boom")
	f := Submit(context.Background(), func(context.Context) (string, error) {
		return "", boom
	})
	if _, err := f.Get(context.Background()); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
}

func TestGetRespectsCallerContext(t *testing.T) {
	f := Submit(context.Background(), func(context.Context) (int, error) {
		time.Sleep(time.Hour) // never finishes within the test
		return 1, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := f.Get(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want DeadlineExceeded", err)
	}
}

func TestDone(t *testing.T) {
	release := make(chan struct{})
	f := Submit(context.Background(), func(context.Context) (int, error) {
		<-release
		return 7, nil
	})
	if f.Done() {
		t.Fatal("Done true before completion")
	}
	close(release)
	if _, err := f.Get(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !f.Done() {
		t.Fatal("Done false after completion")
	}
}
