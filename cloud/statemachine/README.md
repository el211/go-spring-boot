# cloud/statemachine — GoSpring StateMachine

The GoSpring port of **Spring Statemachine**: a typed finite state machine with
guarded transitions, entry/exit and transition actions, extended-state
variables and change listeners. States and events are two caller-chosen
`comparable` types, so the machine is type-safe end to end. Container-free, no
I/O.

## Mapping to Spring Statemachine

| Spring Statemachine | `cloud/statemachine` |
|---|---|
| `StateMachineBuilder` / configurer | `New[S,E](initial)` + fluent `Builder` |
| `transition().source().event().target()` | `Permit(from, on, to)` |
| `Guard` | `When(Guard)` / `Guard[S,E]` |
| `Action` (transition) | `Do(Action)` |
| entry / exit actions | `OnEntry` / `OnExit` |
| `StateMachineListener.stateChanged` | `OnTransition(func(from, to, event))` |
| `ExtendedState` | `SetVar` / `Var` |
| `sendEvent` | `Send(ctx, event)` |

## Example (turnstile)

```go
type State int
type Event int
const (Locked State = iota; Unlocked)
const (Coin Event = iota; Push)

m := statemachine.New[State, Event](Locked).
    Permit(Locked, Coin, Unlocked).Do(unlock).
    Permit(Unlocked, Push, Locked).Do(lock).
    OnEntry(Unlocked, logEntry).
    OnTransition(func(from, to State, e Event) { /* audit */ }).
    Build()

changed, err := m.Send(ctx, Coin) // true, nil → now Unlocked
m.Send(ctx, Push)                  // back to Locked
```

## Semantics

- **Ignored events.** `Send` returns `(false, nil)` when no transition matches
  the (state, event) pair, or every candidate's guard rejects — events are
  ignored, not errors, matching Spring's default.
- **Guard-based choice.** Several transitions may share a (from, event); the
  first whose guard permits wins.
- **Atomic transitions.** On a fire: exit actions → transition action → entry
  actions → state change → listeners. If any action returns an error, the
  transition aborts and the machine stays in its original state.
- **Concurrency.** `Send` is serialised by a mutex; extended-state variables use
  a separate lock so guards/actions can read and write them safely.

## Status

- [x] Typed states/events, guarded transitions, transition/entry/exit actions
- [x] Listeners and extended-state variables
- [ ] Hierarchical (nested) states and regions (parallel states)
- [ ] Persistence / state-machine recovery
