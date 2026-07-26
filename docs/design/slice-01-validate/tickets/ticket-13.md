---
id: 13
type: module
slice: slice-01-validate
blocked_by: [01, 02, 03, 11, 12]
inputs: [docs/design/slice-01-validate/contracts.md, docs/design/slice-01-validate/module-tree.md, sandbox/EMULATION.md]
outputs: [internal/validate/logic.go, internal/validate/buildspecloader_test.go]
io: none
skills: []
---

### TICKET 13 — slice-01-validate/BuildSpecLoader: late-bound loader-strategy factory

**io:** none → skills: [] (factory; performs no I/O itself — **unit-tested by formula**).

**Context (only this module — APPEND to `internal/validate/logic.go`):**
- contract (`contracts.md` §BuildSpecLoader): `BuildSpecLoader(provider: ProviderConfig, timeout
  time.Duration, httpClient *http.Client) -> SpecLoader`. Selects and constructs **exactly one**
  loader implementation — `FileSpecLoader` (ticket 11) when `spec_path` set, `HTTPSpecLoader`
  (ticket 12) when `spec_url` set. **Must be called by the head AFTER `NewConfig`**, so `timeout`
  is always the real, already-validated `cfg.Settings.Timeout` — **never a hardcoded constant**
  (EMULATION.md S15 guards exactly this: a loader built early with a constant timeout makes the
  `TIMEOUT_ERROR` scenario unreachable). Deps = `timeout time.Duration` (= `cfg.Settings.Timeout`,
  a config value, not I/O); `HTTPClient *http.Client` (only used if the HTTP branch is selected —
  encapsulated into the returned `HTTPSpecLoader`, never exposed to the head).
  - antecedent: exactly one of `spec_path`/`spec_url` set (guaranteed by `NewConfig`'s
    antecedent, ticket 07).
  - consequent: a `SpecLoader` (interface). No failure branch (construction only).
- **unit tests: 2** (`contracts.md` §4 formula) — 1 happy + 1 branch: selects `FileSpecLoader` vs
  `HTTPSpecLoader` (1 branch, both outcomes distinguishable by concrete type).
- component scenario(s) to green: none directly; it is the mechanism that makes scenario 8
  (`TIMEOUT_ERROR`) reachable once wired late by the head (ticket 22).

**Dependencies:** `ProviderConfig` type (ticket 03); `FileSpecLoader`/`NewFileSpecLoader` (ticket
11); `HTTPSpecLoader`/`NewHTTPSpecLoader` (ticket 12).

**Subagent instruction:** write the 2 unit tests in
`internal/validate/buildspecloader_test.go` → implement `BuildSpecLoader` in
`internal/validate/logic.go` (append) → run units → green → done. Touch no other module.

**Verify:** `go build ./internal/validate/... && go vet ./internal/validate/... && go test ./internal/validate/... -run TestBuildSpecLoader`.

**Acceptance:** package `validate` builds/vets clean; 2 unit tests green.
