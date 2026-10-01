# gs-cache-gen — GoSpring caching generator

The GoSpring port of Spring's caching abstraction (`@Cacheable` / `@CachePut` /
`@CacheEvict`). It reads a service interface whose methods carry `//cache:`
directives and emits a decorator that wraps an implementation with cache-aside
logic over [`cloud/cache`](../../cloud/cache) — a build-time replacement for
Spring's AOP cache interceptor (no proxies, no reflection).

## Run

```bash
go install go-spring.org/gs-cache-gen@latest

//go:generate go run go-spring.org/gs-cache-gen -in service.go
```

Generates `service_cache.go` with a `New<Iface>Cache(inner, c *cache.Cache)`
constructor returning the same interface, so wiring is a one-line swap.

## Directives

Put one directive in a method's doc comment:

| Directive | Spring | Behaviour |
|---|---|---|
| `//cache:cacheable name=<n> [key=<p,...>] [ttl=<dur>]` | `@Cacheable` | serve from cache; on miss call inner and store |
| `//cache:put name=<n> [key=<p,...>] [ttl=<dur>]` | `@CachePut` | always call inner, then store the result |
| `//cache:evict name=<n> [key=<p,...>] [all]` | `@CacheEvict` | call inner, then delete the entry |

- `name` — cache namespace (required).
- `key` — comma-separated parameter names forming the key; omitted = all
  parameters after `ctx`. Keys are built with `cache.BuildKey` (the KeyGenerator
  analog).
- `ttl` — `30s` / `5m` / `2h`; omitted = no expiry.
- `all` (evict) — target the bare-name entry.

`cacheable` and `put` methods must return `(T, error)`. Methods with no
directive are emitted as transparent pass-throughs, so the decorator implements
the whole interface.

## Flags

| Flag | Default | Meaning |
|---|---|---|
| `-in` | *(required)* | input Go file with cache-annotated interfaces |
| `-out` | `<in>_cache.go` | output file |
| `-v` / `-vv` / `--verbose=N` | 0 | log level per `gs/CLAUDE.md` |

See [`cloud/cache/example`](../../cloud/cache/example) for the full, tested
workflow.
