# GoSpring: 1:1 Spring Boot Port — Roadmap & Module Inventory

Goal: reproduce every major Spring project as a `gospring-*` counterpart with the
same public surface (names, module layout, starter structure, annotation-style
tags, `@EnableX` toggles). This document is the spec: each row is a module we
commit to build.

## Translation rules (how 1:1 maps to Go)

| Spring (JVM) mechanism | GoSpring equivalent | Literal 1:1? |
|---|---|---|
| IoC container / DI | `gs.Provide` + constructor injection (already here) | ✅ behavioral |
| `@Autowired` / `@Component` | wiring via `gs.Object` / `gs.Provide` + struct tags | ✅ behavioral |
| `@ConfigurationProperties` | layered config engine + `gs.Dync[T]` | ✅ already here |
| Starters (`spring-boot-starter-*`) | `starter/starter-*` modules | ✅ already here |
| Annotations (`@Entity`, `@Query`, …) | struct tags + marker comments | ✅ syntactic analog |
| Runtime proxies (Spring Data derived queries) | **build-time codegen** (like `gs-http-gen`) | ⚠️ same API, codegen not proxy |
| Classpath scanning | Go linker visibility + registration-on-import | ⚠️ same effect, explicit import |
| AOP weaving | generated wrappers / middleware | ⚠️ same effect, no weaving |

Everything marked ⚠️ gives the **same developer-facing API**; only the underlying
mechanism differs (codegen at build, not proxies at runtime). This is the only
unavoidable compromise porting to a statically compiled language.

## Module inventory

Status legend: ✅ exists · 🟡 partial (needs a unifying abstraction layer) · ❌ missing

### Core framework (Spring Framework)
| Spring project | GoSpring module | Status | Notes |
|---|---|---|---|
| spring-core (IoC/DI) | `spring/gs` | ✅ | constructor DI |
| spring-context (events, i18n, validation) | `spring/gs` + `stdlib/i18n`, `stdlib/validation` | 🟡 | app-event bus missing |
| spring-expression (SpEL) | `spring/conf` (expr-lang) | 🟡 | expression engine exists, no SpEL-parity API |
| spring-aop | — | ❌ | reproduce via generated interceptors |
| spring-test | `gs.RunTest` + `gs-mock` | ✅ | |

### Spring Boot
| Spring project | GoSpring module | Status |
|---|---|---|
| spring-boot (auto-config, lifecycle) | `spring` | ✅ |
| spring-boot-starter-* | `starter/*` (90+) | ✅ |
| spring-boot-actuator | `cloud/actuator`, `starter-actuator` | ✅ |
| spring-boot-devtools | — | ❌ |

### Web (Spring MVC / WebFlux)
| Spring project | GoSpring module | Status |
|---|---|---|
| spring-webmvc | `starter-gin`/`-echo`/`-http-server` | ✅ (as starters) |
| spring-webflux (reactive) | — | ❌ (Go uses goroutines, not reactive streams) |
| spring-hateoas | `cloud/hateoas` | ✅ Link + Entity/Collection models + HAL `_links` serialisation (affordances pending) |
| spring-graphql | `gospring-graphql` | ❌ |
| spring-web-services (SOAP) | `cloud/ws` | ✅ SOAP 1.1 envelope + `@PayloadRoot` dispatch + typed endpoints + faults + net/http server (WSDL/SOAP 1.2 pending) |
| spring-websocket (transport) | `starter-websocket` / `-coder` | ✅ (as starters) |
| spring-messaging / STOMP | `cloud/stomp` | ✅ frame codec + SimpleBroker + `@MessageMapping`/`@SendTo` dispatcher (connection FSM + external relay pending) |

