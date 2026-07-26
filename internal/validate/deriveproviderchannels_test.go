package validate

import (
	"reflect"
	"testing"

	"github.com/lerenn/asyncapi-codegen/pkg/asyncapi/parser"
)

// mustParseProviderSpec parses doc (an AsyncAPI 3.0 YAML document, servers inlined directly on the
// channel — not via `#/servers/*` `$ref`, so these fixtures don't need InlineServerRefs) through
// the real pinned `lerenn/asyncapi-codegen` parser and Process()es it, mirroring what
// parseProviderSpec (io_spec.go) hands DeriveProviderChannels in production.
func mustParseProviderSpec(t *testing.T, doc string) ProviderSpec {
	t.Helper()

	spec, err := parser.FromYAML(parser.FromYAMLParams{Data: []byte(doc)})
	if err != nil {
		t.Fatalf("parser.FromYAML: %v", err)
	}
	if err := spec.Process(); err != nil {
		t.Fatalf("spec.Process(): %v", err)
	}

	return ProviderSpec{Doc: spec}
}

// TestDeriveProviderChannels realizes contracts.md §4's formula for DeriveProviderChannels: 1
// happy + 2 branches = 3 — the two branches are concept.md D3's two mandatory defaults.
func TestDeriveProviderChannels(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		want ProviderChannels
	}{
		{
			// Happy path: every operation declares an explicit `messages` subset, and there is no
			// `reply` — the baseline, neither default in play. `unusedMsg` on `reqChannel` is
			// deliberately never referenced by any operation, proving the projection carries only
			// what's declared, not everything indiscriminately.
			name: "happy path: explicit operation messages set only the declared direction",
			doc: `
asyncapi: 3.0.0
info: {title: happy, version: 1.0.0}
channels:
  reqChannel:
    address: TEST.REQUEST
    servers:
      - {host: broker:9092, protocol: kafka}
    messages:
      requestMsg: {payload: {type: object}}
      unusedMsg: {payload: {type: object}}
  respChannel:
    address: TEST.RESPONSE
    servers:
      - {host: broker:9092, protocol: kafka}
    messages:
      responseMsg: {payload: {type: object}}
operations:
  receiveRequest:
    action: receive
    channel: {$ref: '#/channels/reqChannel'}
    messages:
      - {$ref: '#/channels/reqChannel/messages/requestMsg'}
  sendResponse:
    action: send
    channel: {$ref: '#/channels/respChannel'}
    messages:
      - {$ref: '#/channels/respChannel/messages/responseMsg'}
`,
			want: ProviderChannels{
				"TEST.REQUEST": ProviderChannel{Protocol: "kafka", Receive: []string{"requestMsg"}, Messages: map[string]ProviderMessage{
					"requestMsg": {Payload: &ProviderSchema{Type: "object"}},
					"unusedMsg":  {Payload: &ProviderSchema{Type: "object"}},
				}},
				"TEST.RESPONSE": ProviderChannel{Protocol: "kafka", Send: []string{"responseMsg"}, Messages: map[string]ProviderMessage{
					"responseMsg": {Payload: &ProviderSchema{Type: "object"}},
				}},
			},
		},
		{
			// Branch 1 (D3 default #1): the operation declares no `messages` at all — must expand
			// to ALL of its channel's messages, not zero.
			name: "branch: operation without messages expands to all channel messages",
			doc: `
asyncapi: 3.0.0
info: {title: expand, version: 1.0.0}
channels:
  auditChannel:
    address: TEST.AUDIT
    servers:
      - {host: broker:9092, protocol: kafka}
    messages:
      eventA: {payload: {type: object}}
      eventB: {payload: {type: object}}
operations:
  receiveAudit:
    action: receive
    channel: {$ref: '#/channels/auditChannel'}
`,
			want: ProviderChannels{
				"TEST.AUDIT": ProviderChannel{Protocol: "kafka", Receive: []string{"eventA", "eventB"}, Messages: map[string]ProviderMessage{
					"eventA": {Payload: &ProviderSchema{Type: "object"}},
					"eventB": {Payload: &ProviderSchema{Type: "object"}},
				}},
			},
		},
		{
			// Branch 2 (D3 default #2): the operation's `reply` must inject the OPPOSITE action
			// onto the reply channel. There is no separate operation on `respChannel` — its whole
			// entry in the projection comes from the `reply`, proving the injection in isolation
			// (mirrors EMULATION.md S3's reverse case). The reply omits its own `messages`, so
			// this also exercises D3's first default nested inside the second.
			name: "branch: operation with reply injects opposite direction on reply channel",
			doc: `
asyncapi: 3.0.0
info: {title: reply, version: 1.0.0}
channels:
  reqChannel:
    address: TEST.BALANCE.REQUEST
    servers:
      - {host: broker:9092, protocol: kafka}
    messages:
      getBalanceRequest: {payload: {type: object}}
  respChannel:
    address: TEST.BALANCE.RESPONSE
    servers:
      - {host: broker:9092, protocol: kafka}
    messages:
      getBalanceResponse: {payload: {type: object}}
operations:
  receiveBalanceRequest:
    action: receive
    channel: {$ref: '#/channels/reqChannel'}
    reply:
      channel: {$ref: '#/channels/respChannel'}
`,
			want: ProviderChannels{
				"TEST.BALANCE.REQUEST": ProviderChannel{Protocol: "kafka", Receive: []string{"getBalanceRequest"}, Messages: map[string]ProviderMessage{
					"getBalanceRequest": {Payload: &ProviderSchema{Type: "object"}},
				}},
				"TEST.BALANCE.RESPONSE": ProviderChannel{Protocol: "kafka", Send: []string{"getBalanceResponse"}, Messages: map[string]ProviderMessage{
					"getBalanceResponse": {Payload: &ProviderSchema{Type: "object"}},
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := mustParseProviderSpec(t, tt.doc)

			got := DeriveProviderChannels(spec)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeriveProviderChannels() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
