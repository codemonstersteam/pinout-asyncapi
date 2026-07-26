package validate

import (
	"errors"
	"testing"
)

// baseComparisonInputs returns a fresh, mutually-consistent (Config, ConsumedContract,
// ProviderChannels) triple — cfg.consumer.Channels is a subset of contract.channels' addresses.
// Same-package test, so unexported fields are set directly (valid-by-construction is
// NewComparison's own antecedent to verify, not something the test must route through
// NewConfig/NewConsumedContract to obtain).
func baseComparisonInputs() (Config, ConsumedContract, ProviderChannels) {
	cfg := Config{
		consumer: Consumer{
			Name:                 "billing-service",
			ConsumedContractPath: "consumed-contract.json",
			Channels:             []string{"RECEIPT.QUEUE.PS"},
		},
		provider: Provider{
			Name:     "payment-service",
			SpecPath: "spec.yaml",
		},
	}

	contract := ConsumedContract{
		consumer: "billing-service",
		channels: []Channel{
			{
				Address:  "RECEIPT.QUEUE.PS",
				Protocol: "amqp",
				Receives: []Message{{Name: "ReceiptIssued"}},
			},
		},
	}

	pchans := ProviderChannels{
		"RECEIPT.QUEUE.PS": ProviderChannel{Protocol: "amqp", Send: []string{"ReceiptIssued"}},
	}

	return cfg, contract, pchans
}

// TestNewComparison realizes contracts.md §4's formula for NewComparison: 1 happy + 1 branch = 2.
func TestNewComparison(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		cfg, contract, pchans := baseComparisonInputs()

		got, err := NewComparison(ComparisonInput{Config: cfg, Contract: contract, ProviderChannels: pchans})
		if err != nil {
			t.Fatalf("NewComparison() unexpected error = %v", err)
		}
		if got.config.consumer.Name != cfg.consumer.Name {
			t.Errorf("config.consumer.Name = %q, want %q", got.config.consumer.Name, cfg.consumer.Name)
		}
		if got.contract.consumer != contract.consumer {
			t.Errorf("contract.consumer = %q, want %q", got.contract.consumer, contract.consumer)
		}
		if len(got.pchannels) != len(pchans) {
			t.Errorf("pchannels = %v, want %v", got.pchannels, pchans)
		}
	})

	t.Run("cfg.Consumer.Channels address absent from contract.Channels", func(t *testing.T) {
		cfg, contract, pchans := baseComparisonInputs()
		cfg.consumer.Channels = []string{"RECEIPT.QUEUE.PS", "ORDER.QUEUE.PS"}

		_, err := NewComparison(ComparisonInput{Config: cfg, Contract: contract, ProviderChannels: pchans})
		if err == nil {
			t.Fatalf("NewComparison() = nil error, want ErrConfigInvalid")
		}
		if !errors.Is(err, ErrConfigInvalid) {
			t.Fatalf("NewComparison() error = %v, want errors.Is(err, ErrConfigInvalid)", err)
		}
	})
}