### Spring Data  ← **first build target**
| Spring project | GoSpring module | Status |
|---|---|---|
| spring-data-commons (`CrudRepository`, `PagingAndSortingRepository`, derived queries, `@Query`, auditing, `Specification`) | `cloud/data` (abstraction) + `gs-data-gen` codegen | ✅ abstraction + derived-query parser + generator done |
| spring-data-jpa | `starter-data-gorm` (binds abstraction → GORM) | ✅ CRUD + paging + derived-query executors, tested over sqlite |
| spring-data-mongodb | `starter-data-mongo` | 🟡 |
| spring-data-redis | `starter-data-redis` | 🟡 go-redis/redigo exist |
| spring-data-elasticsearch | `starter-data-elasticsearch` | 🟡 |
| spring-data-rest | `cloud/datarest` | ✅ exposes any `CrudRepository` as a HAL REST API over net/http (paging query params pending) |

### Spring Security
| Spring project | GoSpring module | Status |
|---|---|---|
| spring-security-core/web (authn/authz, method security) | `cloud/security` (+ `cloud/security/web` filter chain) | ✅ Authentication/SecurityContext/Require + `HttpSecurity`-style SecurityFilterChain DSL (permitAll/authenticated/hasRole, 401/403, bearer resource-server) |
| spring-security-oauth2-client/resource | `starter-*` oauth2 | ✅ |
| spring-authorization-server | oauth2 server | ✅ |
| spring-session | session-redis | ✅ |

### Spring Cloud
| Spring project | GoSpring module | Status |
|---|---|---|
| spring-cloud-config | `starter-config-*` (7 sources) | ✅ |
| spring-cloud-gateway | `gospring-gateway` | 🟡 gateway listed, verify |
| spring-cloud-openfeign | `starter-http-client` + `gs-http-gen` | ✅ |
| spring-cloud-loadbalancer | `cloud/loadbalance` | ✅ |
| spring-cloud-circuitbreaker | `cloud/governance` | ✅ |
| spring-cloud-netflix/discovery | `cloud/discovery` + `starter-registry-*` | ✅ |
| spring-cloud-stream | `cloud/messaging` + MQ starters | ✅ |
| spring-cloud-bus | `starter-config-bus` | ✅ |
| spring-cloud-sleuth / micrometer-tracing | `cloud/observability` + `starter-otel` | ✅ |
| spring-cloud-contract | `gospring-contract` | ❌ |

### Enterprise integration & jobs
| Spring project | GoSpring module | Status |
|---|---|---|
| spring-batch | `cloud/` batch + `starter-scheduler` | 🟡 |
| spring-integration | `cloud/integration` (EIP) | ✅ Message/Channel/DirectChannel + filter/transform/split/route/bridge (aggregator + async channels pending) |
| spring-amqp / spring-kafka / spring-pulsar | `starter-nats` + Kafka/Pulsar/Rabbit/RocketMQ/MQTT | ✅ |
| spring-retry | `cloud/governance` (retry) | ✅ as governance |
| spring-scheduling (`@Scheduled`/`@Async`) | `cloud/scheduling` + `gs-sched-gen` (`@Scheduled` codegen) + `scheduling.Submit`/`Future` (`@Async`) | ✅ codegen + async primitive, proven end to end |
| spring cache abstraction (`@Cacheable`) | `cloud/cache` + `gs-cache-gen` annotations | ✅ `@Cacheable`/`@CachePut`/`@CacheEvict` codegen over `cloud/cache`, proven end to end |

### Spring Modulith  ← **second build target**
| Spring project | GoSpring module | Status |
|---|---|---|
| spring-modulith (module boundaries, allowed-deps verification, module events, docs) | `cloud/modulith` + `gs-modulith` verifier | ✅ model + boundary checker + event bus + `go list` verifier done; docs/async outbox pending |

### Other
| Spring project | GoSpring module | Status |
|---|---|---|
| spring-shell | `cloud/shell` | ✅ command registry + REPL + quote-aware parsing + help/exit (completion/history pending) |
| spring-statemachine | `cloud/statemachine` | ✅ typed FSM: guarded transitions, entry/exit/transition actions, listeners, extended state (hierarchical states pending) |
| spring-rest-docs | `cloud/restdocs` | ✅ test-driven snippet generation (curl/http/fields) with bidirectional field validation (nested paths + asciidoc pending) |

