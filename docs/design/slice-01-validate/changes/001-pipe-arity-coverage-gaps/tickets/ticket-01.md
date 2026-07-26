---
id: 01
type: component
slice: slice-01-validate
blocked_by: []
inputs: [docs/design/slice-01-validate/changes/001-pipe-arity-coverage-gaps/contracts.md, docs/design/slice-01-validate/changes/001-pipe-arity-coverage-gaps/change-delta.md, component-tests/features/validate.feature, component-tests/steps/validate_steps.go, component-tests/fixtures/validate/good/config.yaml, api-specification/report.schema.json]
outputs: [component-tests/features/validate.feature, component-tests/steps/validate_steps.go, component-tests/fixtures/validate/incompatible/config.yaml, component-tests/fixtures/validate/incompatible/consumed-contract.yaml, component-tests/fixtures/validate/incompatible/provider-asyncapi.yaml, component-tests/fixtures/validate/good/report.baseline.json]
skills: [component-tests]
---

### TICKET 01 — slice-01-validate/component: scenario 9 (exit-1 verdict) + the I2 byte baseline

**io:** — (test artifact) → skills: `component-tests`

**THIS TICKET RUNS FIRST, ON UNMODIFIED CODE (`blocked_by: []`, HEAD = `9eef9e9`).** The baseline
artifact it captures is only valid if it is taken **before** any signature in `internal/validate/`
is touched. Do **not** edit any file under `internal/validate/` or `cmd/` in this ticket.

**What this ticket does** — two additions to the *existing, shipped* component suite, both from
`contracts.md` §6 (scenario table) and §5 (Gherkin↔module reconciliation). The 8 existing scenarios
and their fixtures stay **byte-untouched**; this is `patch` discipline, not a rewrite.

#### A. New scenario 9 — the exit-1 primary verdict (`contracts.md` §6, row 9)

Scenario name, verbatim, tagged `@wip` (scenarios 1–8 stay untagged):

```gherkin
  @wip
  Scenario: Consumer incompatible with provider (primary verdict)
    Given a config whose consumer uses a channel address absent from the provider spec
    When I run `pinout-asyncapi validate config.yaml`
    Then the exit code is 1
    And stdout is a schema-valid report with compatible=false
    And stdout report errors[0].code == "CHANNEL_NOT_IN_PROVIDER"
```

Assertion order is fixed by `contracts.md` §6 ("Assertion order for scenario 9"): exit **1** →
stdout is a schema-valid canon-`1.1` report → `compatible == false` → `errors[0].code ==
"CHANNEL_NOT_IN_PROVIDER"`.

**Fixture `component-tests/fixtures/validate/incompatible/` — minimal mutation of `good/`:** copy
`good/`'s three files, then add **one** extra channel address (e.g. `WALLET.FEES.EVENTS`) to
**BOTH** `config.yaml`'s `consumer.channels` **AND** `consumed-contract.yaml`'s `channels`, and to
**neither** the provider spec. `provider-asyncapi.yaml` is copied unchanged. Point the new
`config.yaml`'s `consumed_contract_path`/`spec_path` at `/fixtures/validate/incompatible/…` (the
tester mounts `./fixtures` at `/fixtures`, see `docker-compose.test.yml`).

- **Fixture trap (`contracts.md` §NewComparison, repeated because it silently proves the WRONG
  branch):** if the added address is missing from the consumed-contract, `NewComparison`'s
  cross-artifact check fires → `ErrConfigInvalid` → exit **2**, and the scenario passes for the
  wrong reason. It MUST be in both.
- **`component-tests/fixtures/validate/good/`'s three source files MUST NOT be modified** — part B
  below depends on their exact bytes.

#### B. New Then-step on the EXISTING happy scenario — the I2 byte baseline

Scenario 1 today decodes stdout and checks only `compatible == true` / `len(errors) == 0`
(`validate_steps.go`), so indentation, key order, `schema_version`, `validator`, `interaction`,
`provenance` and `generated_at` are invisible to it — too insensitive to prove a behaviour-preserving
refactor. Add **one** Then-step to scenario 1 (no new scenario — the §6 count stays **9**):

```gherkin
    And stdout bytes equal the captured baseline report
```

Step definition semantics (`contracts.md` §5, new I2 row): **full byte equality** between stdout and
`component-tests/fixtures/validate/good/report.baseline.json`, with exactly one surgical
normalization — `generated_at` MUST be **present** and MUST parse as **RFC3339**, and only its
*value* is replaced by the same constant on both sides. Every other byte (indentation, key order,
every constant) must be equal. Do not decode-and-compare structs; compare bytes.

**Capturing the baseline (MUST be from the current, unmodified binary):**

```bash
cd component-tests
docker compose -f docker-compose.test.yml build tool
docker compose -f docker-compose.test.yml up -d tool
docker compose -f docker-compose.test.yml run --rm --no-deps --entrypoint /bin-share/tool \
  tester validate /fixtures/validate/good/config.yaml > fixtures/validate/good/report.baseline.json
docker compose -f docker-compose.test.yml down -v --remove-orphans
```

Sanity-check the captured file before committing it: it must be a canon-`1.1` report with
`"compatible": true`, `"errors": []`, a non-empty `uncovered_channels[]`, and an RFC3339
`generated_at`. If it is not, STOP and report — the capture, not the refactor, is broken.

#### C. Feature-file header

Restate the header formula and the STOP-warning per `contracts.md` §6: `N = 2 (happy-class:
compatible + incompatible verdict) + 7 (adapter branches) = 9`, and the warning now guards against a
**10th** scenario. Record why a report-write failure is deliberately **not** a 10th scenario (it is
not one of the 7 adapter branches — it is pinned in `cmd/app/main_test.go`, ticket-03).

**Dependencies:** none — this is the root of the change's dependency graph.

**Subagent instruction:** capture the baseline first (B) → author the `incompatible/` fixture →
add scenario 9 + the Then-step + their step definitions → run the suite → scenarios 1–8 must stay
GREEN, scenario 9 must be present and `@wip`. Touch nothing under `internal/` or `cmd/`.

**Verify:** `bash component-tests/scripts/run-tests.sh` — 9 scenarios collected; the 8 pre-existing
ones GREEN (including scenario 1 with its new byte-baseline step, which passes against the freshly
captured baseline); scenario 9 present and tagged `@wip`.

**Acceptance:**

- [ ] `component-tests/fixtures/validate/good/report.baseline.json` exists, captured from the
      **unmodified** binary at `9eef9e9`, and `git diff --stat component-tests/fixtures/validate/good/`
      shows **no** change to `config.yaml` / `consumed-contract.yaml` / `provider-asyncapi.yaml`.
- [ ] `component-tests/fixtures/validate/incompatible/` exists (3 files); the extra address is in
      both `consumer.channels` and the consumed-contract's `channels`, and absent from the spec.
- [ ] Scenario 9 exists with the verbatim name and the 4 assertions in order, tagged `@wip`.
- [ ] Scenario 1 carries the byte-baseline Then-step; suite total = 9 scenarios.
- [ ] Scenarios 1–8 still GREEN; **no** existing scenario's assertions were weakened or deleted.
- [ ] `git diff --stat api-specification/` = 0 files changed.

Removing `@wip` and driving scenario 9 GREEN is the **@fagan acceptance step**, NOT this ticket.
