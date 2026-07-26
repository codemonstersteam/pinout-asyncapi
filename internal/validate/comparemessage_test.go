package validate

import "testing"

// baseBalanceRequestComparison returns a fresh MessageComparison fixture modeled on
// EMULATION.md §0/§1's base pair: consumer sends getBalanceRequest on WALLET.BALANCE.REQUEST — R5
// (contravariant) territory. Mirrors sandbox/provider/asyncapi.v1.yaml's GetBalanceRequest /
// BalanceRequestPayload and sandbox/consumer/consumed-contract.yaml's matching `sends` entry.
func baseBalanceRequestComparison() MessageComparison {
	return MessageComparison{
		Address:    "WALLET.BALANCE.REQUEST",
		Direction:  DirectionSend,
		MessageKey: "getBalanceRequest",
		Consumer: Message{
			Name:                  "getBalanceRequest",
			ContentType:           "application/json",
			CorrelationIDLocation: "$message.header#/correlationId",
			Headers: &SchemaNode{
				Type: "object",
				Properties: map[string]*SchemaNode{
					"correlationId": {Type: "string"},
				},
			},
			Payload: &SchemaNode{
				Type: "object",
				Properties: map[string]*SchemaNode{
					"clientId":  {Type: "string"},
					"requestId": {Type: "string"},
				},
			},
		},
		Provider: ProviderMessage{
			ContentType:           "application/json",
			CorrelationIDLocation: "$message.header#/correlationId",
			Headers: &ProviderSchema{
				Type:     "object",
				Required: []string{"correlationId"},
				Properties: map[string]*ProviderSchema{
					"correlationId": {Type: "string"},
					"traceparent":   {Type: "string"},
				},
			},
			Payload: &ProviderSchema{
				Type:     "object",
				Required: []string{"clientId", "requestId"},
				Properties: map[string]*ProviderSchema{
					"clientId":  {Type: "string"},
					"requestId": {Type: "string"},
					"locale":    {Type: "string"}, // optional at provider — legal contravariant extension
				},
			},
		},
	}
}

// baseBalanceResponseComparison returns a fresh MessageComparison fixture: consumer receives
// getBalanceResponse on WALLET.BALANCE.RESPONSE — R6 (covariant) territory, with a nested `data`
// object (BalanceData, $ref-resolved upstream) mirroring the sandbox base pair.
func baseBalanceResponseComparison() MessageComparison {
	return MessageComparison{
		Address:    "WALLET.BALANCE.RESPONSE",
		Direction:  DirectionReceive,
		MessageKey: "getBalanceResponse",
		Consumer: Message{
			Name:                  "getBalanceResponse",
			ContentType:           "application/json",
			CorrelationIDLocation: "$message.header#/correlationId",
			Headers: &SchemaNode{
				Type: "object",
				Properties: map[string]*SchemaNode{
					"correlationId": {Type: "string"},
				},
			},
			Payload: &SchemaNode{
				Type: "object",
				Properties: map[string]*SchemaNode{
					"status": {Type: "string"},
					"data": {
						Type: "object",
						Properties: map[string]*SchemaNode{
							"balance":  {Type: "number", Format: "double"},
							"currency": {Type: "string"},
						},
					},
				},
			},
		},
		Provider: ProviderMessage{
			ContentType:           "application/json",
			CorrelationIDLocation: "$message.header#/correlationId",
			Headers: &ProviderSchema{
				Type:     "object",
				Required: []string{"correlationId"},
				Properties: map[string]*ProviderSchema{
					"correlationId": {Type: "string"},
				},
			},
			Payload: &ProviderSchema{
				Type:     "object",
				Required: []string{"status", "data"},
				Properties: map[string]*ProviderSchema{
					"status": {Type: "string"},
					"data": {
						Type:     "object",
						Required: []string{"balance", "currency"},
						Properties: map[string]*ProviderSchema{
							"balance":   {Type: "number", Format: "double"},
							"currency":  {Type: "string"},
							"updatedAt": {Type: "integer", Format: "int64"}, // not read by consumer — legal
						},
					},
					"error": { // not read by consumer — legal covariant extension
						Type: "object",
						Properties: map[string]*ProviderSchema{
							"code":    {Type: "string"},
							"message": {Type: "string"},
						},
					},
				},
			},
		},
	}
}

