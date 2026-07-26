// io_report.go holds ReportWriter, the slice's report-file filesystem pipe (module-tree.md §6).
// io: none — a pure pipe, no logic, not unit-tested (an I/O module, module-tree.md §3 hard rule).
package validate

import (
	"encoding/json"
	"fmt"
	"os"
)

// ReportWriter is the pipe that persists a Report to disk when the run is configured to
// (contracts.md §ReportWriter.Write). Declared here — not logic.go — since it is this file's own
// I/O door; BuildReportWriter (logic.go, ticket 21) is the factory that decides, from Settings,
// which concrete value to hand back.
type ReportWriter interface {
	// Write writes report's bytes to the writer's configured destination (or no-ops, per how the
	// writer was constructed) and always returns report unchanged (ROP pass-through — the pipe
	// never partially mutates its input) so the caller can keep chaining regardless of outcome.
	Write(report Report) (Report, error)
}

// FileReportWriter is ReportWriter's sole implementation: a filesystem pipe writing report.json's
// canon-1.1 bytes (report.schema.json) to Path. The zero value (Path == "") is the no-op writer —
// BuildReportWriter returns it verbatim when settings.save_json_report == false, so Write
// short-circuits before touching the filesystem (Store-style: the head never branches on
// save_json_report itself, contracts.md §BuildReportWriter).
type FileReportWriter struct {
	Path string
}

// Write is the pipe (contracts.md §ReportWriter.Write): marshals report to canon-1.1 JSON bytes
// and writes them to w.Path. A no-op when w.Path == "" (disabled, per construction).
//
// Antecedent: —.
// Consequent: always Report unchanged (ROP pass-through — never partially mutates); nil error on
// success or no-op. A marshal/disk-write failure here has no code in the frozen error taxonomy
// (report.schema.json x-exit-codes has none for it) — out of scope for this design
// (contracts.md §ReportWriter.Write note); the wrapped error is still returned so nothing is
// silently swallowed, but it is deliberately not one of errors.go's sentinels — callers must not
// map it onto the closed {CONFIG_ERROR, FILE_NOT_FOUND, PARSE_ERROR, HTTP_ERROR, TIMEOUT_ERROR}
// taxonomy.
func (w FileReportWriter) Write(report Report) (Report, error) {
	if w.Path == "" {
		return report, nil
	}

	data, err := json.Marshal(report)
	if err != nil {
		return report, fmt.Errorf("validate: marshal report for %s: %w", w.Path, err)
	}

	if err := os.WriteFile(w.Path, data, 0o644); err != nil {
		return report, fmt.Errorf("validate: write report to %s: %w", w.Path, err)
	}

	return report, nil
}
