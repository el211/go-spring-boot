# cloud/restdocs — GoSpring REST Docs

The GoSpring port of **Spring REST Docs**: it turns a real HTTP exchange
captured in a test into documentation snippets (curl request, HTTP
request/response, field tables) written to disk. Because the docs are generated
from tests that must pass — and field coverage is validated both ways — the
documentation cannot drift from the actual API.

## Mapping to Spring REST Docs

| Spring REST Docs | `cloud/restdocs` |
|---|---|
| `MockMvc` + `document(...)` | `Documenter.Document(name, Capture, ...)` |
| generated snippets dir | `New(dir)` output tree |
| `curl-request` / `http-request` / `http-response` | same snippet files (Markdown) |
| `fieldWithPath(...)` | `Field(path, desc)` |
| `requestFields` / `responseFields` | `WithRequestFields` / `WithResponseFields` |
| undocumented/missing field fails the test | bidirectional validation in `fieldTable` |

## Example (in a test)

```go
req := httptest.NewRequest("GET", "/users/1", nil)
rec := httptest.NewRecorder()
handler.ServeHTTP(rec, req)

docs := restdocs.New("build/snippets")
err := docs.Document("get-user", restdocs.FromRecorder(req, nil, rec),
    restdocs.WithResponseFields(
        restdocs.Field("id", "The user id").WithType("Number"),
        restdocs.Field("name", "The user name").WithType("String"),
    ))
```

Writes `build/snippets/get-user/{curl-request,http-request,http-response,response-fields}.md`,
ready to include into a hand-written guide.

## Validation contract

As in Spring, field coverage is checked both ways and a gap fails the test:

- a non-optional documented field absent from the payload → error (`not present`);
- an actual payload field with no descriptor → error (`undocumented`);
- `Field(...).AsOptional()` exempts a field from the presence check.

## Status

- [x] curl / HTTP request / HTTP response snippets (Markdown)
- [x] Field tables with bidirectional top-level validation and optional fields
- [ ] Nested/`a.b.c` JSON paths and arrays
- [ ] AsciiDoc output and request-parameter / path-parameter snippets
