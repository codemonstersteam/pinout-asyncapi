package validate

import (
	"net/http"
	"testing"
	"time"
)

// TestBuildSpecLoader realizes contracts.md §4's formula for BuildSpecLoader: 1 happy + 1 branch
// = 2 — selects FileSpecLoader vs HTTPSpecLoader, both outcomes distinguishable by concrete type.
func TestBuildSpecLoader(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
		want     any // zero value of the expected concrete SpecLoader type
	}{
		{
			name:     "happy path: spec_path set selects FileSpecLoader",
			provider: Provider{Name: "payment-service", SpecPath: "spec.yaml"},
			want:     FileSpecLoader{},
		},
		{
			name:     "branch: spec_url set selects HTTPSpecLoader",
			provider: Provider{Name: "payment-service", SpecURL: "https://example.com/spec.yaml"},
			want:     HTTPSpecLoader{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := BuildSpecLoader(tt.provider, 30*time.Second, http.DefaultClient)

			switch tt.want.(type) {
			case FileSpecLoader:
				if _, ok := loader.(FileSpecLoader); !ok {
					t.Fatalf("BuildSpecLoader() = %T, want FileSpecLoader", loader)
				}
			case HTTPSpecLoader:
				if _, ok := loader.(HTTPSpecLoader); !ok {
					t.Fatalf("BuildSpecLoader() = %T, want HTTPSpecLoader", loader)
				}
			}
		})
	}
}
