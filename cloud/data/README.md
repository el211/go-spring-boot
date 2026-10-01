# cloud/data — GoSpring Data

The GoSpring port of **spring-data-commons**: a backend-neutral repository
abstraction over a persistence store. It is container-free and imports no
backend — a binding starter (`starter-data-gorm`, `starter-data-mongo`, …)
provides a concrete repository bean per entity.

## Mapping to Spring Data

| Spring Data | `cloud/data` |
|---|---|
| `Repository<T, ID>` | `Repository[T, ID comparable]` (marker) |
| `CrudRepository<T, ID>` | `CrudRepository[T, ID]` |
| `PagingAndSortingRepository<T, ID>` | `PagingAndSortingRepository[T, ID]` |
| `Optional<T>` return | `(T, found bool, err error)` triple |
| `Sort` / `Sort.Order` / `Direction` | `Sort` / `Order` / `Direction` |
| `Pageable` / `PageRequest.of(...)` | `Pageable` / `PageRequest(...)` |
| `Page<T>` | `Page[T]` |
| `PartTree` (derived-query parser) | `ParseMethod` → `Query` |
| `OptimisticLockingFailureException` | `ErrOptimisticLock` |

## Derived queries: codegen, not proxies

Spring derives a query from a method name like `findByEmailAndStatus` at runtime
and builds a proxy. GoSpring does the same parse — see `ParseMethod`, which
produces a structured `Query` — at **build time**, and a generator emits a
static method body against this abstraction. The developer-facing API is
identical; only the mechanism (codegen vs. runtime proxy) differs, because Go is
statically compiled.

Supported method-name grammar (same keywords as Spring):

- **Subjects**: `Find`/`Get`/`Read`/`Query`, `Count`, `Exists`, `Delete`/`Remove`
- **Operators**: `Is`/`Equals`, `Not`, `LessThan(Equal)`, `GreaterThan(Equal)`,
  `Like`/`NotLike`, `Containing`/`StartingWith`/`EndingWith`, `In`/`NotIn`,
  `Between`, `IsNull`/`IsNotNull`, `True`/`False`
- **Connectors**: `And`, `Or`
- **Modifiers**: `Distinct`, `First`/`Top<N>`, `OrderBy<Prop>(Asc|Desc)…`

```go
// FindByEmailAndStatusOrderByCreatedAtDesc  parses to:
//   Subject=Find  Criteria=[email Eq, status Eq]  Connectors=[And]
//   Sort=[createdAt DESC]  BindCount=2
q, _ := data.ParseMethod("FindByEmailAndStatusOrderByCreatedAtDesc")
```

## Target repository shape

A business repository embeds `CrudRepository` and declares derived methods; the
generator fills the bodies:

```go
type User struct {
    ID     int64  `gorm:"primaryKey"`
    Email  string
    Status string
}

//go:generate gospring-data gen
type UserRepository interface {
    data.CrudRepository[User, int64]

    FindByEmail(ctx context.Context, email string) (User, bool, error)
    FindByStatusOrderByCreatedAtDesc(ctx context.Context, status string) ([]User, error)
    CountByStatus(ctx context.Context, status string) (int64, error)
}
```

## Observability

Bindings call `data.Observe(ctx, op, found, err)` to record
`data.operation.total{operation,status}` — the logical repository operation the
SQL layer cannot name. Status is `ok` / `empty` (no row matched) / `error`; a
miss is never counted as a success. Property values never appear as labels.

## Status

- [x] Core abstraction (`CrudRepository`, `PagingAndSortingRepository`, `Sort`, `Pageable`, `Page`)
- [x] Derived-query parser (`ParseMethod` → `Query`)
- [x] Observability seam (`Observe`)
- [ ] `starter-data-gorm` — first backend binding + translator (`Query` → GORM)
- [ ] `gospring-data` generator — emit method bodies from `//go:generate`
- [ ] Further bindings: mongo, redis, elasticsearch
