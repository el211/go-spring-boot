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

package statemachine

import "context"

// Builder assembles a [Machine]'s transition table fluently, the analog of
// Spring's StateMachineBuilder / StateMachineConfigurer:
//
//	m := statemachine.New[State, Event](Locked).
//	    Permit(Locked, Coin, Unlocked).Do(unlockDoor).
//	    Permit(Unlocked, Push, Locked).Do(lockDoor).
//	    OnEntry(Unlocked, logEntry).
//	    Build()
type Builder[S, E comparable] struct {
	initial   S
	trans     []*transition[S, E]
	last      *transition[S, E]
	onEntry   map[S][]Action[S, E]
	onExit    map[S][]Action[S, E]
	listeners []func(from, to S, event E)
}

// New starts a builder whose machine begins in initial.
func New[S, E comparable](initial S) *Builder[S, E] {
	return &Builder[S, E]{
		initial: initial,
		onEntry: map[S][]Action[S, E]{},
		onExit:  map[S][]Action[S, E]{},
	}
}

// Permit adds a transition from -> to on event. Chain [Builder.When] and
// [Builder.Do] to attach a guard and an action to this transition. Adding
// several transitions for the same (from, event) is allowed; at send time the
// first whose guard permits wins, giving guard-based choice.
func (b *Builder[S, E]) Permit(from S, on E, to S) *Builder[S, E] {
	t := &transition[S, E]{from: from, event: on, to: to}
	b.trans = append(b.trans, t)
	b.last = t
	return b
}

// When attaches a guard to the most recently added transition.
func (b *Builder[S, E]) When(g Guard[S, E]) *Builder[S, E] {
	if b.last != nil {
		b.last.guard = g
	}
	return b
}

// Do attaches an action to the most recently added transition.
func (b *Builder[S, E]) Do(a Action[S, E]) *Builder[S, E] {
	if b.last != nil {
		b.last.action = a
	}
	return b
}

// OnEntry registers an action run whenever state is entered.
func (b *Builder[S, E]) OnEntry(state S, a Action[S, E]) *Builder[S, E] {
	b.onEntry[state] = append(b.onEntry[state], a)
	return b
}

// OnExit registers an action run whenever state is left.
func (b *Builder[S, E]) OnExit(state S, a Action[S, E]) *Builder[S, E] {
	b.onExit[state] = append(b.onExit[state], a)
	return b
}

// OnTransition registers a listener invoked after every state change, the
// analog of a StateMachineListener's stateChanged callback.
func (b *Builder[S, E]) OnTransition(l func(from, to S, event E)) *Builder[S, E] {
	b.listeners = append(b.listeners, l)
	return b
}

// Build freezes the configuration into a ready [Machine] in the initial state.
func (b *Builder[S, E]) Build() *Machine[S, E] {
	table := map[S]map[E][]transition[S, E]{}
	for _, t := range b.trans {
		if table[t.from] == nil {
			table[t.from] = map[E][]transition[S, E]{}
		}
		table[t.from][t.event] = append(table[t.from][t.event], *t)
	}
	return &Machine[S, E]{
		current:   b.initial,
		table:     table,
		onEntry:   b.onEntry,
		onExit:    b.onExit,
		listeners: b.listeners,
		vars:      map[string]any{},
	}
}

// compile-time assurance the action signature is usable as a plain closure.
var _ Action[int, int] = func(context.Context, EventContext[int, int]) error { return nil }
