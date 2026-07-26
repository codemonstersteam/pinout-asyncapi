package validate

import (
	"errors"
	"strings"
	"testing"
)

// expectedTestConsumer is the fixed `expectedConsumer` dependency (= cfg.Consumer.Name) used
// across TestNewConsumedContract's table.
const expectedTestConsumer = "billing-service"

// validCapturedHash is a schema-valid provenance.captured_hash: `sha256:` + 64 lowercase hex
// digits.
var validCapturedHash = "sha256:" + strings.Repeat("a", 64)

// baseRawContract returns a fresh, schema-valid RawContract (consumer matches
// expectedTestConsumer). Each test case mutates a copy via a mutate func so cases never share
// slice/pointer state.
func baseRawContract() RawContract {
	return RawContract{
		SchemaVersion: "1.0",
		Consumer:      expectedTestConsumer,
		Provenance: Provenance{
			Provider:        "payment-service",
			ProviderVersion: "1.2.3",
			CapturedHash:    validCapturedHash,
		},
		Channels: []Channel{
			{
				Address:  "RECEIPT.QUEUE.PS",
				Protocol: "amqp",
				Sends:    []Message{{Name: "getBalanceRequest"}},
			},
		},
	}
}

// TestNewConsumedContract realizes contracts.md §4's formula for NewConsumedContract: 1 happy +
// 7 branches = 8.
func TestNewConsumedContract(t *testing.T) {
	tests := []struct {
		name    string
		raw     RawContract
		wantErr bool
	}{
		{
			name:    "happy path",
			raw:     baseRawContract(),
			wantErr: false,
		},
		{
			// Decode itself happens upstream in ContractStore.Load; NewConsumedContract can only
			// observe its "undecodable content" antecedent clause as a RawContract that never
			// picked up any decoded values (the zero-value shape an empty/garbage document
			// decodes into).
			name:    "undecodable content",
			raw:     RawContract{},
			wantErr: true,
		},
		{
			name: "schema_version != \"1.0\"",
			raw: func() RawContract {
				r := baseRawContract()
				r.SchemaVersion = "2.0"
				return r
			}(),
			wantErr: true,
		},
		{
			name: "consumer != expectedConsumer",
			raw: func() RawContract {
				r := baseRawContract()
				r.Consumer = "other-service"
				return r
			}(),
			wantErr: true,
		},
		{
			name: "captured_hash regex mismatch",
			raw: func() RawContract {
				r := baseRawContract()
				r.Provenance.CapturedHash = "sha256:not-a-valid-hash"
				return r
			}(),
			wantErr: true,
		},
		{
			name: "empty channels",
			raw: func() RawContract {
				r := baseRawContract()
				r.Channels = []Channel{}
				return r
			}(),
			wantErr: true,
		},
		{
			name: "channel missing address/protocol",
			raw: func() RawContract {
				r := baseRawContract()
				r.Channels[0].Address = ""
				return r
			}(),
			wantErr: true,
		},
		{
			name: "channel with neither sends nor receives",
			raw: func() RawContract {
				r := baseRawContract()
				r.Channels[0].Sends = nil
				r.Channels[0].Receives = nil
				return r
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contract, err := NewConsumedContract(tt.raw, expectedTestConsumer)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("NewConsumedContract(%+v, %q) = nil error, want ErrParseError", tt.raw, expectedTestConsumer)
				}
				if !errors.Is(err, ErrParseError) {
					t.Fatalf("NewConsumedContract(%+v, %q) error = %v, want errors.Is(err, ErrParseError)", tt.raw, expectedTestConsumer, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("NewConsumedContract(%+v, %q) unexpected error = %v", tt.raw, expectedTestConsumer, err)
			}

			if contract.consumer != tt.raw.Consumer {
				t.Errorf("contract.consumer = %q, want %q", contract.consumer, tt.raw.Consumer)
			}
			if contract.provenance != tt.raw.Provenance {
				t.Errorf("contract.provenance = %+v, want %+v", contract.provenance, tt.raw.Provenance)
			}
			if len(contract.channels) != len(tt.raw.Channels) {
				t.Errorf("len(contract.channels) = %d, want %d", len(contract.channels), len(tt.raw.Channels))
			}
		})
	}
}
