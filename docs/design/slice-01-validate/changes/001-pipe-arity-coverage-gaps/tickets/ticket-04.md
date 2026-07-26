---
id: 04
type: module
slice: slice-01-validate
blocked_by: [01, 02, 03]
inputs: [docs/design/slice-01-validate/changes/001-pipe-arity-coverage-gaps/contracts.md, docs/design/slice-01-validate/changes/001-pipe-arity-coverage-gaps/module-tree.md, docs/design/slice-01-validate/changes/001-pipe-arity-coverage-gaps/adr/001-factory-and-product-method-are-two-nodes.md, internal/validate/logic.go, internal/validate/head.go, internal/validate/newconsumedcontract_test.go]
outputs: [internal/validate/logic.go, internal/validate/head.go, internal/validate/newconsumedcontract_test.go, internal/validate/buildcontractparser_test.go]
io: none
skills: []
---

### TICKET 04 — slice-01-validate/BuildContractParser + ContractParser.Parse: bind the consumer name

**io:** none → skills: [] (pure logic; edit-in-place reshape of an existing, shipped node)

**Nature: reshape, behaviour-preserving (`patch`).** `NewConsumedContract(raw, expectedConsumer)`
(`internal/validate/logic.go:120`) takes **two** data arguments. `expectedConsumer` is an
already-`NewConfig`-validated **config scalar**, not a data entity flowing through the pipe — it moves
to construction. Tree nodes **7 + 8** (`module-tree.md` §3); one reshape, one ticket: the factory and
its product method cannot compile apart (ADR-001 splits them as *design* nodes, not as tickets).

**Contract (`contracts.md` §BuildContractParser + §ContractParser.Parse):**

- Factory — Signature: `BuildContractParser(consumerName: string) -> ContractParser`
  - Input: `consumerName` (= `cfg.consumer.Name`). Deps: —. `io: none`.
  - Antecedent: `consumerName` non-empty (guaranteed by `NewConfig`).
  - Consequent: a `ContractParser` carrying that name in an **unexported** field. **No failure
    branch** — this totality is precisely why hoisting the bind ahead of the chain is
    behaviour-preserving. Performs no I/O; it is a collaborator, not an I/O object.
- Product method — `(p ContractParser) Parse(raw RawContract) (ConsumedContract, error)`
  - Input: `RawContract` — **one** data entity. Deps: — (the name is captured in the receiver).
  - **Antecedent, unchanged verbatim:** decodes as valid YAML/JSON; `schema_version == "1.0"`;
    `consumer == p.expectedConsumer`; `provenance.captured_hash` matches
    `^sha256:[0-9a-f]{64}$`; `channels` non-empty; each channel has non-empty `address`/`protocol`
    and at least one of `sends`/`receives`.
  - **Consequent, unchanged verbatim:** success → `ConsumedContract`; any antecedent clause fails →
    `ErrContractInvalid` → `PARSE_ERROR` → exit **3**. Still the **only** way a `ConsumedContract`
    comes to exist (subtype, not guard).

Both live in `internal/validate/logic.go`, next to each other, with the `ContractParser` struct
beside its factory (`module-tree.md` §6 — **no file is added, removed or renamed**).

**Call-site rewiring (same ticket — the tree must compile and stay green):** in
`internal/validate/head.go`, replace `NewConsumedContract(rawContract, cfg.consumer.Name)` with a
pre-chain bind + a one-argument chain step:

```go
parser := BuildContractParser(cfg.consumer.Name)   // pre-chain bind block, after NewConfig
…
contract, err := parser.Parse(rawContract)         // chain: one data entity
```

Place the bind in the block right after `NewConfig` (`module-tree.md` §4). Ticket-07 normalizes the
final shape of that block; do not move `BuildSpecLoader`/`BuildReportWriter` here.

**Unit tests (`contracts.md` §4):**

- `internal/validate/newconsumedcontract_test.go` — **mechanical call-site updates only**:
  `NewConsumedContract(raw, name)` → `BuildContractParser(name).Parse(raw)`. **0 assertions deleted
  or weakened** (BRD N5); the 8 cases (1 happy + 7 branches) stay exactly as they are.
  **C0 does not apply to these calls.** The `.Parse(` here is the **domain** method
  `ContractParser.Parse(RawContract)` (core logic, `contracts.md` §4 rows 7–8) — not the CLI ingress
  adapter's package-level `cli.Parse` (Go: unqualified `validate.Parse(` in
  `internal/validate/adapter.go`), which is what C0 forbids unit-testing (`PLAN.md` §2.10,
  `contracts.md` §`cli.Parse`). Do **not** delete or weaken these cases to satisfy a bare `Parse(`
  grep — the qualified criterion excludes receiver-qualified `.Parse(` by construction.
- `internal/validate/buildcontractparser_test.go` — **new**, **N = 1** (happy only; the factory binds
  an already-validated scalar and has no antecedent branch). Follow the shape of the shipped
  `buildspecloader_test.go` / `buildreportwriter_test.go`.

**Dependencies:** ticket-01 (component RED-first edge), ticket-02 (both byte anchors captured before
any signature is touched — MUST hold), ticket-03 (serializes the `logic.go`/`head.go` edits and lands
the binary-level pins first).

**Subagent instruction:** split the node in `logic.go` → rewire the one call site in `head.go` →
update `newconsumedcontract_test.go` mechanically → add `buildcontractparser_test.go` → run the whole
package suite + the anchors → green → done. Touch no other module.

**Verify:** `go build ./... && go vet ./... && go test ./internal/validate/... && go test ./cmd/...`

**Acceptance:**

- [ ] `NewConsumedContract` no longer exists; `BuildContractParser` + `ContractParser.Parse` do, with
      the antecedent/consequent above unchanged verbatim.
- [ ] `parser.Parse(rawContract)` in `head.go` takes exactly **one** data argument.
- [ ] `newconsumedcontract_test.go`: same number of cases, same assertions, call sites updated only.
- [ ] `buildcontractparser_test.go` exists with 1 test.
- [ ] **Regression envelope holds (E1):** `go test ./internal/validate/... ./cmd/...` GREEN,
      including ticket-02's determinism anchor **without regenerating the golden**.
- [ ] `git diff --stat api-specification/` = 0 files changed.