// baseLedgerPostedComparison returns a fresh MessageComparison fixture: consumer receives
// ledgerPosted on WALLET.LEDGER.EVENTS — array `items` recursion territory (EMULATION.md S10's
// `tags: array of string`).
func baseLedgerPostedComparison() MessageComparison {
	return MessageComparison{
		Address:    "WALLET.LEDGER.EVENTS",
		Direction:  DirectionReceive,
		MessageKey: "ledgerPosted",
		Consumer: Message{
			Name:        "ledgerPosted",
			ContentType: "application/json",
			Payload: &SchemaNode{
				Type: "object",
				Properties: map[string]*SchemaNode{
					"entryId": {Type: "string"},
					"amount":  {Type: "number", Format: "double"},
					"tags": {
						Type:  "array",
						Items: &SchemaNode{Type: "string"},
					},
				},
			},
		},
		Provider: ProviderMessage{
			ContentType: "application/json",
			Payload: &ProviderSchema{
				Type:     "object",
				Required: []string{"entryId", "amount"},
				Properties: map[string]*ProviderSchema{
					"entryId": {Type: "string"},
					"amount":  {Type: "number", Format: "double"},
					"tags": {
						Type:  "array",
						Items: &ProviderSchema{Type: "string"},
					},
					"postedBy": {Type: "string"}, // not read by consumer — legal
				},
			},
		},
	}
}

