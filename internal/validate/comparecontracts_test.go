package validate

import "testing"

// baseCompareContractsComparison returns a fresh Comparison fixture over two consumer channels —
// WALLET.BALANCE.REQUEST (send) and WALLET.TRANSFER.REQUEST (send) — both bound to the same
// provider protocol ("kafka"), modeled on EMULATION.md's shared-server-protocol base pair (S0/S2's
// setup before mutation). A third provider-only channel, WALLET.NOTIFY.EVENT, is never selected by
// consumer.channels — the uncovered_channels fixture (EMULATION.md S0).
// Same-package test, so unexported fields are set directly — CompareContracts's antecedent (a valid
// Comparison) is NewComparison's own concern to verify, not this fold's.
func baseCompareContractsComparison() Comparison {
	cfg := Config{
		consumer: Consumer{
			Name:                 "wallet-consumer",
			ConsumedContractPath: "consumed-contract.json",
			Channels:             []string{"WALLET.TRANSFER.REQUEST", "WALLET.BALANCE.REQUEST"},
		},
		provider: Provider{
			Name:     "wallet-provider",
			SpecPath: "spec.yaml",
		},
	}

	contract := ConsumedContract{
		consumer: "wallet-consumer",
		provenance: Provenance{
			Provider:        "wallet-provider",
			ProviderVersion: "1.0.0",
			CapturedHash:    "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		},
		channels: []Channel{
			{
				Address:  "WALLET.BALANCE.REQUEST",
				Protocol: "kafka",
				Sends:    []Message{{Name: "getBalanceRequest"}},
			},
			{
				Address:  "WALLET.TRANSFER.REQUEST",
				Protocol: "kafka",
				Sends:    []Message{{Name: "getTransferRequest"}},
			},
		},
	}

	pchans := ProviderChannels{
		"WALLET.BALANCE.REQUEST":  ProviderChannel{Protocol: "kafka", Receive: []string{"getBalanceRequest"}},
		"WALLET.TRANSFER.REQUEST": ProviderChannel{Protocol: "kafka", Receive: []string{"getTransferRequest"}},
		"WALLET.NOTIFY.EVENT":     ProviderChannel{Protocol: "kafka", Send: []string{"balanceChanged"}},
	}

	return Comparison{config: cfg, contract: contract, pchannels: pchans}
}

// TestCompareContracts realizes contracts.md §4's formula for CompareContracts: 1 happy + 2
// branches = 3.
func TestCompareContracts(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		c := baseCompareContractsComparison()

		got := CompareContracts(c)

		if len(got.Violations) != 0 {
			t.Fatalf("CompareContracts() violations = %v, want none", got.Violations)
		}
		if got.ConsumerName != "wallet-consumer" {
			t.Errorf("ConsumerName = %q, want %q", got.ConsumerName, "wallet-consumer")
		}
		if got.Provenance != c.contract.provenance {
			t.Errorf("Provenance = %+v, want %+v", got.Provenance, c.contract.provenance)
		}
		if len(got.UncoveredChannels) != 1 || got.UncoveredChannels[0] != "WALLET.NOTIFY.EVENT" {
			t.Errorf("UncoveredChannels = %v, want [WALLET.NOTIFY.EVENT]", got.UncoveredChannels)
		}
	})

	t.Run("violations accumulate across channels without short-circuit (EMULATION.md S2)", func(t *testing.T) {
		c := baseCompareContractsComparison()
		// simulate one shared-server protocol mutation (kafka -> amqp) touching both channels.
		for addr, pc := range c.pchannels {
			pc.Protocol = "amqp"
			c.pchannels[addr] = pc
		}

		got := CompareContracts(c)

		if len(got.Violations) != 2 {
			t.Fatalf("CompareContracts() violations = %v, want exactly 2 (one per affected channel, no short-circuit)", got.Violations)
		}
		bySubject := map[string]Violation{}
		for _, v := range got.Violations {
			bySubject[v.Subject] = v
		}
		for _, addr := range []string{"WALLET.BALANCE.REQUEST", "WALLET.TRANSFER.REQUEST"} {
			v, ok := bySubject[addr]
			if !ok {
				t.Fatalf("CompareContracts() violations = %v, want one PROTOCOL_MISMATCH for %q", got.Violations, addr)
			}
			if v.Code != CodeProtocolMismatch {
				t.Errorf("violation for %q Code = %q, want %q", addr, v.Code, CodeProtocolMismatch)
			}
		}
	})

	t.Run("uncovered_channels populated without affecting compatible (EMULATION.md S0)", func(t *testing.T) {
		c := baseCompareContractsComparison()
		c.pchannels["WALLET.HISTORY.EVENT"] = ProviderChannel{Protocol: "kafka", Send: []string{"historyRecorded"}}

		got := CompareContracts(c)

		if len(got.Violations) != 0 {
			t.Fatalf("CompareContracts() violations = %v, want none — uncovered_channels must not affect the verdict", got.Violations)
		}
		want := []string{"WALLET.HISTORY.EVENT", "WALLET.NOTIFY.EVENT"}
		if len(got.UncoveredChannels) != len(want) {
			t.Fatalf("UncoveredChannels = %v, want %v", got.UncoveredChannels, want)
		}
		for i, addr := range want {
			if got.UncoveredChannels[i] != addr {
				t.Errorf("UncoveredChannels[%d] = %q, want %q (sorted)", i, got.UncoveredChannels[i], addr)
			}
		}
	})
}

