# cloud/modulith — GoSpring Modulith

The GoSpring port of **Spring Modulith**: an explicit module model over an
application's packages, with allowed-dependency rules and an in-process module
event bus. Container-free and I/O-free — the [`gs-modulith`](../../gs/gs-modulith)
tool loads a real import graph and feeds it to [`Modules.Check`](check.go).

## Mapping to Spring Modulith

| Spring Modulith | `cloud/modulith` |
|---|---|
| `@ApplicationModule` | `Module{Name, BasePackage, AllowedDependencies, Exposed}` |
| `ApplicationModules.of(...)` | `New(mods...)` |
| `ApplicationModules.verify()` | `Modules.Check(graph)` → `[]Violation` |
| sub-packages are internal by default | packages deeper than `BasePackage` are internal unless `Exposed` |
| `allowedDependencies` | `AllowedDependencies` (empty = open) |
| `ApplicationEventPublisher` + `@ApplicationModuleListener` | `Bus` + `Publish` / `Subscribe` |

## Module boundaries

```go
mods := modulith.New(
    modulith.Module{Name: "order", BasePackage: "app/order",
        AllowedDependencies: []string{"catalog", "shared"}},
    modulith.Module{Name: "catalog", BasePackage: "app/catalog",
        Exposed: []string{"app/catalog/api"}},
    modulith.Module{Name: "shared", BasePackage: "app/shared"},
)
violations := mods.Check(graph) // graph: []modulith.Package{ImportPath, Imports}
```

Two kinds of violation are reported: `DisallowedDependency` (the module may not
depend on the target at all) and `InternalAccess` (the dependency is allowed but
the imported package is internal to the target module).

## Module events

```go
type OrderPlaced struct{ ID int64 }

bus := modulith.NewBus()
modulith.Subscribe(bus, func(ctx context.Context, e OrderPlaced) error {
    // catalog reacts to an order module event — no direct call between them
    return nil
})
_ = modulith.Publish(bus, ctx, OrderPlaced{ID: 42})
```

Delivery is synchronous and ordered; subscriber errors are joined.

## Status

- [x] Module model (`Module`, `Modules`, ownership, exposed/internal)
- [x] Boundary checker (`Check` → `Violation`)
- [x] In-process typed event bus (`Bus`, `Publish`, `Subscribe`)
- [x] `gs-modulith` verifier CLI (loads graph via `go list`, fails on violations)
- [ ] Async / transactional-outbox event delivery
- [ ] Module documentation (C4 / diagram) generation
