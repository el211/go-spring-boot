# gs-modulith — GoSpring Modulith verifier

The build-time counterpart to Spring Modulith's `ApplicationModules.verify()`.
It loads a declared module model (`modulith.json`) and the application's real
import graph (via `go list -json ./...`), then fails if any package crosses a
module boundary it may not — a disallowed dependency, or reaching into another
module's internal packages.

## Run

```bash
go install go-spring.org/gs-modulith@latest
gs-modulith -config modulith.json -dir .
```

Exit code is non-zero when violations are found, so it drops straight into CI or
a `go test` wrapper.

## Flags

| Flag | Default | Meaning |
|---|---|---|
| `-config` | `modulith.json` | module declaration file |
| `-dir` | `.` | module root to analyse (`go list ./...` runs here) |
| `-v` / `-vv` / `--verbose=N` | 0 | log level (per `gs/CLAUDE.md`): step lines; `-v` adds argv; `-vv` adds dir + per-module detail |

## modulith.json

```json
{
  "modules": [
    {
      "name": "order",
      "basePackage": "github.com/acme/shop/order",
      "allowedDependencies": ["catalog", "shared"]
    },
    {
      "name": "catalog",
      "basePackage": "github.com/acme/shop/catalog",
      "exposed": ["github.com/acme/shop/catalog/api"]
    },
    { "name": "shared", "basePackage": "github.com/acme/shop/shared" }
  ]
}
```

- `allowedDependencies` — whitelisted module names. **Empty/omitted = open** (may
  depend on any module); a non-empty list is a strict whitelist.
- `exposed` — API packages beyond the base package that other modules may import.
  Everything deeper than the base (and not exposed) is internal.

The rule engine lives in [`cloud/modulith`](../../cloud/modulith); this tool only
loads inputs and reports results.
