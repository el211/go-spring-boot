# GoSpring Boot — Ported Spring Projects

This fork reproduces Spring's project ecosystem in idiomatic Go. Each module
below mirrors a Spring project's developer-facing API; where Spring relies on
JVM runtime magic (bytecode proxies, classpath scanning, AOP weaving), the
GoSpring equivalent uses **build-time code generation** or explicit wiring
instead — the API matches, the mechanism is static.

The full mapping and remaining work live in [GOSPRING_ROADMAP.md](GOSPRING_ROADMAP.md).

## Modules

| # | Spring project | GoSpring module(s) | How it maps |
|---|---|---|---|
| 1 | Spring Data | [`cloud/data`](cloud/data) + [`starter-data-gorm`](starter/starter-data-gorm) + [`gs-data-gen`](gs/gs-data-gen) | `CrudRepository`/`PagingAndSortingRepository`, derived-query parser, GORM binding, method-name → code generator |
| 2 | Spring Data REST | [`cloud/datarest`](cloud/datarest) | any repository → HAL REST API (CRUD + `?page/size/sort`) over net/http |
| 3 | Spring Modulith | [`cloud/modulith`](cloud/modulith) + [`gs-modulith`](gs/gs-modulith) | module model + event bus; `go list` boundary verifier |
| 4 | Spring Cache | [`cloud/cache`](cloud/cache) + [`gs-cache-gen`](gs/gs-cache-gen) | `@Cacheable`/`@CachePut`/`@CacheEvict` decorator codegen |
| 5 | Spring Scheduling | [`cloud/scheduling`](cloud/scheduling) + [`gs-sched-gen`](gs/gs-sched-gen) | `@Async` (`Future`/`Submit`) + `@Scheduled` registration codegen |
| 6 | Spring StateMachine | [`cloud/statemachine`](cloud/statemachine) | typed FSM: guards, actions, listeners, hierarchical states |
| 7 | Spring Shell | [`cloud/shell`](cloud/shell) | command registry + interactive REPL |
| 8 | Spring Integration | [`cloud/integration`](cloud/integration) | EIP channels + filter/transform/split/route/aggregator |
| 9 | Spring HATEOAS | [`cloud/hateoas`](cloud/hateoas) | `EntityModel`/`CollectionModel` with HAL `_links` |
| 10 | Spring Messaging / STOMP | [`cloud/stomp`](cloud/stomp) | STOMP frames, `SimpleBroker`, `@MessageMapping`/`@SendTo` |
| 11 | Spring REST Docs | [`cloud/restdocs`](cloud/restdocs) | test-driven API snippets with field validation |
| 12 | Spring Security | [`cloud/security`](cloud/security) + [`cloud/security/web`](cloud/security/web) | `Authentication`/`SecurityContext` + `HttpSecurity` filter chain |
| 13 | Spring Web Services | [`cloud/ws`](cloud/ws) | contract-first SOAP endpoints (`@PayloadRoot`, faults) |
| 14 | Spring GraphQL | [`starter-graphql`](starter/starter-graphql) | query/mutation builder + GraphQL-over-HTTP (on graphql-go) |

## The one translation rule

Spring's `findByEmailAndStatus`, `@Cacheable`, `@Scheduled`, `@MessageMapping`
are resolved by the JVM at runtime. In a statically compiled language that is
done at **build time** instead: the `gs-*-gen` tools parse your interfaces and
emit ordinary Go. So the code you write reads like Spring, but there is no
reflection and nothing to discover at startup.

## Conventions every module follows

- Lives in `cloud/` (dependency-light abstraction) or, when it needs a
  third-party engine, in its own `starter/` module — exactly how Spring splits
  core vs. starters. GraphQL (graphql-go) is the one external-engine module.
- Ships a `README.md` with a Spring → GoSpring mapping table.
- Has tests — several proven end to end (Data over sqlite, Data REST / STOMP /
  Security / SOAP / GraphQL over `net/http`, the scheduler firing real jobs).
- Code generators expose the shared `-v`/`-vv` verbosity flags (`gs/CLAUDE.md`).
