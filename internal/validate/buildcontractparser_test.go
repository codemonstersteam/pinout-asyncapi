package validate

import "testing"

// TestBuildContractParser realizes contracts.md §4's formula for BuildContractParser: 1 happy +
// 0 branches = 1 — it binds an already-`NewConfig`-validated scalar and has no antecedent branch
// of its own (contracts.md §BuildContractParser).
func TestBuildContractParser(t *testing.T) {
	tests := []struct {
		name         string
		consumerName string
	}{
		{
			name:         "happy path: consumerName is bound into the parser",
			consumerName: "billing-service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := BuildContractParser(tt.consumerName)

			if parser.expectedConsumer != tt.consumerName {
				t.Fatalf("BuildContractParser(%q).expectedConsumer = %q, want %q", tt.consumerName, parser.expectedConsumer, tt.consumerName)
			}
		})
	}
}
