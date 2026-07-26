# Realized 1:1 from docs/design/slice-01-validate/contracts.md §6 "Component-scenario set"
# (the scenario table there is the verbatim source) and docs/design/slice-01-validate/use-case.md
# (Cockburn Extensions — scenario names reused verbatim). N = 1 (happy) + 7 (distinguishable
# adapter branches: CONFIG_ERROR, FILE_NOT_FOUND x2, PARSE_ERROR x2, HTTP_ERROR, TIMEOUT_ERROR)
# = 8. Do not add/remove scenarios here; a 9th scenario or a merged/split scenario is a design
# act, not a realization act (STOP, back to wirth-moduledesigner / contracts.md). R1-R9 verdict
# cases are unit boundaries (contracts.md §4), not component scenarios — do not author them here.
#
# ticket-23 wired the real pipe: cmd/app's command surface is now `validate` (was the scaffold's
# placeholder `run`, NOT_IMPLEMENTED/exit 3, ticket-01), and validate_steps.go's exec call was
# updated to match. @wip tags remain below regardless — only the fixer (@fagan) removes @wip, on
# GREEN, at slice acceptance — never this ticket.
@component @slice-01-validate
Feature: Forward-compatibility validation of a consumer<->provider pair (async, AsyncAPI 3.0)

  Scenario: Happy path: consumer compatible with provider
    Given a config whose consumed-contract and provider spec are reachable and compatible
    When I run `pinout-asyncapi validate config.yaml`
    Then the exit code is 0
    And stdout is a schema-valid report with compatible=true and errors==[]
    And uncovered provider channels are listed in uncovered_channels[]

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
