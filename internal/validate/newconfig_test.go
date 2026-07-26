package validate

import (
	"errors"
	"testing"
	"time"
)

// baseRawConfig returns a fresh, schema-valid RawConfig (all settings at their zero value, i.e.
// "not set" -> schema defaults apply). Each test case mutates a copy via a mutate func so cases
// never share slice/pointer state.
func baseRawConfig() RawConfig {
	return RawConfig{
		Consumer: Consumer{
			Name:                 "billing-service",
			ConsumedContractPath: "consumed-contract.json",
			Channels:             []string{"RECEIPT.QUEUE.PS"},
		},
		Provider: Provider{
			Name:     "payment-service",
			SpecPath: "spec.yaml",
		},
	}
}

// TestNewConfig realizes contracts.md §4's formula for NewConfig: 1 happy + 10 branches = 11.
func TestNewConfig(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*RawConfig)
		wantErr bool
	}{
		{
			name:    "happy path",
			mutate:  func(r *RawConfig) {},
			wantErr: false,
		},
		{
			name: "empty consumer.name",
			mutate: func(r *RawConfig) {
				r.Consumer.Name = ""
			},
			wantErr: true,
		},
		{
			name: "empty consumed_contract_path",
			mutate: func(r *RawConfig) {
				r.Consumer.ConsumedContractPath = ""
			},
			wantErr: true,
		},
		{
			name: "empty channels",
			mutate: func(r *RawConfig) {
				r.Consumer.Channels = []string{}
			},
			wantErr: true,
		},
		{
			name: "duplicate in channels",
			mutate: func(r *RawConfig) {
				r.Consumer.Channels = []string{"RECEIPT.QUEUE.PS", "RECEIPT.QUEUE.PS"}
			},
			wantErr: true,
		},
		{
			name: "empty-string element in channels",
			mutate: func(r *RawConfig) {
				r.Consumer.Channels = []string{"RECEIPT.QUEUE.PS", ""}
			},
			wantErr: true,
		},
		{
			name: "empty provider.name",
			mutate: func(r *RawConfig) {
				r.Provider.Name = ""
			},
			wantErr: true,
		},
		{
			name: "both spec_path and spec_url set",
			mutate: func(r *RawConfig) {
				r.Provider.SpecURL = "https://example.com/spec.yaml"
			},
			wantErr: true,
		},
		{
			name: "neither spec_path nor spec_url set",
			mutate: func(r *RawConfig) {
				r.Provider.SpecPath = ""
			},
			wantErr: true,
		},
		{
			name: "log_level out of enum",
			mutate: func(r *RawConfig) {
				r.Settings.LogLevel = "trace"
			},
			wantErr: true,
		},
		{
			name: "timeout <= 0",
			mutate: func(r *RawConfig) {
				r.Settings.Timeout = -1
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := baseRawConfig()
			tt.mutate(&raw)

			cfg, err := NewConfig(raw)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("NewConfig(%+v) = nil error, want ErrConfigInvalid", raw)
				}
				if !errors.Is(err, ErrConfigInvalid) {
					t.Fatalf("NewConfig(%+v) error = %v, want errors.Is(err, ErrConfigInvalid)", raw, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("NewConfig(%+v) unexpected error = %v", raw, err)
			}

			// Happy path: verify schema defaults were applied to the unset settings.
			if cfg.settings.LogLevel != "info" {
				t.Errorf("settings.LogLevel = %q, want default %q", cfg.settings.LogLevel, "info")
			}
			if cfg.settings.Timeout != 30*time.Second {
				t.Errorf("settings.Timeout = %v, want default %v", cfg.settings.Timeout, 30*time.Second)
			}
			if !cfg.settings.SaveJSONReport {
				t.Errorf("settings.SaveJSONReport = false, want default true")
			}
			if cfg.settings.JSONReportFile != "compatibility_report.json" {
				t.Errorf("settings.JSONReportFile = %q, want default %q", cfg.settings.JSONReportFile, "compatibility_report.json")
			}
			if cfg.settings.IgnoreWarnings {
				t.Errorf("settings.IgnoreWarnings = true, want default false")
			}
			if cfg.consumer.Name != raw.Consumer.Name {
				t.Errorf("consumer.Name = %q, want %q", cfg.consumer.Name, raw.Consumer.Name)
			}
		})
	}
}
