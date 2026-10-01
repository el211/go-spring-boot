# starter-graphql — GoSpring GraphQL

The GoSpring port of **Spring GraphQL**: a thin, Spring-flavoured layer over the
[graphql-go](https://github.com/graphql-go/graphql) engine. You register query
and mutation fields with resolver functions — the `@QueryMapping` /
`@MutationMapping` analog — and get a built schema and a GraphQL-over-HTTP
endpoint. The engine handles parsing, validation and execution.

This lives in its own module (not in dependency-light `cloud/`) because it pulls
in the GraphQL engine — exactly how Spring ships GraphQL as its own starter.

## Mapping to Spring GraphQL

| Spring GraphQL | `starter-graphql` (package `gql`) |
|---|---|
| `@QueryMapping` | `Builder.Query(Field{...})` |
| `@MutationMapping` | `Builder.Mutation(Field{...})` |
| `DataFetcher` / resolver method | `Resolver func(ctx, args, source)` |
| `GraphQlSource` (schema) | `Builder.Schema()` |
| HTTP endpoint | `Builder.Handler()` |

## Example

```go
b := gql.New()
b.Query(gql.Field{
    Name: "user",
    Type: userType, // a graphql.Object
    Args: graphql.FieldConfigArgument{"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)}},
    Resolve: func(ctx context.Context, args map[string]any, _ any) (any, error) {
        return repo.FindById(ctx, int64(args["id"].(int)))
    },
})
h, _ := b.Handler()
http.Handle("/graphql", h)
```

```
POST /graphql {"query":"{ user(id:1){ id name } }"}
→ 200 {"data":{"user":{"id":1,"name":"Ada"}}}
```

## Semantics

- **Code-first schema.** Types are graphql-go objects; a resolver returns
  `(any, error)` and receives the field args plus the parent `source`.
- **GraphQL-over-HTTP.** POST JSON `{query, variables, operationName}`; execution
  errors appear in the response `errors` array with HTTP 200 (per the spec), and
  only a malformed request body is a 400.
- **Request context.** The HTTP request context flows into every resolver, so
  auth/tracing carried on the context (see `cloud/security`, observability) is
  available to field fetchers.

## Status

- [x] Query/mutation registration, schema build, HTTP endpoint, request context
- [ ] Schema-first SDL (`.graphqls`) loading
- [ ] Subscriptions (WebSocket) and DataLoader-style batching
