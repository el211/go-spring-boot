# gs-data-gen — GoSpring Data generator

The build-time counterpart to Spring Data's runtime repository proxy. It reads a
Go file declaring repository interfaces that embed `data.CrudRepository[T, ID]`
(or `PagingAndSortingRepository`) and emits, for each, a backend-bound struct
whose derived-query methods delegate to the backend's `Query*` executors.

Spring derives `findByEmailAndStatus` from the method name at runtime; this does
the same parse (reusing `cloud/data.ParseMethod`, so the grammar is single-
sourced) at build time and writes static Go — no reflection, no proxies.

## Install / run

```bash
go install go-spring.org/gs-data-gen@latest

# or via go:generate next to the interface:
//go:generate go run go-spring.org/gs-data-gen -in repo.go
```

## Flags

| Flag | Default | Meaning |
|---|---|---|
| `-in` | *(required)* | input Go file declaring repository interfaces |
| `-out` | `<in>_gen.go` | output file |
| `-backend` | `go-spring.org/starter-data-gorm` | backend import path |
| `-backend-alias` | `gormdata` | backend package alias in generated code |
| `-v` / `-vv` / `--verbose=N` | 0 | log level (per `gs/CLAUDE.md`): step lines; `-v` adds argv; `-vv` adds cwd + per-method detail |

## Method-name grammar

Reuses the full `cloud/data` derived-query grammar: subjects
`Find`/`Get`/`Read`/`Query`, `Count`, `Exists`, `Delete`/`Remove`; operators
`Is`/`Equals`, `Not`, `LessThan(Equal)`, `GreaterThan(Equal)`, `Like`/`NotLike`,
`Containing`/`StartingWith`/`EndingWith`, `In`/`NotIn`, `Between`,
`IsNull`/`IsNotNull`, `True`/`False`; `And`/`Or`; `Distinct`, `First`/`Top<N>`,
`OrderBy…`. The executor is chosen from the subject and the method's return
shape (slice → `QueryMany`, single → `QueryOne`, etc.).

See [`starter-data-gorm/example`](../../starter/starter-data-gorm/example) for a
generated file and its end-to-end test.
