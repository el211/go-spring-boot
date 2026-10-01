# starter-data-gorm — GoSpring Data: GORM binding

The GORM backend for [`cloud/data`](../../cloud/data) (GoSpring Data), the port
of **spring-data-jpa**. It implements the backend-neutral `CrudRepository` /
`PagingAndSortingRepository` over GORM and translates a parsed derived-query
`data.Query` into GORM clauses.

## Use

Construct one repository per entity and expose it as a bean:

```go
gs.Provide(gormdata.New[User, int64])
```

`*gormdata.Repository[T, ID]` gives you the whole CRUD + paging surface:

```go
repo := gormdata.New[User, int64](db)
u, found, err := repo.FindById(ctx, 42)
page, err := repo.FindAllPaged(ctx, data.PageRequest(0, 20, data.By("createdAt").Descending()))
```

## Derived queries

Don't hand-write finders — declare them on an interface and let
[`gs-data-gen`](../../gs/gs-data-gen) emit the bodies, which call the `Query*`
executors on this package:

```go
//go:generate go run go-spring.org/gs-data-gen -in repo.go
type UserRepository interface {
    data.CrudRepository[User, int64]
    FindByEmail(ctx context.Context, email string) (User, bool, error)
    FindByStatusOrderByCreatedAtDesc(ctx context.Context, status string) ([]User, error)
}
```

See [`example/`](example) for the full, tested workflow (hand-written `repo.go`
+ generated `repo_gen.go` + a test exercising both derived and inherited CRUD
methods against sqlite).

## Observability

Every operation records `data.operation.total{operation,status}` via
`data.Observe`, naming the logical repository method the SQL layer cannot see.
