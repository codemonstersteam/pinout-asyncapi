---
id: 14
type: module
slice: slice-01-validate
blocked_by: [01, 02, 03]
inputs: [docs/design/slice-01-validate/contracts.md, docs/design/slice-01-validate/module-tree.md, sandbox/EMULATION.md]
outputs: [internal/validate/logic.go, internal/validate/deriveproviderchannels_test.go]
io: none
skills: []
---

### TICKET 14 — slice-01-validate/DeriveProviderChannels: project provider spec into channel map

**io:** none → skills: [] (pure projection — **unit-tested by formula**).

**Context (only this module — APPEND to `internal/validate/logic.go`):**
- contract (`contracts.md` §DeriveProviderChannels): `DeriveProviderChannels(spec: ProviderSpec)
  -> ProviderChannels`. Projects the parsed spec into `address -> {protocol, send: [msg-key],
  receive: [msg-key]}`, expanding two AsyncAPI-3.0 defaults: an operation with no `messages`
  means **all** of the channel's messages; `reply` injects the opposite direction onto the reply
  channel (D3). Message identity = the `channel.messages` **map key** (D8 — the parser overwrites
  `Message.Name` with a generated Go identifier; EMULATION.md §4 point 3). Deps = —.
  - antecedent: a `ProviderSpec` produced by a successful `Process()` (typed, `#/servers/*` refs
    already inlined).
  - consequent: `ProviderChannels`, total over any valid `ProviderSpec` — no failure branch.
- **unit tests: 3** (`contracts.md` §4 formula) — 1 happy + 2 branches: operation with no
  `messages` → expands to all channel messages; operation with `reply` → injects the opposite
  direction on the reply channel. Use EMULATION.md §0 (the base projection table) as the
  reference fixture shape.
- component scenario(s) to green: none directly (pure logic feeding scenario 1's happy path,
  greened later by @fagan).

**Dependencies:** `ProviderSpec`/`ProviderChannels` types (ticket 03).

**Subagent instruction:** write the 3 unit tests in
`internal/validate/deriveproviderchannels_test.go` → implement `DeriveProviderChannels` in
`internal/validate/logic.go` (append) → run units → green → done. Touch no other module.

**Verify:** `go build ./internal/validate/... && go vet ./internal/validate/... && go test ./internal/validate/... -run TestDeriveProviderChannels`.

**Acceptance:** package `validate` builds/vets clean; 3 unit tests green.
