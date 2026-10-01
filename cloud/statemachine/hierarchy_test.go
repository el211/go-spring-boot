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
	"testing"
)

// Media-player style hierarchy: Playing and Paused are substates of On. The
// powerOff event is declared once on the On superstate and applies to both.
const (
	On State = iota + 10
	Playing
	Paused
	Off
)

const (
	play Event = iota + 10
	pause
	powerOff
)

func player() *Machine[State, Event] {
	return New[State, Event](Playing).
		Substate(Playing, On).
		Substate(Paused, On).
		Permit(Playing, pause, Paused).
		Permit(Paused, play, Playing).
		// Shared transition on the superstate:
		Permit(On, powerOff, Off).
		Build()
}

func TestSubstateInheritsSuperstateTransition(t *testing.T) {
	ctx := context.Background()

	m := player()
	// powerOff is not defined on Playing; it must bubble to On.
	if changed, err := m.Send(ctx, powerOff); !changed || err != nil {
		t.Fatalf("powerOff from Playing = %v, %v", changed, err)
	}
	if m.State() != Off {
		t.Fatalf("state = %v, want Off", m.State())
	}

	// Same shared transition works from the other substate.
	m = player()
	if _, err := m.Send(ctx, pause); err != nil {
		t.Fatal(err)
	}
	if m.State() != Paused {
		t.Fatalf("state = %v, want Paused", m.State())
	}
	if changed, _ := m.Send(ctx, powerOff); !changed || m.State() != Off {
		t.Fatalf("powerOff from Paused -> %v", m.State())
	}
}

func TestOwnTransitionWinsOverInherited(t *testing.T) {
	ctx := context.Background()
	m := player()
	// Playing defines its own 'pause'; it should be used, not bubbled.
	if _, err := m.Send(ctx, pause); err != nil {
		t.Fatal(err)
	}
	if m.State() != Paused {
		t.Fatalf("own transition not taken: %v", m.State())
	}
}

func TestUnhandledEventStillIgnored(t *testing.T) {
	ctx := context.Background()
	m := player()
	// 'play' is not valid from Playing, nor from On → ignored.
	if changed, err := m.Send(ctx, play); changed || err != nil {
		t.Fatalf("play@Playing = %v, %v", changed, err)
	}
}
