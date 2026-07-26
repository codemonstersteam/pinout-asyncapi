package validate

import (
	"encoding/json"
	"testing"
)

// TestReportJSONShape_MatchesSchema is a TEMPORARY sanity check (not part of ticket-20.md's own
// formula — ReportWriter.Write is an I/O pipe, unit-tested: none) requested alongside ticket 20
// because this ticket is the first module that actually serializes Report to JSON, and the fix it
// carries (domain.go's Violation gaining json tags, see ticket 19's flagged gap) is otherwise
// unverified. No JSON-schema validator dependency exists in go.mod/go.sum (checked), so this
// structurally compares json.Marshal(Report)'s key set — and its errors[] item's key set — against
// api-specification/report.schema.json's `properties`/`required` field-by-field, rather than
// pulling in a new dependency for one test. Delete freely once a real schema-validating check
// exists (e.g. a component-test fixture assertion).
func TestReportJSONShape_MatchesSchema(t *testing.T) {
	report := Report{
		SchemaVersion: "1.1",
		Validator:     "pinout-asyncapi",
		Interaction:   "async",
		Consumer:      ReportConsumer{Name: "wallet-consumer"},
		GeneratedAt:   "2026-07-25T00:00:00Z",
		Compatible:    false,
		Provenance: Provenance{
			Provider:        "wallet-provider",
			ProviderVersion: "1.0.0",
			CapturedHash:    "sha256:" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcd",
		},
		Errors: []Violation{
			{
				Code:     CodeTypeMismatch,
				Message:  "type mismatch",
				Subject:  "WALLET.BALANCE.RESPONSE receive getBalanceResponse",
				Location: "WALLET.BALANCE.RESPONSE receive getBalanceResponse payload.data.balance",
				Details:  "expected number, got string",
				Context:  map[string]string{"consumer_protocol": "kafka", "provider_protocol": "amqp"},
			},
			{
				// Second element deliberately omits Location/Details/Context (zero values) to
				// prove omitempty drops them cleanly rather than emitting "" / null.
				Code:    CodeChannelNotInProvider,
				Message: "channel not in provider",
				Subject: "WALLET.BALANCE.RESPONSE",
			},
		},
		UncoveredChannels: []string{"WALLET.OTHER.CHANNEL"},
	}

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal(report): %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	// report.schema.json properties (additionalProperties: false — this set must match exactly).
	wantTop := []string{
		"schema_version", "validator", "interaction", "consumer", "generated_at",
		"compatible", "provenance", "errors", "uncovered_channels",
	}
	assertExactKeys(t, decoded, wantTop, "top-level report")

	requiredTop := []string{
		"schema_version", "validator", "interaction", "consumer", "generated_at",
		"compatible", "provenance", "errors",
	}
	for _, k := range requiredTop {
		if _, ok := decoded[k]; !ok {
			t.Errorf("required top-level field %q missing", k)
		}
	}

	consumer, ok := decoded["consumer"].(map[string]any)
	if !ok {
		t.Fatalf("consumer: want object, got %T", decoded["consumer"])
	}
	assertExactKeys(t, consumer, []string{"name"}, "consumer") // version omitted (zero value)

	provenance, ok := decoded["provenance"].(map[string]any)
	if !ok {
		t.Fatalf("provenance: want object, got %T", decoded["provenance"])
	}
	assertExactKeys(t, provenance, []string{"provider", "provider_version", "captured_hash"}, "provenance")

	errs, ok := decoded["errors"].([]any)
	if !ok {
		t.Fatalf("errors: want array, got %T", decoded["errors"])
	}
	if len(errs) != 2 {
		t.Fatalf("errors: want 2 items, got %d", len(errs))
	}

	first, ok := errs[0].(map[string]any)
	if !ok {
		t.Fatalf("errors[0]: want object, got %T", errs[0])
	}
	assertExactKeys(t, first, []string{"code", "message", "subject", "location", "details", "context"}, "errors[0]")
	if ctx, ok := first["context"].(map[string]any); !ok {
		t.Errorf("errors[0].context: want object (schema type:object), got %T (nil/null fails the schema)", first["context"])
	} else if len(ctx) != 2 {
		t.Errorf("errors[0].context: want 2 keys, got %d", len(ctx))
	}

	second, ok := errs[1].(map[string]any)
	if !ok {
		t.Fatalf("errors[1]: want object, got %T", errs[1])
	}
	// Zero-value location/details/context must be entirely ABSENT (omitempty), not present as
	// "" / null — a present `context: null` would violate the schema's type:"object".
	assertExactKeys(t, second, []string{"code", "message", "subject"}, "errors[1] (omitempty fields)")

	if _, ok := decoded["uncovered_channels"].([]any); !ok {
		t.Errorf("uncovered_channels: want array, got %T", decoded["uncovered_channels"])
	}
}

func assertExactKeys(t *testing.T, obj map[string]any, want []string, label string) {
	t.Helper()

	wantSet := make(map[string]bool, len(want))
	for _, k := range want {
		wantSet[k] = true
	}

	for k := range obj {
		if !wantSet[k] {
			t.Errorf("%s: unexpected key %q (report.schema.json additionalProperties: false)", label, k)
		}
	}
	for _, k := range want {
		if _, ok := obj[k]; !ok {
			t.Errorf("%s: expected key %q missing", label, k)
		}
	}
}
