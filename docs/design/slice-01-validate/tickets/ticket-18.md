---
id: 18
type: module
slice: slice-01-validate
blocked_by: [01, 02, 03, 15, 16, 17]
inputs: [docs/design/slice-01-validate/contracts.md, docs/design/slice-01-validate/module-tree.md, sandbox/EMULATION.md]
outputs: [internal/validate/logic.go, internal/validate/comparecontracts_test.go]
io: none
skills: []
---

### TICKET 18 — slice-01-validate/CompareContracts: fold R1–R9 over every channel (never short-circuits)

**io:** none → skills: [] (pure fold logic — **unit-tested by formula**). **The loop lives here**,
not in the head.

**Context (only this module — APPEND to `internal/validate/logic.go`):**
- contract (`contracts.md` §CompareContracts): `CompareContracts(c: Comparison) -> Outcome`. For
  every channel in `consumer.channels` (**sorted order — determinism**), builds a
  `ChannelComparison` (ticket 16's helper type) and calls `ResolveChannelDirection` (ticket 16);
  for each resolved message builds a `MessageComparison` (ticket 17's helper type) and calls
  `CompareMessage` (ticket 17); **accumulates every violation without short-circuiting across
  channels** (several may surface in one run — EMULATION.md S2 is the regression anchor: a single
  shared-server protocol mutation must yield **one violation per affected channel**, not one
  collapsed record). Also computes `uncovered_channels[]` (provider channels outside
  `consumer.channels` — informational, no verdict/exit effect) and carries forward `ConsumerName`/
  `Provenance` for `FoldReport` (ticket 19). Deps = —. Never returns an `Error` — `incompatible`
  is a legitimate `Outcome` value (use-case.md, note after Extensions 6a–6i).
  - antecedent: a valid `Comparison` (ticket 15).
  - consequent: `Outcome{Violations, UncoveredChannels, ConsumerName, Provenance}`.
- **unit tests: 3** (`contracts.md` §4 formula) — 1 happy + 2 branches: violations accumulate
  across ≥2 channels without short-circuit (S2 regression anchor); `uncovered_channels`
  populated without affecting `compatible` (S0 regression anchor — EMULATION.md §1).
- component scenario(s) to green: none directly (verdict codes are UNIT-covered here per
  `contracts.md` §6); its ∅-violations path underpins component scenario 1 (happy), greened later
  by @fagan.

**Dependencies:** `ResolveChannelDirection` (ticket 16), `CompareMessage` (ticket 17);
`Comparison`/`Outcome`/`Violation` types (ticket 03).

**Subagent instruction:** write the 3 unit tests in
`internal/validate/comparecontracts_test.go` → implement `CompareContracts` in
`internal/validate/logic.go` (append) → run units → green → done. Touch no other module.

**Verify:** `go build ./internal/validate/... && go vet ./internal/validate/... && go test ./internal/validate/... -run TestCompareContracts`.

**Acceptance:** package `validate` builds/vets clean; 3 unit tests green.
