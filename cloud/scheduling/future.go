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

import "context"

// Future is the result of an asynchronous computation — the GoSpring analog of
// Spring's @Async return value (java.util.concurrent.Future / CompletableFuture).
// [Submit] starts work on a goroutine and hands back a Future; the caller does
// other work and later blocks on [Future.Get].
//
// A Future is single-shot: the computation runs once and its (value, error) is
// cached, so Get may be called any number of times and from any goroutine.
type Future[T any] struct {
	done chan struct{}
	val  T
	err  error
}

// Submit runs fn on its own goroutine and returns a [Future] for its result.
// ctx is passed through to fn and also bounds the wait: a [Future.Get] whose
// own context is cancelled returns early even if fn is still running (fn keeps
// running until it observes ctx itself — the GoSpring abstraction never kills a
// goroutine from the outside, matching how the scheduler treats a job).
func Submit[T any](ctx context.Context, fn func(context.Context) (T, error)) *Future[T] {
	f := &Future[T]{done: make(chan struct{})}
	go func() {
		defer close(f.done)
		f.val, f.err = fn(ctx)
	}()
	return f
}

// Get blocks until the computation finishes and returns its result, or returns
// early with ctx.Err() if ctx is cancelled first. The computation's own result
// is still cached and observable by a later Get once it completes.
func (f *Future[T]) Get(ctx context.Context) (T, error) {
	select {
	case <-f.done:
		return f.val, f.err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

// Done reports whether the computation has finished, for a non-blocking poll.
func (f *Future[T]) Done() bool {
	select {
	case <-f.done:
		return true
	default:
		return false
	}
}
