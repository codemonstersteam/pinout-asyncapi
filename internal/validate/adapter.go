// adapter.go holds the slice's ingress/egress doors (module-tree.md §6): cli.Parse (ticket-04)
// and cli.ResolveExitCode (this ticket). Both are `io: none` — they carry no domain logic, only
// the cobra-facing shape of the CLI (cli-io discipline).
package validate

import (
	"errors"
	"fmt"
)

// Parse is the ingress door (contracts.md §cli.Parse): argv's one positional arg -> Invocation.
// Antecedent: exactly one non-flag argument present. In practice the caller's cobra command
// enforces that antecedent via its own Args validator (e.g. cobra.ExactArgs(1)) before Parse is
// ever invoked, so a malformed argc surfaces as cobra's usage error, never this function's
// return — the guard below only satisfies the Go (T, error) signature (Result<Invocation,
// Error>) defensively; it is not one of errors.go's domain sentinels.
func Parse(args []string) (Invocation, error) {
	if len(args) != 1 {
		return Invocation{}, fmt.Errorf("validate: expected exactly one argument (config path), got %d", len(args))
	}
	return Invocation{ConfigPath: args[0]}, nil
}

// ResolveExitCode is the egress door (contracts.md §cli.ResolveExitCode): maps
// ProcessValidate's outcome — the Go idiom for Result<Report, Error> is a (Report, error) return
// pair — to the exit-code grid (module-tree.md §5). Purely a Result->int mapping: printing the
// report to stdout / logging to stderr and calling os.Exit is main's job (wiring, ticket-23), not
// this function's.
//
// Antecedent: none — total function over the closed Error taxonomy (errors.go's five sentinels).
// Consequent: exactly one of {0,1,2,3}.
func ResolveExitCode(report Report, err error) int {
	switch {
	case err == nil && report.Compatible:
		// Ok(compatible) — grid row "0 | compatible".
		return 0
	case err == nil:
		// Ok(incompatible) — grid row "1 | incompatible (verdict)".
		return 1
	case errors.Is(err, ErrConfigInvalid):
		// CONFIG_ERROR — grid row "2 | config".
		return 2
	default:
		// FILE_NOT_FOUND | PARSE_ERROR | HTTP_ERROR | TIMEOUT_ERROR — grid row "3 | io · parse".
		// The closed Error taxonomy (errors.go) has no member outside {ErrConfigInvalid,
		// ErrFileNotFound, ErrParseError, ErrHTTPError, ErrTimeoutError}; every non-nil,
		// non-ErrConfigInvalid err is one of the remaining four, all mapping to exit 3.
		return 3
	}
}
