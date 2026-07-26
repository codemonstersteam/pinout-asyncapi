---
id: 17
type: module
slice: slice-01-validate
blocked_by: [01, 02, 03, 15]
inputs: [docs/design/slice-01-validate/contracts.md, docs/design/slice-01-validate/module-tree.md, sandbox/EMULATION.md]
outputs: [internal/validate/logic.go, internal/validate/comparemessage_test.go]
io: none
skills: []
---

### TICKET 17 — slice-01-validate/CompareMessage: R5–R9 message-level variance/type/metadata rules

**io:** none → skills: [] (pure logic — **unit-tested by formula**; the *sole* unit boundary for
R5–R9, per `contracts.md` §6's "verdict is a unit, not a component scenario" rule).

**Context (only this module — APPEND to `internal/validate/logic.go`):**
- contract (`contracts.md` §CompareMessage): `CompareMessage(m: MessageComparison) ->
  []Violation`. `MessageComparison` is a **private, unexported helper struct** (not in the frozen
  message catalog) — one message pair matched by the `channel.messages` map key (D8), with its
  direction's variance rule; construct it next to `ResolveChannelDirection` (ticket 16) in
  `logic.go` (`CompareContracts`, ticket 18, builds one per resolved message). Deps = —.
  Over **both** `payload` and `headers`:
  - **R5** (`sends`, contravariant) a field the provider `required`s is not in
    `fields(consumer.sends)` → `CodeMissingRequiredSentField`.
  - **R6** (`receives`, covariant) a field the consumer reads is absent from
    `properties(provider)` → `CodeReadsFieldNotProvided` (catches provider field removal).
  - **R7** a shared field's `type`/`format` diverges, recursively (nested object / array `items` /
    resolved `$ref`) → `CodeTypeMismatch`.
  - **R8** effective `contentType` diverges, **only when** `message.content_type` is declared in
    the contract → `CodeContentTypeMismatch` (absent ⇒ not checked).
  - **R9** `correlationId.location` diverges, **only when** `message.correlation_id_location` is
    declared in the contract, **not gated by communication pattern** (D9) → `CodeCorrelationIDMismatch`.
  - antecedent: a resolved `MessageComparison` (message key matched on both sides, via
    `ResolveChannelDirection`, ticket 16).
  - consequent: zero or more of `{CodeMissingRequiredSentField, CodeReadsFieldNotProvided,
    CodeTypeMismatch, CodeContentTypeMismatch, CodeCorrelationIDMismatch}`.
- **unit tests: 10** (`contracts.md` §4 formula) — 1 happy + 9 branches: R5 missing required sent
  field; R6 read field not provided; R7 type mismatch; R7 format mismatch; R7 recursion (nested
  object / array `items` / resolved `$ref`); R8 content-type mismatch (declared); R8 not checked
  (undeclared); R9 correlationId mismatch (declared); R9 not checked (undeclared). Use
  EMULATION.md §2 "Уровень сообщения" (S4–S8) as reference fixtures — S5 (ticket 03's `SchemaNode`
  has no `additionalProperties` guard, so covariance is the ONLY thing that catches removal) and
  S8 (R9's undeclared-side must NOT fire) are the two anchor cases.
- component scenario(s) to green: none (R5–R9 are unit boundaries, not component scenarios,
  `contracts.md` §6).

**Dependencies:** `Message`/`SchemaNode`/`Violation` types (ticket 03); `ResolveChannelDirection`
(ticket 16, shares the `MessageComparison` shape).

**Subagent instruction:** write the 10 unit tests in
`internal/validate/comparemessage_test.go` → implement `CompareMessage` (+ the private
`MessageComparison` helper type) in `internal/validate/logic.go` (append) → run units → green →
done. Touch no other module.

**Verify:** `go build ./internal/validate/... && go vet ./internal/validate/... && go test ./internal/validate/... -run TestCompareMessage`.

**Acceptance:** package `validate` builds/vets clean; 10 unit tests green.
