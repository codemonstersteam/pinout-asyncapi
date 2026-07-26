---
id: 05
type: module
slice: slice-01-validate
blocked_by: [02, 04]
inputs: [docs/design/slice-01-validate/changes/001-pipe-arity-coverage-gaps/contracts.md, docs/design/slice-01-validate/changes/001-pipe-arity-coverage-gaps/module-tree.md, internal/validate/logic.go, internal/validate/domain.go, internal/validate/head.go, internal/validate/newcomparison_test.go]
outputs: [internal/validate/domain.go, internal/validate/logic.go, internal/validate/head.go, internal/validate/newcomparison_test.go]
io: none
skills: []
---

### TICKET 05 — slice-01-validate/NewComparison: materialize the three-stream join as `ComparisonInput`

**io:** none → skills: [] (pure logic; edit-in-place reshape of an existing, shipped node)

**Nature: the join is NAMED, not removed (`patch`).** `NewComparison(cfg, contract, pchans)`
(`internal/validate/logic.go:576`) is the one point in the slice where three flows **genuinely**
unite, so its arity is not an accident to be deleted — it is materialized as a type so the node takes
**one** argument and the join becomes visible in the type system (`module-tree.md` §3, "One data
argument"). Tree node **14**.

**New type — `ComparisonInput` (`internal/validate/domain.go`, `contracts.md` §1):**

```go
// A plain transport DTO: exported fields, no validity claim of its own — validity is
// established by NewComparison, the one node it feeds.
type ComparisonInput struct {
	Config           Config
	Contract         ConsumedContract
	ProviderChannels ProviderChannels
}
```

**Counter-rule it must NOT become (anti-pattern gate, BRD R4 / `module-tree.md` §3):** it is **not** a
shared `Context`/`State`. It is assembled once, consumed by **exactly one** node, and **never threaded
further down the pipe** — `CompareContracts` keeps receiving the validated `Comparison`, not the
input. It carries **three** fields and **no** ports, no clock, no config bag. A fourth unrelated
field, or a use surviving past `NewComparison`, would be the violation.

**Contract (`contracts.md` §NewComparison):**

- Signature: `NewComparison(in: ComparisonInput) -> Result<Comparison, Error>`
  (Go: `func NewComparison(in ComparisonInput) (Comparison, error)`)
- Input: **one** data entity. Deps: —. `io: none`.
- **Antecedent, unchanged verbatim:** every address in `in.Config.Consumer.Channels` is present among
  `in.Contract.Channels[].Address`.
- **Consequent, unchanged verbatim:** success → `Comparison` (valid by construction); failure →
  `ErrConfigInvalid` (`CONFIG_ERROR`) → exit **2**, still before any report is written.

**Call-site rewiring (same ticket — the tree must compile and stay green):** in
`internal/validate/head.go`:

```go
comparison, err := NewComparison(ComparisonInput{
	Config:           cfg,
	Contract:         contract,
	ProviderChannels: pchannels,
})
```

Composite-literal note: `ComparisonInput` is deliberately outside the "valid by construction, no naked
literal" set (`module-tree.md` §3) — this literal in the head is the sanctioned form.

**Unit tests (`contracts.md` §4, `NewComparison` N = 2 — unchanged):**
`internal/validate/newcomparison_test.go` — **mechanical call-site updates only**
(`NewComparison(cfg, contract, pchans)` → `NewComparison(ComparisonInput{…})`). **0 assertions
deleted or weakened** (BRD N5); the happy case and the "config address absent from the contract →
`ErrConfigInvalid`" branch stay exactly as they are.

**Dependencies:** ticket-02 (both byte anchors captured before any signature is touched — MUST hold),
ticket-04 (serializes the `logic.go`/`head.go` edits; carries the ticket-01 component edge
transitively).

**Subagent instruction:** add `ComparisonInput` to `domain.go` → change `NewComparison`'s signature in
`logic.go` (body logic unchanged, only field access `cfg` → `in.Config` etc.) → rewire the one call
site in `head.go` → update `newcomparison_test.go` mechanically → run the package suite + the anchors
→ green → done. Touch no other module.

**Verify:** `go build ./... && go vet ./... && go test ./internal/validate/... && go test ./cmd/...`

**Acceptance:**

- [ ] `NewComparison` takes exactly **one** argument; `ComparisonInput` lives in `domain.go` with
      exactly the three fields above.
- [ ] `ComparisonInput` appears in exactly two places: its declaration and the head's literal — it is
      never passed to `CompareContracts` or beyond.
- [ ] `newcomparison_test.go`: same cases, same assertions, call sites updated only.
- [ ] The channel-scope cross-check still yields `ErrConfigInvalid` → exit 2 (component scenario 2
      stays GREEN).
- [ ] **Regression envelope holds (E1):** `go test ./internal/validate/... ./cmd/...` GREEN,
      including ticket-02's determinism anchor **without regenerating the golden**.
