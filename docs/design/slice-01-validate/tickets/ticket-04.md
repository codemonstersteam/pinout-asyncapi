---
id: 04
type: module
slice: slice-01-validate
blocked_by: [01, 02, 03]
inputs: [docs/design/slice-01-validate/contracts.md, docs/design/slice-01-validate/module-tree.md]
outputs: [internal/validate/adapter.go]
io: none
skills: []
---

### TICKET 04 — slice-01-validate/cli.Parse: ingress door (argv → Invocation)

**io:** none → skills: [] (ingress adapter — design per the **`cli-io`** discipline, cobra; the
driving adapter of the hexagon is `io: none`, never `http-io`). **Not unit-tested** (parse-map;
asserted by component scenarios).

**Context (only this module):**
- contract (`contracts.md` §cli.Parse): `Parse(args []string) -> Result[Invocation, Error]`.
  Input = process argv. Deps = —. The one positional arg (`<config.yaml>`) is the sole domain
  parameter; no flags beyond `--help`/`--version` (boilerplate).
  - antecedent: exactly one non-flag argument present (else usage error via cobra's own
    `--help`/usage path, not a domain `Error`).
  - consequent: Ok → `Invocation{ConfigPath}`. No domain failure branch (parsing only).
- unit tests: **none** (ingress parse-map).
- component scenario(s) to green: none directly (a well-formed invocation underlies every
  scenario) — greened later.

**Dependencies:** `Invocation` type (ticket 03). cobra (from scaffold).

**Subagent instruction:** implement `cli.Parse` in `internal/validate/adapter.go` (new file) →
`go build`/`vet` → done. Touch no other module.

**Verify:** `go build ./internal/validate/... && go vet ./internal/validate/...`.

**Acceptance:** package `validate` builds and vets clean with `Parse` present; no units. Component
GREEN is @fagan's step.
