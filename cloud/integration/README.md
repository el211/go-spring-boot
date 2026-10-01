# cloud/integration — GoSpring Integration

The GoSpring port of **Spring Integration** (Enterprise Integration Patterns):
typed [Message]s flow through [Channel]s and are shaped by composable endpoints.
The building blocks are generic and synchronous by default, so a flow is
ordinary type-checked Go — no reflection, no runtime dispatch table.

## Mapping to Spring Integration

| Spring Integration | `cloud/integration` |
|---|---|
| `Message<T>` + `MessageHeaders` | `Message[T]{Payload, Headers}` |
| `MessageChannel` | `Channel[T]` |
| `DirectChannel` | `DirectChannel[T]` (synchronous pub/sub) |
| `MessageFilter` | `Filter[T]` |
| `Transformer` | `Transform[T, R]` |
| `Splitter` | `Split[T, R]` |
| `Router` | `Route[T]` |
| `<bridge/>` | `Bridge[T]` |
| service-activator | `Handle[T]` |

## Example flow

```go
in    := integration.NewDirectChannel[int]()
evens := integration.NewDirectChannel[int]()
out   := integration.NewDirectChannel[string]()

integration.Filter(in, evens, func(n int) bool { return n%2 == 0 })
integration.Transform(evens, out, func(n int) (string, error) {
    return strconv.Itoa(n), nil
})
integration.Handle(out, func(ctx context.Context, m integration.Message[string]) error {
    log.Println(m.Payload)
    return nil
})

_ = integration.SendPayload(ctx, in, 42) // flows through filter → transform → handler
```

## Semantics

- **Synchronous by default.** `DirectChannel.Send` runs every subscriber inline
  on the caller's goroutine and joins their errors, so a flow surfaces failures
  at the source — the Spring `DirectChannel` contract.
- **Header propagation.** Payload-changing endpoints (transform, split) carry the
  source message's headers onto the new payload; `WithHeader` copies, never
  mutates, so fan-out is safe.
- **Type-safe.** Every channel and endpoint is generic over its payload type;
  payload changes are explicit at the `Transform[T,R]` / `Split[T,R]` boundary.

## Status

- [x] Message + headers, Channel, synchronous DirectChannel
- [x] Filter, Transform, Split, Route, Bridge, Handle endpoints
- [ ] Aggregator / resequencer (stateful correlation)
- [ ] Queue/executor channels (asynchronous delivery) and poller endpoints