## Build order (each module is the template for the next)

1. **`gospring-data`** — ✅ **DONE.** `cloud/data` abstraction (`Repository[T,ID]`,
   `CrudRepository`, `PagingAndSortingRepository`, `Pageable`, `Sort`) + derived-query
   parser (`ParseMethod`), `starter-data-gorm` binding (`Query`→GORM translator +
   executors), and the `gs-data-gen` generator. Proven end to end over sqlite.
2. **`gospring-modulith`** — ✅ **DONE.** `cloud/modulith` model + boundary checker
   + in-process event bus, and the `gs-modulith` CLI that loads the real import
   graph via `go list` and fails on violations. Proven on a temp module in tests.
3. **`gospring-cache`** — ✅ **DONE.** `cloud/cache.BuildKey` key generator + the
   `gs-cache-gen` generator emitting `@Cacheable`/`@CachePut`/`@CacheEvict`
   cache-aside decorators over `cloud/cache`. Proven end to end against an
   in-memory cache.
4. **`gospring-scheduling`** — ✅ **DONE.** `scheduling.Submit`/`Future` (the
   `@Async` primitive) + the `gs-sched-gen` generator emitting a scheduler
   `Register` function from `//schedule:` directives (fixedRate/fixedDelay/cron).
   Proven end to end: generated registration fires jobs on a real scheduler.
5. **`gospring-statemachine`** — ✅ **DONE.** `cloud/statemachine`: typed FSM with
   a fluent builder, guarded transitions, entry/exit/transition actions,
   listeners and extended-state variables. Fully unit-tested (turnstile, guard
   choice, action-abort, listeners).
6. **`gospring-shell`** — ✅ **DONE.** `cloud/shell`: command registry, REPL over
   injected reader/writer, quote-aware parsing, built-in help/exit. Fully tested.
7. **`gospring-integration`** — ✅ **DONE.** `cloud/integration`: typed Message +
   Channel/DirectChannel and the core EIP endpoints (filter/transform/split/
   route/bridge/handle). Fully tested with a worked multi-stage flow.
8. **`gospring-hateoas`** — ✅ **DONE.** `cloud/hateoas`: Link + Entity/Collection
   models rendering HAL `_links`. Fully tested.
9. **`gospring-data-rest`** — ✅ **DONE.** `cloud/datarest`: any `CrudRepository`
   exposed as a HAL REST API over net/http, composing GoSpring Data + HATEOAS.
10. **`gospring-stomp`** — ✅ **DONE.** `cloud/stomp`: STOMP frame codec,
    destination `SimpleBroker`, `SimpMessagingTemplate` and an `@MessageMapping`
    /`@SendTo` dispatcher. WebSocket transport (starters) plugs in beneath it.
11. **`gospring-restdocs`** — ✅ **DONE.** `cloud/restdocs`: test-driven API
    snippet generation with bidirectional field-coverage validation.
12. **`gospring-security` web filter chain** — ✅ **DONE.** `cloud/security/web`:
    the `HttpSecurity`/`SecurityFilterChain` DSL (ordered ant-matched access
    rules, role/authority checks, 401 vs 403, bearer resource-server) binding the
    existing `cloud/security` primitives into one cohesive module.
13. **`gospring-ws`** — ✅ **DONE.** `cloud/ws`: contract-first SOAP endpoints
    (SOAP 1.1 envelope, `@PayloadRoot` dispatch, typed handlers, faults, server).
14. Fill ❌ rows above, closest-existing-module-as-template each time. Remaining:
    GraphQL (needs an engine decision), WebFlux (reactive — poor fit for Go),
    devtools.

> Conventions every module follows: `starter/DESIGN.md` + per-module `DESIGN`/`USAGE`.
