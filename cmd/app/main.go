// Command tool — pinout-asyncapi's CLI entry point.
//
// Live wiring (ticket-23, module-tree.md §4): cli.Parse(args) -> Deps (register.go) ->
// ProcessValidate -> code := cli.ResolveExitCode(report, err) -> report JSON always attempted on
// stdout (machine channel) when one exists, diagnostics/logs on stderr, os.Exit(code).
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"pinout-asyncapi/internal/shared/report"
	"pinout-asyncapi/internal/validate"
)

const (
	version = "0.1.0"
	appName = "pinout-asyncapi"
)

// newRootCmd собирает корневую cobra-команду. Вынесено в функцию, чтобы тесты
// могли получить свежее дерево команд без глобального состояния.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           appName,
		Short:         "Forward-compatibility validator for a consumer<->provider AsyncAPI 3.0 pair",
		Long:          appName + " validate <config.yaml> — checks a consumer's declared usage against a provider's AsyncAPI 3.0 spec (internal/validate, slice-01-validate).",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newVersionCmd())
	root.AddCommand(newValidateCmd())
	return root
}

// newVersionCmd — стабильная команда: печатает версию, exit 0. Используется smoke-тестом
// (структурная проверка «бинарь запускается и печатает что-то»), доменом не затрагивается.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Показать версию",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "%s version %s\n", appName, version)
			return nil
		},
	}
}

// newValidateCmd — the live ingress door: argv -> cli.Parse -> Deps (register.go) ->
// ProcessValidate -> cli.ResolveExitCode -> stdout/stderr/exit code (module-tree.md §2, §4).
func newValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <config.yaml>",
		Short: "Validate a consumer contract against a provider AsyncAPI 3.0 spec",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inv, err := validate.Parse(args)
			if err != nil {
				// Defensive only — cobra.ExactArgs(1) already enforces Parse's antecedent
				// (adapter.go's own doc comment); never reached in practice.
				return err
			}

			rep, procErr := validate.ProcessValidate(inv, validate.NewDeps())
			code := validate.ResolveExitCode(rep, procErr)

			switch {
			case code == 2:
				// CONFIG_ERROR (module-tree.md §5): no report written, on stdout or otherwise
				// (use-case.md Minimal guarantee, "on exit ∈ {2} no report is written") — the
				// diagnostic goes to stderr instead, the only channel carrying anything here.
				fmt.Fprintln(cmd.ErrOrStderr(), procErr)
			case procErr != nil:
				// exit 3: the pipe short-circuited before FoldReport ever ran (io/parse
				// failure), so ProcessValidate's own report value carries nothing yet.
				// use-case.md's Minimal guarantee only says such a report "may carry the
				// io/parse code in errors[]" (not schema-valid canon-1.1, that's only
				// mandated for exit ∈ {0,1}) — synthesize just that much here. The error is
				// already embedded as errors[0].message; deliberately not ALSO duplicated to
				// stderr — stdout is the sole machine channel and must stay clean JSON.
				_ = report.WriteJSON(cmd.OutOrStdout(), reportForError(procErr))
			default:
				// exit 0/1: the real canon-1.1 report FoldReport produced.
				_ = report.WriteJSON(cmd.OutOrStdout(), rep)
			}

			if code == 0 {
				return nil
			}
			return &exitError{code: code}
		},
	}
}

// reportForError synthesizes the minimal exit-3 report body (see newValidateCmd's comment
// above) from ProcessValidate's raw error. errorCode classifies it via the same closed sentinel
// taxonomy cli.ResolveExitCode itself switches on (errors.go); CONFIG_ERROR never reaches here
// (handled separately, no report at all).
func reportForError(err error) validate.Report {
	return validate.Report{
		Compatible: false,
		Errors: []validate.Violation{{
			Code:    errorCode(err),
			Message: err.Error(),
		}},
	}
}

// errorCode maps err onto its report.schema.json error code via the closed sentinel taxonomy
// (errors.go) — every sentinel's Error() text already IS its code string.
func errorCode(err error) string {
	switch {
	case errors.Is(err, validate.ErrFileNotFound):
		return validate.ErrFileNotFound.Error()
	case errors.Is(err, validate.ErrParseError):
		return validate.ErrParseError.Error()
	case errors.Is(err, validate.ErrHTTPError):
		return validate.ErrHTTPError.Error()
	case errors.Is(err, validate.ErrTimeoutError):
		return validate.ErrTimeoutError.Error()
	default:
		return ""
	}
}

// exitError несёт exit-код через cobra наружу в main().
type exitError struct{ code int }

func (e *exitError) Error() string { return fmt.Sprintf("exit code %d", e.code) }

func main() {
	if err := newRootCmd().Execute(); err != nil {
		var ee *exitError
		if errors.As(err, &ee) {
			os.Exit(ee.code)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