// TestCompareContracts_ProviderSchemaReachability proves the fix for the gap flagged at the end of
// ticket-18 (decisions.log/memory.md): Comparison.pchannels (DeriveProviderChannels, ticket 14) now
// carries provider message schemas, not just message-key sets, so CompareMessage's R5-R9 are
// reachable end-to-end through CompareContracts's own fold — not just provable at CompareMessage's
// own unit level (10/10 green there already). One send-direction message (WALLET.BALANCE.REQUEST)
// exercises R5/R7/R8/R9 together via a real provider schema/contentType/correlationId mismatch; one
// receive-direction message (WALLET.BALANCE.RESPONSE) exercises R6 via a field the provider dropped.
func TestCompareContracts_ProviderSchemaReachability(t *testing.T) {
	cfg := Config{
		consumer: Consumer{
			Name:                 "wallet-consumer",
			ConsumedContractPath: "consumed-contract.json",
			Channels:             []string{"WALLET.BALANCE.REQUEST", "WALLET.BALANCE.RESPONSE"},
		},
		provider: Provider{Name: "wallet-provider", SpecPath: "spec.yaml"},
	}

	contract := ConsumedContract{
		consumer: "wallet-consumer",
		channels: []Channel{
			{
				Address:  "WALLET.BALANCE.REQUEST",
				Protocol: "kafka",
				Sends: []Message{{
					Name:                  "getBalanceRequest",
					ContentType:           "application/json",
					CorrelationIDLocation: "$message.header#/correlationId",
					Payload: &SchemaNode{
						Type: "object",
						Properties: map[string]*SchemaNode{
							"clientId": {Type: "string"},
						},
					},
				}},
			},
			{
				Address:  "WALLET.BALANCE.RESPONSE",
				Protocol: "kafka",
				Receives: []Message{{
					Name: "getBalanceResponse",
					Payload: &SchemaNode{
						Type: "object",
						Properties: map[string]*SchemaNode{
							"status":  {Type: "string"},
							"balance": {Type: "number"},
						},
					},
				}},
			},
		},
	}

	pchans := ProviderChannels{
		"WALLET.BALANCE.REQUEST": ProviderChannel{
			Protocol: "kafka",
			Receive:  []string{"getBalanceRequest"},
			Messages: map[string]ProviderMessage{
				"getBalanceRequest": {
					// R8: differs from the consumer's declared "application/json".
					ContentType: "application/xml",
					// R9: differs from the consumer's declared header location.
					CorrelationIDLocation: "$message.payload#/correlationId",
					Payload: &ProviderSchema{
						// R7: differs from the consumer's declared "object".
						Type: "array",
						// R5: "requestId" is required but never among what the consumer sends.
						Required: []string{"clientId", "requestId"},
						Properties: map[string]*ProviderSchema{
							"clientId": {Type: "string"},
						},
					},
				},
			},
		},
		"WALLET.BALANCE.RESPONSE": ProviderChannel{
			Protocol: "kafka",
			Send:     []string{"getBalanceResponse"},
			Messages: map[string]ProviderMessage{
				"getBalanceResponse": {
					Payload: &ProviderSchema{
						Type: "object",
						// R6: the consumer reads "balance" too, which the provider no longer offers.
						Properties: map[string]*ProviderSchema{
							"status": {Type: "string"},
						},
					},
				},
			},
		},
	}

	c := Comparison{config: cfg, contract: contract, pchannels: pchans}

	got := CompareContracts(c)

	byCode := map[string][]Violation{}
	for _, v := range got.Violations {
		byCode[v.Code] = append(byCode[v.Code], v)
	}

	for _, code := range []string{
		CodeMissingRequiredSentField,
		CodeReadsFieldNotProvided,
		CodeTypeMismatch,
		CodeContentTypeMismatch,
		CodeCorrelationIDMismatch,
	} {
		if len(byCode[code]) == 0 {
			t.Errorf("CompareContracts() violations = %v, want at least one %s — R5-R9 must be reachable end-to-end through the fold, not just at CompareMessage's own unit level", got.Violations, code)
		}
	}
}
