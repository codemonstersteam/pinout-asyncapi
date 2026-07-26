---
id: 02
type: component
slice: slice-01-validate
blocked_by: [01]
inputs: [docs/design/slice-01-validate/contracts.md, docs/design/slice-01-validate/use-case.md, sandbox/EMULATION.md, api-specification/config.schema.json, api-specification/consumed-contract.schema.json, api-specification/report.schema.json]
outputs: [component-tests/features/validate.feature, component-tests/steps/validate_steps.go]
skills: [component-tests]
---

### TICKET 02 — slice-01-validate/component: realize the 8 designed scenarios as RED `.feature`

**io:** — (test artifact) → skills: `component-tests`

**What this ticket does:** mechanically lay the **already-designed** component set
(`contracts.md` §6 "Component-scenario set", scenario table) into an executable `.feature` +
step-defs + fixtures in the scaffolded `component-tests/` harness, tag every scenario `@wip`,
drive to **RED by business reason** (the modules do not exist yet). Invent **no** new scenario;
the set is frozen at **8** (`1 happy + 7 adapter branches`). R1–R9 verdict cases are **unit**
boundaries (`contracts.md` §4), not component scenarios — do not author them here.

**Scenarios to realize (verbatim wording from `contracts.md` §6, black-box — assert
`(exit code, stdout JSON)`):**
1. `@wip` Happy path: consumer compatible with provider → exit 0, schema-valid report
   `compatible: true`, `errors: []`, `uncovered_channels[]` listed. Fixture: EMULATION.md §1 (S0).
2. `@wip` 2a. Config fails schema validation → exit 2, **no report on stdout**. Representative
   fixture: EMULATION.md S11 (`spec_path` + `spec_url` both set).
3. `@wip` 3a. `consumed_contract_path` is unreachable at run time → exit 3,
   `errors[0].code=FILE_NOT_FOUND`. Fixture: EMULATION.md S12.
4. `@wip` 3b. `consumed-contract` fails to parse or validate → exit 3,
   `errors[0].code=PARSE_ERROR`. Fixture: malformed/consumer-mismatch consumed-contract.
5. `@wip` 4a. `spec_path` is configured but the file is unreachable → exit 3,
   `errors[0].code=FILE_NOT_FOUND`. Fixture: missing spec file.
6. `@wip` 4b. The provider spec does not parse as AsyncAPI 3.0 → exit 3,
   `errors[0].code=PARSE_ERROR`. Fixture: EMULATION.md S13 (bad YAML / `asyncapi: 2.6.0`).
7. `@wip` 4c. `spec_url` is unreachable or returns a non-2xx status → exit 3,
   `errors[0].code=HTTP_ERROR`. Fixture: EMULATION.md S14 (404).
8. `@wip` 4d. The `spec_url` request exceeds `settings.timeout` → exit 3,
   `errors[0].code=TIMEOUT_ERROR`. Fixture: EMULATION.md S15 (slow `spec_url`, `timeout: 1`).

**Fixtures/stubs:** per-scenario `config.yaml` + consumed-contract + provider-spec fixtures;
scenarios 7 & 8 need a **real-protocol HTTP stub** for `spec_url` (404 / stall-past-timeout) —
one-shot-binary flavour. Do **NOT** author unit-level verdict cases here (R1–R9, exit 1) —
those are unit boundaries of `ResolveChannelDirection`/`CompareMessage`, proven per-rule in
`internal/validate/*_test.go`, not component scenarios (`contracts.md` §6 formula).

**Dependencies:** scaffolded `component-tests/` harness (ticket 01).

**Verify:** `bash component-tests/scripts/run-tests.sh` — all 8 scenarios present and **RED** (fail
because the binary is still the placeholder), each tagged `@wip`.

**Acceptance:** all 8 slice component scenarios exist, tagged `@wip`, **RED-ready** (RED for a
business reason, not a harness error). Removing `@wip` + GREEN is the **@fagan acceptance step**,
NOT this ticket.
