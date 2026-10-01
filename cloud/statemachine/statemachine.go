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

// Package statemachine is the GoSpring port of Spring Statemachine: a typed
// finite state machine with guarded transitions, entry/exit and transition
// actions, extended-state variables and change listeners.
//
// States and events are two caller-chosen comparable types (typically enums),
// so the machine is type-safe end to end. A [Builder] assembles the transition
// table; the resulting [Machine] is driven by [Machine.Send]. The package is
// container-free and does no I/O.
package statemachine

import (
	"context"
	"sync"
)

// Guard decides whether a transition may fire for a given event, the analog of
// Spring's org.springframework.statemachine.guard.Guard. A nil guard always
// permits.
type Guard[S, E comparable] func(ctx context.Context, ec EventContext[S, E]) bool

// Action runs during a transition or on state entry/exit, the analog of
// Spring's Action. Returning an error aborts the transition, leaving the
// machine in its original state.
type Action[S, E comparable] func(ctx context.Context, ec EventContext[S, E]) error

// EventContext is the per-event data handed to guards and actions — the analog
// of Spring's StateContext.
type EventContext[S, E comparable] struct {
	From    S
	Event   E
	To      S
	Machine *Machine[S, E]
}

// transition is one edge in the table.
type transition[S, E comparable] struct {
	from   S
	event  E
	to     S
	guard  Guard[S, E]
	action Action[S, E]
}

// Machine is a running state machine. It is safe for concurrent [Machine.Send]
// calls; events are applied one at a time. Extended-state variables have their
// own lock so a guard or action may read and write them without deadlocking.
type Machine[S, E comparable] struct {
	mu        sync.Mutex
	current   S
	table     map[S]map[E][]transition[S, E]
	onEntry   map[S][]Action[S, E]
	onExit    map[S][]Action[S, E]
	listeners []func(from, to S, event E)
	varsMu    sync.RWMutex
	vars      map[string]any
}

// State returns the current state.
func (m *Machine[S, E]) State() S {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.current
}

// CanFire reports whether event would cause a transition from the current
// state, evaluating guards against ctx.
func (m *Machine[S, E]) CanFire(ctx context.Context, event E) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.match(ctx, event) != nil
}

// Send applies event. It returns changed=true when a transition fired. A guard
// that rejects every candidate (or no transition for the pair) returns
// (false, nil) — an ignored event, not an error, matching Spring's default
// deferred/ignored handling. An action error aborts the transition and leaves
// the state unchanged.
func (m *Machine[S, E]) Send(ctx context.Context, event E) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	t := m.match(ctx, event)
	if t == nil {
		return false, nil
	}
	ec := EventContext[S, E]{From: m.current, Event: event, To: t.to, Machine: m}

	if err := runAll(ctx, m.onExit[m.current], ec); err != nil {
		return false, err
	}
	if t.action != nil {
		if err := t.action(ctx, ec); err != nil {
			return false, err
		}
	}
	if err := runAll(ctx, m.onEntry[t.to], ec); err != nil {
		return false, err
	}

	from := m.current
	m.current = t.to
	for _, l := range m.listeners {
		l(from, t.to, event)
	}
	return true, nil
}

// match returns the first transition from the current state for event whose
// guard permits, or nil. Caller holds mu.
func (m *Machine[S, E]) match(ctx context.Context, event E) *transition[S, E] {
	for i := range m.table[m.current][event] {
		t := &m.table[m.current][event][i]
		ec := EventContext[S, E]{From: m.current, Event: event, To: t.to, Machine: m}
		if t.guard == nil || t.guard(ctx, ec) {
			return t
		}
	}
	return nil
}

func runAll[S, E comparable](ctx context.Context, actions []Action[S, E], ec EventContext[S, E]) error {
	for _, a := range actions {
		if a == nil {
			continue
		}
		if err := a(ctx, ec); err != nil {
			return err
		}
	}
	return nil
}

// SetVar stores an extended-state variable (Spring's ExtendedState).
func (m *Machine[S, E]) SetVar(key string, val any) {
	m.varsMu.Lock()
	defer m.varsMu.Unlock()
	m.vars[key] = val
}

// Var reads an extended-state variable; ok is false when absent.
func (m *Machine[S, E]) Var(key string) (val any, ok bool) {
	m.varsMu.RLock()
	defer m.varsMu.RUnlock()
	val, ok = m.vars[key]
	return
}
