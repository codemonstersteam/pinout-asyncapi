---
id: 16
type: module
slice: slice-01-validate
blocked_by: [01, 02, 03, 15]
inputs: [docs/design/slice-01-validate/contracts.md, docs/design/slice-01-validate/module-tree.md, sandbox/EMULATION.md]
outputs: [internal/validate/logic.go, internal/validate/resolvechanneldirection_test.go]
io: none
skills: []
---

### TICKET 16 — slice-01-validate/ResolveChannelDirection: R1–R4 channel/direction resolution

**io:** none → skills: [] (pure logic — **unit-tested by formula**; the *sole* unit boundary for
R1–R4, per `contracts.md` §6's "verdict is a unit, not a component scenario" rule).

**Context (only this module — APPEND to `internal/validate/logic.go`):**
- contract (`contracts.md` §ResolveChannelDirection): `ResolveChannelDirection(ch:
  ChannelComparison) -> (resolved ResolvedDirection, violations []Violation)`.
  `ChannelComparison` is a **private, unexported helper struct** (not in the frozen message
  catalog, `contracts.md` §1) — one consumer channel + its provider-projected counterpart from
  `ProviderChannels`, per direction; construct it as a small internal type in `logic.go` next to
  this function (`CompareContracts`, ticket 18, builds one per channel when it folds). Deps = —.
  - **R1** channel absent from the provider projection → `CodeChannelNotInProvider`.
  - **R2** `channel.protocol` (consumer) ≠ the provider channel's protocol → `CodeProtocolMismatch`.
  - **R3** no counterpart operation for the consumer's direction on the channel (`reply` already
    expanded by `DeriveProviderChannels`, D3) → `CodeDirectionNotInProvider`.
  - **R4** direction exists but the message key is absent from the provider's messages on that
    direction → `CodeMessageNotInProvider`.
  - antecedent: a `ChannelComparison` for one address. **Hierarchical stop**: no channel ⇒ no
    direction check; no direction ⇒ no message check (EMULATION.md §3) — do not cascade
    derivative violations.
  - consequent: the resolved message set to hand to `CompareMessage` (ticket 17), plus zero or
    more of `{CodeChannelNotInProvider, CodeProtocolMismatch, CodeDirectionNotInProvider,
    CodeMessageNotInProvider}`.
- **unit tests: 5** (`contracts.md` §4 formula) — 1 happy + 4 branches: R1 channel absent; R2
  protocol mismatch; R3 direction absent; R4 message key absent. Use EMULATION.md §2 "Уровень
  канала" (S1–S3b) as reference fixtures — **S2 is the regression anchor for fold, not
  short-circuit** (a single protocol mutation on a shared server must yield one violation **per
  affected channel**, not a single collapsed one) — but that accumulation itself is
  `CompareContracts`'s job (ticket 18); this module's unit only proves R2 fires per call.
- component scenario(s) to green: none (R1–R4 are unit boundaries, not component scenarios,
  `contracts.md` §6).

**Dependencies:** `Comparison`/`ProviderChannels`/`Violation` types (ticket 03); `NewComparison`
(ticket 15, for realistic `Comparison` fixtures in tests).

**Subagent instruction:** write the 5 unit tests in
`internal/validate/resolvechanneldirection_test.go` → implement `ResolveChannelDirection` (+ the
private `ChannelComparison`/`ResolvedDirection` helper types) in `internal/validate/logic.go`
(append) → run units → green → done. Touch no other module.

**Verify:** `go build ./internal/validate/... && go vet ./internal/validate/... && go test ./internal/validate/... -run TestResolveChannelDirection`.

**Acceptance:** package `validate` builds/vets clean; 5 unit tests green.
