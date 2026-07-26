package validate

import "testing"

// baseChannelComparison returns a fresh ChannelComparison fixture modeled on EMULATION.md §0/§1's
// base pair: consumer sends getBalanceRequest on WALLET.BALANCE.REQUEST (kafka), and the provider
// projection listens for exactly that message on the counterpart (receive) direction — S0's
// channel-level row, before any mutation.
func baseChannelComparison() ChannelComparison {
	return ChannelComparison{
		Address:   "WALLET.BALANCE.REQUEST",
		Direction: DirectionSend,
		Protocol:  "kafka",
		Messages:  []Message{{Name: "getBalanceRequest"}},
		Provider: ProviderChannel{
			Protocol: "kafka",
			Receive:  []string{"getBalanceRequest"},
		},
		ProviderFound: true,
	}
}

// TestResolveChannelDirection realizes contracts.md §4's formula: 1 happy + 4 branches (R1-R4) = 5.
func TestResolveChannelDirection(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		ch := baseChannelComparison()

		resolved, violations := ResolveChannelDirection(ch)

		if len(violations) != 0 {
			t.Fatalf("ResolveChannelDirection() violations = %v, want none", violations)
		}
		if resolved.Address != ch.Address || resolved.Direction != ch.Direction {
			t.Errorf("resolved = {%q %q}, want {%q %q}", resolved.Address, resolved.Direction, ch.Address, ch.Direction)
		}
		if len(resolved.Messages) != 1 || resolved.Messages[0].Name != "getBalanceRequest" {
			t.Errorf("resolved.Messages = %v, want [{Name: getBalanceRequest}]", resolved.Messages)
		}
	})

	t.Run("R1 channel absent from provider projection", func(t *testing.T) {
		ch := baseChannelComparison()
		ch.ProviderFound = false
		ch.Provider = ProviderChannel{}

		resolved, violations := ResolveChannelDirection(ch)

		if len(violations) != 1 {
			t.Fatalf("ResolveChannelDirection() violations = %v, want exactly 1", violations)
		}
		v := violations[0]
		if v.Code != CodeChannelNotInProvider {
			t.Errorf("violations[0].Code = %q, want %q", v.Code, CodeChannelNotInProvider)
		}
		if v.Location != "WALLET.BALANCE.REQUEST" {
			t.Errorf("violations[0].Location = %q, want %q", v.Location, "WALLET.BALANCE.REQUEST")
		}
		if resolved.Messages != nil {
			t.Errorf("resolved.Messages = %v, want nil (hierarchical stop)", resolved.Messages)
		}
	})

	t.Run("R2 protocol mismatch", func(t *testing.T) {
		ch := baseChannelComparison()
		ch.Provider.Protocol = "amqp"

		resolved, violations := ResolveChannelDirection(ch)

		if len(violations) != 1 {
			t.Fatalf("ResolveChannelDirection() violations = %v, want exactly 1", violations)
		}
		v := violations[0]
		if v.Code != CodeProtocolMismatch {
			t.Errorf("violations[0].Code = %q, want %q", v.Code, CodeProtocolMismatch)
		}
		if v.Location != "WALLET.BALANCE.REQUEST" {
			t.Errorf("violations[0].Location = %q, want %q", v.Location, "WALLET.BALANCE.REQUEST")
		}
		if got, want := v.Context["consumer"], "kafka"; got != want {
			t.Errorf("violations[0].Context[consumer] = %q, want %q", got, want)
		}
		if got, want := v.Context["provider"], "amqp"; got != want {
			t.Errorf("violations[0].Context[provider] = %q, want %q", got, want)
		}
		// R2 does not short-circuit R3/R4 (TASK.md §5): the message still resolves.
		if len(resolved.Messages) != 1 || resolved.Messages[0].Name != "getBalanceRequest" {
			t.Errorf("resolved.Messages = %v, want [{Name: getBalanceRequest}] (R2 must not block R3/R4)", resolved.Messages)
		}
	})

	t.Run("R3 direction absent from provider", func(t *testing.T) {
		ch := baseChannelComparison()
		ch.Provider.Receive = nil

		resolved, violations := ResolveChannelDirection(ch)

		if len(violations) != 1 {
			t.Fatalf("ResolveChannelDirection() violations = %v, want exactly 1", violations)
		}
		v := violations[0]
		if v.Code != CodeDirectionNotInProvider {
			t.Errorf("violations[0].Code = %q, want %q", v.Code, CodeDirectionNotInProvider)
		}
		if v.Location != "WALLET.BALANCE.REQUEST send" {
			t.Errorf("violations[0].Location = %q, want %q", v.Location, "WALLET.BALANCE.REQUEST send")
		}
		if resolved.Messages != nil {
			t.Errorf("resolved.Messages = %v, want nil (hierarchical stop)", resolved.Messages)
		}
	})

	t.Run("R4 message key absent", func(t *testing.T) {
		ch := baseChannelComparison()
		ch.Provider.Receive = []string{"getBalanceRequestV2"}

		resolved, violations := ResolveChannelDirection(ch)

		if len(violations) != 1 {
			t.Fatalf("ResolveChannelDirection() violations = %v, want exactly 1", violations)
		}
		v := violations[0]
		if v.Code != CodeMessageNotInProvider {
			t.Errorf("violations[0].Code = %q, want %q", v.Code, CodeMessageNotInProvider)
		}
		if v.Location != "WALLET.BALANCE.REQUEST send getBalanceRequest" {
			t.Errorf("violations[0].Location = %q, want %q", v.Location, "WALLET.BALANCE.REQUEST send getBalanceRequest")
		}
		if len(resolved.Messages) != 0 {
			t.Errorf("resolved.Messages = %v, want empty (message key did not match)", resolved.Messages)
		}
	})
}
