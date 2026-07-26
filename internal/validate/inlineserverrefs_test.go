package validate

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// serverRefFixture mirrors sandbox/provider/asyncapi.v1.yaml's shape (concept.md D2): a
// top-level `servers` map and a channel whose `servers` list holds a `$ref` into it — the
// canonical `channel.servers` form the pinned parser (v0.63.0) fails to resolve on its own.
const serverRefFixture = `
servers:
  broker:
    host: kafka:9092
    protocol: kafka
    description: Основная шина
channels:
  balanceRequest:
    address: WALLET.BALANCE.REQUEST
    servers:
      - $ref: '#/servers/broker'
    messages:
      getBalanceRequest:
        $ref: '#/components/messages/GetBalanceRequest'
`

// TestInlineServerRefs realizes contracts.md §4's formula for InlineServerRefs: 1 happy + 0
// branches = 1 (the no-op case is the same consequent shape, not a second branch).
func TestInlineServerRefs(t *testing.T) {
	var doc YAMLTree
	if err := yaml.Unmarshal([]byte(serverRefFixture), &doc); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}

	got := InlineServerRefs(doc)

	channels, ok := got["channels"].(map[string]any)
	if !ok {
		t.Fatalf("channels = %T, want map[string]any", got["channels"])
	}
	balanceRequest, ok := channels["balanceRequest"].(map[string]any)
	if !ok {
		t.Fatalf("channels.balanceRequest = %T, want map[string]any", channels["balanceRequest"])
	}
	channelServers, ok := balanceRequest["servers"].([]any)
	if !ok || len(channelServers) != 1 {
		t.Fatalf("channels.balanceRequest.servers = %#v, want a 1-element slice", balanceRequest["servers"])
	}
	inlined, ok := channelServers[0].(map[string]any)
	if !ok {
		t.Fatalf("channels.balanceRequest.servers[0] = %T, want map[string]any", channelServers[0])
	}

	if _, hasRef := inlined["$ref"]; hasRef {
		t.Errorf("channels.balanceRequest.servers[0] still has $ref: %#v", inlined)
	}
	if got := inlined["host"]; got != "kafka:9092" {
		t.Errorf("inlined server host = %v, want %q", got, "kafka:9092")
	}
	if got := inlined["protocol"]; got != "kafka" {
		t.Errorf("inlined server protocol = %v, want %q", got, "kafka")
	}

	// The unresolved `#/components/messages/...` $ref elsewhere in the tree is untouched —
	// InlineServerRefs only resolves `#/servers/*`.
	messages, ok := balanceRequest["messages"].(map[string]any)
	if !ok {
		t.Fatalf("channels.balanceRequest.messages = %T, want map[string]any", balanceRequest["messages"])
	}
	getBalanceRequest, ok := messages["getBalanceRequest"].(map[string]any)
	if !ok {
		t.Fatalf("messages.getBalanceRequest = %T, want map[string]any", messages["getBalanceRequest"])
	}
	if ref := getBalanceRequest["$ref"]; ref != "#/components/messages/GetBalanceRequest" {
		t.Errorf("messages.getBalanceRequest.$ref = %v, want unchanged", ref)
	}

	// servers.broker itself is untouched (still the original body, not a ref).
	servers, ok := got["servers"].(map[string]any)
	if !ok {
		t.Fatalf("servers = %T, want map[string]any", got["servers"])
	}
	broker, ok := servers["broker"].(map[string]any)
	if !ok {
		t.Fatalf("servers.broker = %T, want map[string]any", servers["broker"])
	}
	if broker["host"] != "kafka:9092" {
		t.Errorf("servers.broker.host = %v, want %q", broker["host"], "kafka:9092")
	}
}
