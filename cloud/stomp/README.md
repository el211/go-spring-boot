# cloud/stomp — GoSpring STOMP messaging

The GoSpring port of Spring's **STOMP messaging** (the `spring-messaging` layer
that rides on top of WebSocket). It provides the wire frame codec, a
destination-based broker, a messaging template, and an `@MessageMapping`-style
dispatcher.

The transport (WebSocket) already exists in your fork as the websocket starters;
this package is the **protocol and messaging model above it**, so it is
transport-agnostic and fully testable in memory.

## Mapping to Spring

| Spring | `cloud/stomp` |
|---|---|
| STOMP frame | `Frame` + `Marshal` / `ParseFrame` |
| `SimpleBroker` (`/topic`, `/queue`) | `Broker` (destination subscribe/send, ant matching) |
| `SimpMessagingTemplate.convertAndSend` | `Broker.ConvertAndSend` |
| `@MessageMapping` | `Dispatcher.Map(destination, handler)` |
| `@SendTo` | `Map(..., sendTo)` |
| application vs broker destinations | `Dispatcher` (app) vs `Broker` (`/topic`, `/queue`) |

## Flow

```go
broker := stomp.NewBroker()
disp   := stomp.NewDispatcher(broker)

// @MessageMapping("/app/greet") @SendTo("/topic/greetings")
disp.Map("/app/greet", func(ctx context.Context, m stomp.Message) (any, error) {
    return map[string]string{"reply": "hello " + string(m.Body)}, nil
}, "/topic/greetings")

// A client subscribes to the broker destination…
broker.Subscribe("/topic/greetings", func(m stomp.Message) { send(m) })

// …and the transport hands an inbound SEND frame to the dispatcher:
_ = disp.DispatchFrame(ctx, frame) // runs the handler, relays the result to /topic/greetings
```

## Wiring to WebSocket

A websocket handler decodes each text message with `ParseFrame`:

- `SUBSCRIBE` → `broker.Subscribe(destination, writeFrameToConn)`
- `SEND` to an app destination → `disp.DispatchFrame(ctx, frame)`
- `broker.Send` / `ConvertAndSend` → server-initiated `MESSAGE` frames to subscribers

The RabbitMQ / Kafka starters can back the broker instead of the in-memory
`SimpleBroker` (Spring's "full" external broker relay) — same `Broker`-shaped
seam.

## Status

- [x] STOMP 1.2 frame codec with header escaping
- [x] Destination broker (exact + `*` / `**` ant patterns), messaging template
- [x] `@MessageMapping` dispatcher with `@SendTo` relay
- [ ] Full STOMP connection state machine (CONNECT/receipts/heart-beats)
- [ ] External broker relay (RabbitMQ/Kafka) behind the `Broker` seam
