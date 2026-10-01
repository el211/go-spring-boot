# cloud/datarest — GoSpring Data REST

The GoSpring port of **spring-data-rest**: it exposes a
[`cloud/data`](../data) `CrudRepository` as a ready-made REST API, with
responses rendered as HAL via [`cloud/hateoas`](../hateoas). It composes the two
building blocks into an out-of-the-box CRUD API over the standard `net/http`
mux — no controllers to write.

## Mapping to spring-data-rest

| spring-data-rest | `cloud/datarest` |
|---|---|
| auto-exposed repository | `Resource[T, ID]` + `Register(mux)` |
| collection / item resource | `GET/POST /{path}`, `GET/PUT/DELETE /{path}/{id}` |
| HAL `_links` / self links | via `cloud/hateoas` |
| id ↔ path conversion | `ParseID` (+ `Int64ID`, `StringID` helpers) |

## Routes

| Method | Path | Repository call |
|---|---|---|
| `GET` | `/{path}` | `FindAll` → `CollectionModel` |
| `POST` | `/{path}` | `Save` → `201` + `Location` |
| `GET` | `/{path}/{id}` | `FindById` → `200` / `404` |
| `PUT` | `/{path}/{id}` | `Save` → `200` |
| `DELETE` | `/{path}/{id}` | `DeleteById` → `204` |

## Example

```go
mux := http.NewServeMux()
datarest.Resource[User, int64]{
    Path:    "users",
    Repo:    userRepo,            // any cloud/data CrudRepository (e.g. the GORM binding)
    ParseID: datarest.Int64ID,
    IDOf:    func(u User) int64 { return u.ID },
}.Register(mux)
http.ListenAndServe(":8080", mux)
```

```
POST /users {"name":"Ada"}      → 201, Location: /users/1
GET  /users/1                   → 200 application/hal+json
    {"id":1,"name":"Ada","_links":{"self":{"href":"/users/1"}}}
```

Drop in the [`starter-data-gorm`](../../starter/starter-data-gorm) repository and
the same four lines expose a database table as a hypermedia API.

## Status

- [x] CRUD routes over any `CrudRepository`, HAL responses, int64/string ids
- [ ] Paging/sorting query params wired to `PagingAndSortingRepository`
- [ ] Exposing derived-query methods as search endpoints
