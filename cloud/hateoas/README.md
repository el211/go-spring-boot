# cloud/hateoas — GoSpring HATEOAS

The GoSpring port of **Spring HATEOAS**: it adds hypermedia links to API
resources and serialises them in HAL (the `_links` convention). Transport-
neutral — it produces JSON, not routes.

## Mapping to Spring HATEOAS

| Spring HATEOAS | `cloud/hateoas` |
|---|---|
| `Link` / `IanaLinkRelations` | `Link` + `RelSelf`/`RelNext`/… constants |
| `RepresentationModel` | `RepresentationModel` (embed to make a type linkable) |
| `EntityModel<T>` | `EntityModel[T]` |
| `CollectionModel<T>` | `CollectionModel[T]` |
| `model.add(link)` | `model.Add(link)` |
| HAL `_links` rendering | `MarshalJSON` (HAL) |

## Example

```go
e := hateoas.Entity(user,
    hateoas.Self("/users/1"),
    hateoas.NewLink(hateoas.RelNext, "/users/2"),
)
b, _ := json.Marshal(e)
```

```json
{
  "id": 1,
  "name": "Ada",
  "_links": {
    "self": { "href": "/users/1" },
    "next": { "href": "/users/2" }
  }
}
```

## Semantics

- **HAL hoisting.** `EntityModel` renders the content's own fields at the top
  level next to `_links`, exactly like Spring's HAL serialiser. Non-object
  content (a scalar/array) is nested under `content` instead.
- **Relation arrays.** A relation used once renders as a single link object;
  used more than once it renders as an array — HAL's rule.
- **Omitted when empty.** No links → no `_links` key.
- **Collections.** `CollectionModel` renders `{"content":[...], "_links":{...}}`.

## Status

- [x] Link (templated/type/title), RepresentationModel, Entity/Collection models
- [x] HAL serialisation (hoisting, relation arrays, omit-when-empty)
- [ ] Affordances (HAL-FORMS) and link builders bound to a router
