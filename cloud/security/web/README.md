# cloud/security/web — GoSpring Security filter chain

The GoSpring port of Spring Security's **HttpSecurity / SecurityFilterChain**: a
per-request `net/http` filter that authenticates the caller, publishes the
[`security.Authentication`](..) on the request context, and authorises the
request against ordered path rules.

It is the cohesive HTTP-security module that ties together the primitives
already in [`cloud/security`](..) — `Authentication`, the SecurityContext
accessors (`WithAuthentication`/`FromContext`), `TokenValidator`, bearer parsing
— into one Spring-Security-shaped programming model, instead of leaving them as
loose parts.

## Mapping to Spring Security

| Spring Security | `cloud/security/web` |
|---|---|
| `SecurityFilterChain` / `HttpSecurity` | `FilterChain` + `Then(handler)` |
| `authorizeHttpRequests(...)` | `Authorize(rules...)` |
| `requestMatchers("/x/**")` | `Matcher("/x/**")` |
| `anyRequest()` | `AnyRequest()` |
| `permitAll` / `denyAll` / `authenticated` | `PermitAll` / `DenyAll` / `Authenticated` |
| `hasRole` / `hasAuthority` | `HasRole` (adds `ROLE_`) / `HasAuthority` |
| OAuth2 resource-server (bearer) | `BearerAuthenticator(TokenValidator)` |
| `AuthenticationEntryPoint` / `AccessDeniedHandler` | `OnDenied` (401 vs 403) |

## Example

```go
chain := web.NewFilterChain().
    Authentication(web.BearerAuthenticator(jwtValidator)).
    Authorize(
        web.Matcher("/public/**").PermitAll(),
        web.Matcher("/admin/**").HasRole("ADMIN"),
        web.Matcher("/api/**").Authenticated(),
        web.AnyRequest().DenyAll(),
    )
http.ListenAndServe(":8080", chain.Then(appMux))
```

Inside a handler, the identity is on the context exactly as elsewhere in the
ecosystem:

```go
auth, _ := security.FromContext(r.Context())
```

## Semantics

- **Ordered, first-match rules.** The first rule whose ant pattern matches the
  path decides; put specific rules first and `AnyRequest()` last.
- **Secure by default.** A request matching no rule is denied.
- **401 vs 403.** Anonymous denial → 401; an authenticated caller lacking the
  authority → 403. A present-but-invalid credential → 401. Override with
  `OnDenied`.
- **One identity model.** The chain stores the same `security.Authentication`
  the RPC-side `Require` guard reads, so HTTP and non-HTTP paths agree.

## Status

- [x] Filter chain, ordered ant-matched rules, role/authority checks, 401/403
- [x] Bearer/OAuth2 resource-server authenticator over `TokenValidator`
- [ ] CSRF enforcement filter (tokens already in `cloud/security`)
- [ ] Form-login / session authentication filter
