# gs-sched-gen — GoSpring @Scheduled generator

The GoSpring port of Spring's `@Scheduled`. It reads a task interface whose
methods carry `//schedule:` directives and emits a `Register<Iface>` function
that binds each method to a [`cloud/scheduling`](../../cloud/scheduling) trigger
and schedules it — the build-time replacement for Spring's `@Scheduled`
post-processor (no reflection).

(For `@Async`, use the runtime primitive `scheduling.Submit` / `scheduling.Future`
directly — no codegen needed.)

## Run

```bash
go install go-spring.org/gs-sched-gen@latest

//go:generate go run go-spring.org/gs-sched-gen -in tasks.go
```

Generates `tasks_sched.go` with:

```go
func RegisterReportTasks(s *scheduling.Scheduler, impl ReportTasks) error
```

Call it once at startup with your implementation and a scheduler.

## Directives

One directive per method; the method must be `func(ctx context.Context) error`
(the shape `scheduling.NewJob` runs).

| Directive | Spring | Trigger |
|---|---|---|
| `//schedule:fixedRate=5s` | `@Scheduled(fixedRate=...)` | `scheduling.FixedRate` |
| `//schedule:fixedDelay=30s` | `@Scheduled(fixedDelay=...)` | `scheduling.FixedDelay` |
| `//schedule:cron="0 0 * * *"` | `@Scheduled(cron=...)` | `scheduling.ParseCron` |

Options (all optional): `initialDelay=1s`, `jitter=500ms` (fixed* only),
`timeout=10s`, `name=<job name>`. Durations accept `ms`/`s`/`m`/`h`.

## Flags

| Flag | Default | Meaning |
|---|---|---|
| `-in` | *(required)* | input Go file with scheduled interfaces |
| `-out` | `<in>_sched.go` | output file |
| `-v` / `-vv` / `--verbose=N` | 0 | log level per `gs/CLAUDE.md` |

See [`cloud/scheduling/example`](../../cloud/scheduling/example) for the full,
tested workflow.
