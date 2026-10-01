# cloud/ws — GoSpring Web Services (SOAP)

The GoSpring port of **Spring Web Services**: a contract-first SOAP endpoint
model. Requests are routed by the local name of their payload root element (the
`@PayloadRoot` analog) to a typed handler; the SOAP 1.1 envelope and Fault
handling are provided by the package. Marshalling is plain `encoding/xml`, so
request/response types are ordinary structs with xml tags.

## Mapping to Spring WS

| Spring Web Services | `cloud/ws` |
|---|---|
| `@Endpoint` + `@PayloadRoot(localPart=...)` | `Register(d, rootElement, handler)` |
| marshalling endpoint adapter | request/response structs via `encoding/xml` |
| `MessageDispatcherServlet` | `Dispatcher.Server()` (net/http) |
| `SoapFault` | `Fault{Code, String}` |

## Example

```go
type GetUserRequest  struct { XMLName xml.Name `xml:"getUserRequest"`;  ID int `xml:"id"` }
type GetUserResponse struct { XMLName xml.Name `xml:"getUserResponse"`; Name string `xml:"name"` }

d := ws.NewDispatcher()
ws.Register(d, "getUserRequest", func(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
    u, ok, err := repo.FindById(ctx, int64(req.ID))
    if err != nil { return GetUserResponse{}, err }
    if !ok { return GetUserResponse{}, ws.Fault{Code: "soap:Client", String: "no such user"} }
    return GetUserResponse{Name: u.Name}, nil
})
http.Handle("/ws", d.Server())
```

A POSTed SOAP envelope whose `Body` contains `<getUserRequest>` is unmarshalled
into `GetUserRequest`, handled, and the returned `GetUserResponse` is wrapped in
a SOAP response envelope. A returned error becomes a SOAP `Fault` with HTTP 500.

## Semantics

- **Dispatch by payload root.** The first element inside `<soap:Body>` selects
  the endpoint, exactly like `@PayloadRoot`.
- **Typed endpoints.** `Register[Req, Resp]` unmarshals into `Req` and marshals
  `Resp`; a parse failure is a `soap:Client` fault.
- **Faults.** Return a `ws.Fault` for a specific code/message, or any error for a
  generic `soap:Server` fault; the server replies HTTP 500 with the fault
  envelope, per the SOAP contract.

## Status

- [x] SOAP 1.1 envelope, payload-root dispatch, typed endpoints, faults, server
- [ ] SOAP 1.2 envelope and WS-Addressing
- [ ] WSDL generation and schema (XSD) validation
