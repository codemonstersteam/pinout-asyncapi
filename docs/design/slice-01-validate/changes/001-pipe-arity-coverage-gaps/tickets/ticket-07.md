---
id: 07
type: module
slice: slice-01-validate
blocked_by: [06]
inputs: [docs/design/slice-01-validate/changes/001-pipe-arity-coverage-gaps/module-tree.md, docs/design/slice-01-validate/changes/001-pipe-arity-coverage-gaps/contracts.md, docs/design/slice-01-validate/changes/001-pipe-arity-coverage-gaps/c4.md, internal/validate/head.go]
outputs: [internal/validate/head.go]
io: none
skills: []
---

### TICKET 07 — slice-01-validate/ProcessValidate: head reads as bind-then-chain

**io:** none → skills: [] (rewiring only — **no module logic**, no new behaviour)

**Nature: the last shape edit (`patch`).** Tickets 04–06 each moved their own bind into the head as
they changed their signature; two binds are still **inside** the chain (`loader` after
`ContractStore.Load`, `writer` after the fold, `internal/validate/head.go`). This ticket hoists them
so the head reads as **one** bind block followed by a chain in which **every step takes exactly one
data entity**. Tree node **2**.

**Signature: `ProcessValidate(inv: Invocation, deps: Deps) -> Result<Report, Error>` — UNCHANGED**
(BRD N4). `Deps{Clock, HTTPClient}` — **unchanged** (BRD N4). Ticket-02's determinism anchor depends
on the head being callable identically before and after; changing either is out of lane → STOP.

**Target shape (`module-tree.md` §4, verbatim order):**

```
  # stage 0: the config every bind below depends on
  | ConfigStore.Load(inv.ConfigPath)  -> RawConfig     [CONFIG_ERROR -> 2]
  | NewConfig(raw)                    -> Config        [CONFIG_ERROR -> 2]

  # bind block — not pipe steps: no data flows through them, no I/O, no failure branch (all four total)
  parser   := BuildContractParser(cfg.consumer.Name)
  loader   := BuildSpecLoader(cfg.provider, cfg.settings.Timeout, deps.HTTPClient)
  reporter := BuildReporter(deps.Clock)
  writer   := BuildReportWriter(cfg.settings)

  # chain — every step takes exactly ONE data entity
  | ContractStore.Load(cfg.consumer.ConsumedContractPath) -> RawContract       [FILE_NOT_FOUND -> 3]
  | parser.Parse(raw)                                     -> ConsumedContract  [PARSE_ERROR -> 3]
  | loader.Load()                                         -> ProviderSpec      [FILE_NOT_FOUND|PARSE_ERROR|HTTP_ERROR|TIMEOUT_ERROR -> 3]
  | DeriveProviderChannels(spec)                          -> ProviderChannels
  | NewComparison(ComparisonInput{cfg, contract, pchans}) -> Comparison         [CONFIG_ERROR -> 2]
  | CompareContracts(comparison)                          -> Outcome
  | reporter.Fold(outcome)                                -> Report
  | writer.Write(report)                                  -> Report
  -> Ok(report)
```

**Why hoisting `loader`/`writer` is behaviour-preserving (the load-bearing argument, `module-tree.md`
§4):** all four factories are **total** — no failure branch, no I/O — so moving them ahead of the
chain cannot change *which* error surfaces first, nor the order of the four I/O touchpoints. In
particular `BuildSpecLoader` still runs **after** `NewConfig`, so `cfg.settings.Timeout` is always the
real, already-validated value (EMULATION.md S15 — this is what keeps the `TIMEOUT_ERROR` scenario
reachable). If any of the four ever grows a failure branch, this hoist stops being safe → STOP.

**Also in this ticket:** update `head.go`'s doc comment to describe bind-then-chain (it currently
narrates the late `BuildSpecLoader` call), keeping the ROP short-circuit note and the
"`CompareContracts` never returns an error on the domain axis — `incompatible` is a value, not an
`Err`" note.

**Not in this ticket:** `register.go` (unchanged — `Deps` and the wiring to `cmd/app` are untouched);
`cmd/app/main.go` (unchanged); any module logic. The head is **not** unit-tested (`contracts.md` §4
hard rule) — it is proven by the component suite and by ticket-02's determinism anchor.

**Dependencies:** ticket-06 (and transitively 05/04/03/02/01 — every signature is final before the
head's final shape is set).

**Subagent instruction:** hoist the two remaining binds into the block, restate the doc comment, run
everything → green → done. Change no signature. Do not regenerate
`internal/validate/testdata/report.golden.json`.

**Verify:** `go build ./... && go vet ./... && go test ./... && bash component-tests/scripts/run-tests.sh`

**Acceptance (closes the change's regression envelope — @fagan verifies, this ticket does not
self-sign):**

- [ ] `head.go` holds exactly one bind block of four factories, placed after `NewConfig`; **zero**
      chain steps take two data arguments (`module-tree.md` §3, the sanctioned-exception list is now
      empty).
- [ ] `ProcessValidate`'s signature and `Deps` are byte-identical to `9eef9e9` (BRD N4).
- [ ] `go build ./... && go vet ./...` green; `go test ./...` green — **51** unit tests in
      `internal/validate` (49 pre-existing + `BuildContractParser` + `BuildReporter`), plus the
      determinism anchor and the 5 `cmd/app` cases.
- [ ] Component suite: **9** scenarios, scenarios 1–8 GREEN (scenario 1 including its byte-baseline
      step) and scenario 9 GREEN — exit 1, `compatible == false`,
      `errors[0].code == "CHANNEL_NOT_IN_PROVIDER"`. `@wip` removal is **@fagan's** act, not this
      ticket's.
- [ ] **E1 escalation trigger clear:** none of the 8 pre-existing fixtures yields a different exit
      code, and `good/` yields the same report bytes modulo `generated_at`'s value — i.e.
      `internal/validate/testdata/report.golden.json` and
      `component-tests/fixtures/validate/good/report.baseline.json` are **unmodified** since ticket-01/02
      (`git diff --stat` = 0 files changed for both). A mismatch means the `patch` assumption broke →
      STOP and re-triage, do not adjust the anchors.
- [ ] `git diff --stat api-specification/` = **0 files changed** (N1 — the frozen contracts are
      untouched by the whole change).
- [ ] `grep -rn 'time.Now()' internal/validate/` matches only `register.go` (I1b).
- [ ] `internal/validate/adapter_test.go` does not exist (C0).