// TestCompareMessage realizes contracts.md §4's formula: 1 happy + 9 branches (R5-R9) = 10.
func TestCompareMessage(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		violations := CompareMessage(baseBalanceRequestComparison())

		if len(violations) != 0 {
			t.Fatalf("CompareMessage() violations = %v, want none", violations)
		}
	})

	t.Run("R5 missing required sent field", func(t *testing.T) {
		m := baseBalanceRequestComparison()
		m.Provider.Payload.Required = append(m.Provider.Payload.Required, "idempotencyKey")

		violations := CompareMessage(m)

		if len(violations) != 1 {
			t.Fatalf("CompareMessage() violations = %v, want exactly 1", violations)
		}
		v := violations[0]
		if v.Code != CodeMissingRequiredSentField {
			t.Errorf("violations[0].Code = %q, want %q", v.Code, CodeMissingRequiredSentField)
		}
		want := "WALLET.BALANCE.REQUEST send getBalanceRequest payload.idempotencyKey"
		if v.Location != want {
			t.Errorf("violations[0].Location = %q, want %q", v.Location, want)
		}
	})

	t.Run("R6 read field not provided", func(t *testing.T) {
		m := baseBalanceResponseComparison()
		delete(m.Provider.Payload.Properties["data"].Properties, "currency")
		m.Provider.Payload.Properties["data"].Required = []string{"balance"}

		violations := CompareMessage(m)

		if len(violations) != 1 {
			t.Fatalf("CompareMessage() violations = %v, want exactly 1", violations)
		}
		v := violations[0]
		if v.Code != CodeReadsFieldNotProvided {
			t.Errorf("violations[0].Code = %q, want %q", v.Code, CodeReadsFieldNotProvided)
		}
		want := "WALLET.BALANCE.RESPONSE receive getBalanceResponse payload.data.currency"
		if v.Location != want {
			t.Errorf("violations[0].Location = %q, want %q", v.Location, want)
		}
	})

	t.Run("R7 type mismatch", func(t *testing.T) {
		m := baseBalanceResponseComparison()
		m.Provider.Payload.Properties["status"].Type = "integer"

		violations := CompareMessage(m)

		if len(violations) != 1 {
			t.Fatalf("CompareMessage() violations = %v, want exactly 1", violations)
		}
		v := violations[0]
		if v.Code != CodeTypeMismatch {
			t.Errorf("violations[0].Code = %q, want %q", v.Code, CodeTypeMismatch)
		}
		want := "WALLET.BALANCE.RESPONSE receive getBalanceResponse payload.status"
		if v.Location != want {
			t.Errorf("violations[0].Location = %q, want %q", v.Location, want)
		}
		if got, want := v.Context["consumer"], "string"; got != want {
			t.Errorf("violations[0].Context[consumer] = %q, want %q", got, want)
		}
		if got, want := v.Context["provider"], "integer"; got != want {
			t.Errorf("violations[0].Context[provider] = %q, want %q", got, want)
		}
	})

	t.Run("R7 format mismatch", func(t *testing.T) {
		m := baseBalanceResponseComparison()
		m.Provider.Payload.Properties["data"].Properties["balance"].Format = "float"

		violations := CompareMessage(m)

		if len(violations) != 1 {
			t.Fatalf("CompareMessage() violations = %v, want exactly 1", violations)
		}
		v := violations[0]
		if v.Code != CodeTypeMismatch {
			t.Errorf("violations[0].Code = %q, want %q", v.Code, CodeTypeMismatch)
		}
		want := "WALLET.BALANCE.RESPONSE receive getBalanceResponse payload.data.balance"
		if v.Location != want {
			t.Errorf("violations[0].Location = %q, want %q", v.Location, want)
		}
		if got, want := v.Context["consumer"], "double"; got != want {
			t.Errorf("violations[0].Context[consumer] = %q, want %q", got, want)
		}
		if got, want := v.Context["provider"], "float"; got != want {
			t.Errorf("violations[0].Context[provider] = %q, want %q", got, want)
		}
	})

	t.Run("R7 recursion — nested object / array items / resolved $ref", func(t *testing.T) {
		// Nested object: BalanceData was resolved from a `$ref` upstream (D2/D8's concern, before
		// CompareMessage ever sees it) — here it is a plain 2-level-deep object (payload.data.balance).
		m := baseBalanceResponseComparison()
		m.Provider.Payload.Properties["data"].Properties["balance"].Type = "string"

		violations := CompareMessage(m)

		if len(violations) != 1 {
			t.Fatalf("CompareMessage() violations = %v, want exactly 1", violations)
		}
		v := violations[0]
		if v.Code != CodeTypeMismatch {
			t.Errorf("violations[0].Code = %q, want %q", v.Code, CodeTypeMismatch)
		}
		want := "WALLET.BALANCE.RESPONSE receive getBalanceResponse payload.data.balance"
		if v.Location != want {
			t.Errorf("violations[0].Location = %q, want %q", v.Location, want)
		}

		// Array items: EMULATION.md S10's `tags: array of string` — recursion into Items.
		lm := baseLedgerPostedComparison()
		lm.Provider.Payload.Properties["tags"].Items.Type = "integer"

		lviolations := CompareMessage(lm)

		if len(lviolations) != 1 {
			t.Fatalf("CompareMessage() (ledgerPosted) violations = %v, want exactly 1", lviolations)
		}
		lv := lviolations[0]
		if lv.Code != CodeTypeMismatch {
			t.Errorf("violations[0].Code = %q, want %q", lv.Code, CodeTypeMismatch)
		}
		wantItems := "WALLET.LEDGER.EVENTS receive ledgerPosted payload.tags.items"
		if lv.Location != wantItems {
			t.Errorf("violations[0].Location = %q, want %q", lv.Location, wantItems)
		}
	})

	t.Run("R8 content-type mismatch (declared)", func(t *testing.T) {
		m := baseBalanceRequestComparison()
		m.Provider.ContentType = "application/avro"

		violations := CompareMessage(m)

		if len(violations) != 1 {
			t.Fatalf("CompareMessage() violations = %v, want exactly 1", violations)
		}
		v := violations[0]
		if v.Code != CodeContentTypeMismatch {
			t.Errorf("violations[0].Code = %q, want %q", v.Code, CodeContentTypeMismatch)
		}
		want := "WALLET.BALANCE.REQUEST send getBalanceRequest"
		if v.Location != want {
			t.Errorf("violations[0].Location = %q, want %q", v.Location, want)
		}
		if got, want := v.Context["consumer"], "application/json"; got != want {
			t.Errorf("violations[0].Context[consumer] = %q, want %q", got, want)
		}
		if got, want := v.Context["provider"], "application/avro"; got != want {
			t.Errorf("violations[0].Context[provider] = %q, want %q", got, want)
		}
	})

	t.Run("R8 not checked (undeclared)", func(t *testing.T) {
		m := baseBalanceRequestComparison()
		m.Consumer.ContentType = ""
		m.Provider.ContentType = "application/avro"

		violations := CompareMessage(m)

		if len(violations) != 0 {
			t.Fatalf("CompareMessage() violations = %v, want none (consumer did not pin content_type)", violations)
		}
	})

	t.Run("R9 correlationId mismatch (declared)", func(t *testing.T) {
		m := baseBalanceRequestComparison()
		m.Provider.CorrelationIDLocation = "$message.payload#/corrId"

		violations := CompareMessage(m)

		if len(violations) != 1 {
			t.Fatalf("CompareMessage() violations = %v, want exactly 1", violations)
		}
		v := violations[0]
		if v.Code != CodeCorrelationIDMismatch {
			t.Errorf("violations[0].Code = %q, want %q", v.Code, CodeCorrelationIDMismatch)
		}
		want := "WALLET.BALANCE.REQUEST send getBalanceRequest"
		if v.Location != want {
			t.Errorf("violations[0].Location = %q, want %q", v.Location, want)
		}
		if got, want := v.Context["consumer"], "$message.header#/correlationId"; got != want {
			t.Errorf("violations[0].Context[consumer] = %q, want %q", got, want)
		}
		if got, want := v.Context["provider"], "$message.payload#/corrId"; got != want {
			t.Errorf("violations[0].Context[provider] = %q, want %q", got, want)
		}
	})

	t.Run("R9 not checked (undeclared)", func(t *testing.T) {
		m := baseBalanceRequestComparison()
		m.Consumer.CorrelationIDLocation = ""
		m.Provider.CorrelationIDLocation = "$message.payload#/corrId"

		violations := CompareMessage(m)

		if len(violations) != 0 {
			t.Fatalf("CompareMessage() violations = %v, want none (consumer did not pin correlation_id_location)", violations)
		}
	})
}
