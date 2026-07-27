# Realized 1:1 from docs/design/slice-01-validate/contracts.md §6 "Component-scenario set", as
# restated by change 001-pipe-arity-coverage-gaps
# (docs/design/slice-01-validate/changes/001-pipe-arity-coverage-gaps/contracts.md §6), and
# docs/design/slice-01-validate/use-case.md (Cockburn Extensions — scenario names reused
# verbatim). N = 2 (happy-class: compatible verdict [scenario 1] + incompatible verdict
# [scenario 9]) + 7 (distinguishable adapter branches: CONFIG_ERROR, FILE_NOT_FOUND x2,
# PARSE_ERROR x2, HTTP_ERROR, TIMEOUT_ERROR) = 9. Do not add/remove scenarios here; a 10th
# scenario or a merged/split scenario is a design act, not a realization act (STOP, back to
# wirth-moduledesigner / contracts.md). R1-R9 verdict cases (Extensions 6a-6i) are unit
# boundaries (contracts.md §4), not component scenarios — do not author them here; scenario 9
# asserts the incompatible verdict end-to-end via one representative rule (R1), exactly as
# scenario 1 asserts the compatible verdict end-to-end. A report-write failure
# (ReportWriter.Write disk/marshal error, the out-of-taxonomy default->exit 3 fallback, D3/C4)
# is deliberately NOT a 10th scenario either — it is not one of the 7 adapter branches; it is
# pinned in cmd/app/main_test.go instead (ticket-03).
#
# ticket-23 wired the real pipe: cmd/app's command surface is now `validate` (was the scaffold's
# placeholder `run`, NOT_IMPLEMENTED/exit 3, ticket-01), and validate_steps.go's exec call was
# updated to match. @wip tags remain below regardless — only the fixer (@fagan) removes @wip, on
# GREEN, at slice acceptance — never this ticket. ticket-01 (001-pipe-arity-coverage-gaps) adds
# scenario 9 (@wip — new surface) and the I2 byte-baseline Then-step on scenario 1 (existing
# surface, untagged).
@component @slice-01-validate
Feature: Forward-compatibility validation of a consumer<->provider pair (async, AsyncAPI 3.0)

  Scenario: Happy path: consumer compatible with provider
    Given a config whose consumed-contract and provider spec are reachable and compatible
    When I run `pinout-asyncapi validate config.yaml`
    Then the exit code is 0
    And stdout is a schema-valid report with compatible=true and errors==[]
    And uncovered provider channels are listed in uncovered_channels[]
    And stdout bytes equal the captured baseline report

  Scenario: 2a. Config fails schema validation
    Given a config where the provider spec source is not exactly one of spec_path/spec_url
    When I run `pinout-asyncapi validate config.yaml`
    Then the exit code is 2
    And no report is printed on stdout

  Scenario: 3a. consumed_contract_path is unreachable at run time
    Given a config pointing at a consumed-contract path that does not exist
    When I run `pinout-asyncapi validate config.yaml`
    Then the exit code is 3
    And stdout report errors[0].code == "FILE_NOT_FOUND"

  Scenario: 3b. consumed-contract fails to parse or validate
    Given a config pointing at a consumed-contract whose consumer does not match config.consumer.name
    When I run `pinout-asyncapi validate config.yaml`
    Then the exit code is 3
    And stdout report errors[0].code == "PARSE_ERROR"

  Scenario: 4a. spec_path is configured but the file is unreachable
    Given a config pointing at a provider spec_path that does not exist
    When I run `pinout-asyncapi validate config.yaml`
    Then the exit code is 3
    And stdout report errors[0].code == "FILE_NOT_FOUND"

  Scenario: 4b. The provider spec does not parse as AsyncAPI 3.0
    Given a config pointing at a provider spec that is not valid AsyncAPI
    When I run `pinout-asyncapi validate config.yaml`
    Then the exit code is 3
    And stdout report errors[0].code == "PARSE_ERROR"

  Scenario: 4c. spec_url is unreachable or returns a non-2xx status
    Given a config with spec_url pointing at a stub that returns 404
    When I run `pinout-asyncapi validate config.yaml`
    Then the exit code is 3
    And stdout report errors[0].code == "HTTP_ERROR"

  Scenario: 4d. The spec_url request exceeds settings.timeout
    Given a config with spec_url pointing at a stub that stalls past settings.timeout
    When I run `pinout-asyncapi validate config.yaml`
    Then the exit code is 3
    And stdout report errors[0].code == "TIMEOUT_ERROR"

  Scenario: Consumer incompatible with provider (primary verdict)
    Given a config whose consumer uses a channel address absent from the provider spec
    When I run `pinout-asyncapi validate config.yaml`
    Then the exit code is 1
    And stdout is a schema-valid report with compatible=false
    And stdout report errors[0].code == "CHANNEL_NOT_IN_PROVIDER"
