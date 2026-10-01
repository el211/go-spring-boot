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

import (
	"context"
	"errors"
	"testing"
)

type State int
type Event int

const (
	Locked State = iota
	Unlocked
)

const (
	Coin Event = iota
	Push
)

func TestTurnstile(t *testing.T) {
	ctx := context.Background()
	var log []string
	m := New[State, Event](Locked).
		Permit(Locked, Coin, Unlocked).Do(func(context.Context, EventContext[State, Event]) error {
		log = append(log, "unlock")
		return nil
	}).
		Permit(Unlocked, Push, Locked).Do(func(context.Context, EventContext[State, Event]) error {
		log = append(log, "lock")
		return nil
	}).
		OnEntry(Unlocked, func(context.Context, EventContext[State, Event]) error {
			log = append(log, "entry:unlocked")
			return nil
		}).
		OnTransition(func(from, to State, e Event) {
			log = append(log, "changed")
		}).
		Build()

	if m.State() != Locked {
		t.Fatalf("initial = %v", m.State())
	}

	// Push while locked is ignored (no transition).
	if changed, err := m.Send(ctx, Push); changed || err != nil {
		t.Fatalf("push@locked = %v, %v", changed, err)
	}

	if changed, err := m.Send(ctx, Coin); !changed || err != nil {
		t.Fatalf("coin = %v, %v", changed, err)
	}
	if m.State() != Unlocked {
		t.Fatalf("after coin = %v", m.State())
	}
	if changed, _ := m.Send(ctx, Push); !changed || m.State() != Locked {
		t.Fatalf("after push = %v", m.State())
	}

	want := []string{"unlock", "entry:unlocked", "changed", "lock", "changed"}
	if len(log) != len(want) {
		t.Fatalf("log = %v, want %v", log, want)
	}
	for i := range want {
		if log[i] != want[i] {
			t.Fatalf("log[%d] = %q, want %q (%v)", i, log[i], want[i], log)
		}
	}
}

func TestGuardChoice(t *testing.T) {
	ctx := context.Background()
	m := New[State, Event](Locked).
		// Two transitions on the same (Locked, Coin); the guard picks.
		Permit(Locked, Coin, Locked).When(func(_ context.Context, ec EventContext[State, Event]) bool {
		v, _ := ec.Machine.Var("broken")
		return v == true
	}).
		Permit(Locked, Coin, Unlocked).
		Build()

	m.SetVar("broken", true)
	if changed, _ := m.Send(ctx, Coin); !changed || m.State() != Locked {
		t.Fatalf("broken coin -> %v", m.State())
	}

	m.SetVar("broken", false)
	if changed, _ := m.Send(ctx, Coin); !changed || m.State() != Unlocked {
		t.Fatalf("working coin -> %v", m.State())
	}
}

func TestActionErrorAbortsTransition(t *testing.T) {
	ctx := context.Background()
	m := New[State, Event](Locked).
		Permit(Locked, Coin, Unlocked).Do(func(context.Context, EventContext[State, Event]) error {
		return errors.New("jammed")
	}).Build()

	changed, err := m.Send(ctx, Coin)
	if changed || err == nil {
		t.Fatalf("expected abort, got changed=%v err=%v", changed, err)
	}
	if m.State() != Locked {
		t.Fatalf("state changed despite action error: %v", m.State())
	}
}

func TestCanFire(t *testing.T) {
	ctx := context.Background()
	m := New[State, Event](Locked).Permit(Locked, Coin, Unlocked).Build()
	if !m.CanFire(ctx, Coin) {
		t.Error("CanFire(Coin) = false")
	}
	if m.CanFire(ctx, Push) {
		t.Error("CanFire(Push) = true")
	}
}
