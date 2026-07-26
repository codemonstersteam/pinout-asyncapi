package validate

import "testing"

// TestBuildReportWriter realizes contracts.md §4's formula for BuildReportWriter: 1 happy + 1
// branch = 2 — the construction branch on settings.SaveJSONReport, distinguishable at .Write
// (contracts.md §BuildReportWriter, ticket 21).
func TestBuildReportWriter(t *testing.T) {
	tests := []struct {
		name     string
		settings Settings
		wantPath string
	}{
		{
			name:     "happy path: save_json_report true targets json_report_file",
			settings: Settings{SaveJSONReport: true, JSONReportFile: "report.json"},
			wantPath: "report.json",
		},
		{
			name:     "branch: save_json_report false yields no-op writer",
			settings: Settings{SaveJSONReport: false, JSONReportFile: "report.json"},
			wantPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := BuildReportWriter(tt.settings)

			frw, ok := writer.(FileReportWriter)
			if !ok {
				t.Fatalf("BuildReportWriter() = %T, want FileReportWriter", writer)
			}
			if frw.Path != tt.wantPath {
				t.Fatalf("BuildReportWriter().Path = %q, want %q", frw.Path, tt.wantPath)
			}
		})
	}
}
