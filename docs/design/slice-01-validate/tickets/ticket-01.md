---
id: 01
type: scaffold
slice: slice-01-validate
blocked_by: []
inputs: [harness/target-profiles.json, harness/scaffold.sh, api-specification/config.schema.json, api-specification/report.schema.json, README.md]
outputs: [go.mod, cmd/app/main.go, internal/shared/config/config.go, internal/shared/report/writer.go, config.yaml, component-tests/docker-compose.test.yml, component-tests/scripts/run-tests.sh]
skills: [service-scaffold]
---

### TICKET 01 — slice-01-validate/scaffold: clone template-go-cli → runnable placeholder

**Target profile:** `cli` (`harness/target-profiles.json`, profile `cli`) → template
**`template-go-cli`**. Ingress = `cli-io`; component-test flavour = one-shot-binary.

**What this ticket does (mechanical — no logic):** run `harness/scaffold.sh pinout-asyncapi` — it
`git archive`-clones the tracked content of `template-go-cli`, renames the go-module to
`pinout-asyncapi`, preserves the already-frozen `api-specification/` and the design `README.md`,
then `go build ./...`. Then run the template's two verification scripts (build/test + smoke).
Green → done.

**Deterministic outputs (exactly what `scaffold.sh` produces — NOT invented, NOT slice-named):**
- `go.mod` (module renamed `pinout-asyncapi`), `go.sum`
- `cmd/app/main.go` — generic ingress placeholder (`run <config>` → head → `NOT_IMPLEMENTED`/exit 3).
  **Stays `cmd/app/` — there is NO `cmd/<slug>/` rename** (slice identity lives in
  `internal/validate/`).
- template placeholder packages under `internal/` (the generic `example` package +
  `internal/shared/`) — shipped as-is by the template; the `internal/validate/*.go` module
  tickets add the real slice package later. Do NOT reshape placeholders here (they are the
  template's, not slice code).
- `config.yaml` — placeholder config.
- `component-tests/…` boilerplate: `docker-compose.test.yml`, `scripts/run-tests.sh`,
  `Dockerfile.runtime`, `compose/tool.Dockerfile`, `steps/…`, `features/smoke.feature`.

**MUST NOT:** reshape code for the slice, write tests, rename `cmd/app`, overwrite
`api-specification/` or `README.md` (scaffold.sh preserves both), invent the `lerenn/asyncapi-codegen`
wiring here (that is the module tickets).

**Dependencies:** none (`blocked_by: []`) — this ticket blocks all others.

**Verify:** `harness/scaffold.sh` exits 0 (`go build ./...` green) + smoke script green.

**Acceptance:** template cloned, go-module = `pinout-asyncapi`, `go build ./...` green, placeholder
binary runs (`version` prints, `run` → not-implemented/exit 3), frozen contract + README preserved.
