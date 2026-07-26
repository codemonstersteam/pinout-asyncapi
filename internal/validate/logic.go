// logic.go holds the slice's pure core logic functions (module-tree.md §6) — NewConfig (this
// ticket) is the first; later tickets append one function at a time to this same file. Every
// function here is io: none and unit-tested by formula (contracts.md §4).
package validate

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	asyncapiv3 "github.com/lerenn/asyncapi-codegen/pkg/asyncapi/v3"
)

// NewConfig is the slice's valid-by-construction Config constructor (contracts.md §NewConfig):
// validates raw against config.schema.json's invariants and applies its declared defaults,
// producing a Config whose unexported fields can never hold an invalid value.
//
// Antecedent: consumer.name non-empty; consumer.consumed_contract_path non-empty;
// consumer.channels non-empty, unique, no empty-string elements; provider.name non-empty;
// exactly one of provider.spec_path/provider.spec_url set; settings.log_level in
// {debug,info,warn,error} (empty -> schema default "info"); settings.timeout > 0 when set (zero
// -> schema default 30s; negative -> invalid); every other settings field within its declared
// type/range (zero value -> its schema default, per domain.go's RawSettings note).
//
// Consequent: success -> Config; failure (any antecedent clause) -> ErrConfigInvalid
// (CONFIG_ERROR).
func NewConfig(raw RawConfig) (Config, error) {
	if raw.Consumer.Name == "" {
		return Config{}, fmt.Errorf("%w: consumer.name must be non-empty", ErrConfigInvalid)
	}
	if raw.Consumer.ConsumedContractPath == "" {
		return Config{}, fmt.Errorf("%w: consumer.consumed_contract_path must be non-empty", ErrConfigInvalid)
	}
	if len(raw.Consumer.Channels) == 0 {
		return Config{}, fmt.Errorf("%w: consumer.channels must be non-empty", ErrConfigInvalid)
	}
	seen := make(map[string]bool, len(raw.Consumer.Channels))
	for _, ch := range raw.Consumer.Channels {
		if ch == "" {
			return Config{}, fmt.Errorf("%w: consumer.channels must not contain empty-string elements", ErrConfigInvalid)
		}
		if seen[ch] {
			return Config{}, fmt.Errorf("%w: consumer.channels must be unique, duplicate %q", ErrConfigInvalid, ch)
		}
		seen[ch] = true
	}

	if raw.Provider.Name == "" {
		return Config{}, fmt.Errorf("%w: provider.name must be non-empty", ErrConfigInvalid)
	}
	hasPath := raw.Provider.SpecPath != ""
	hasURL := raw.Provider.SpecURL != ""
	if hasPath == hasURL {
		return Config{}, fmt.Errorf("%w: exactly one of provider.spec_path/provider.spec_url must be set", ErrConfigInvalid)
	}

	logLevel := raw.Settings.LogLevel
	if logLevel == "" {
		logLevel = "info"
	} else {
		switch logLevel {
		case "debug", "info", "warn", "error":
		default:
			return Config{}, fmt.Errorf("%w: settings.log_level %q not in {debug,info,warn,error}", ErrConfigInvalid, logLevel)
		}
	}

	timeoutSeconds := raw.Settings.Timeout
	switch {
	case timeoutSeconds == 0:
		timeoutSeconds = 30
	case timeoutSeconds < 0:
		return Config{}, fmt.Errorf("%w: settings.timeout must be > 0, got %d", ErrConfigInvalid, timeoutSeconds)
	}

	saveJSONReport := true
	if raw.Settings.SaveJSONReport != nil {
		saveJSONReport = *raw.Settings.SaveJSONReport
	}

	jsonReportFile := raw.Settings.JSONReportFile
	if jsonReportFile == "" {
		jsonReportFile = "compatibility_report.json"
	}

	return Config{
		consumer: raw.Consumer,
		provider: raw.Provider,
		settings: Settings{
			LogLevel:       logLevel,
			SaveJSONReport: saveJSONReport,
			JSONReportFile: jsonReportFile,
			Timeout:        time.Duration(timeoutSeconds) * time.Second,
			IgnoreWarnings: raw.Settings.IgnoreWarnings,
		},
	}, nil
}

// capturedHashPattern is consumed-contract.schema.json's provenance.captured_hash pattern:
// `^sha256:[0-9a-f]{64}$`.
var capturedHashPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// NewConsumedContract is the slice's valid-by-construction ConsumedContract constructor
// (contracts.md §NewConsumedContract): validates raw against consumed-contract.schema.json's
// invariants, producing a ConsumedContract whose unexported fields can never hold an invalid
// value. expectedConsumer (= cfg.Consumer.Name) is a scalar config value, not a second data
// entity — it never appears in RawContract.
//
// Antecedent: raw decodes as a well-formed consumed-contract (ContractStore.Load's concern —
// here checked as raw.SchemaVersion/Consumer/Provenance/Channels actually carrying decoded
// values, not the zero-value shape of an empty/garbage document); schema_version == "1.0";
// consumer == expectedConsumer; provenance.captured_hash matches ^sha256:[0-9a-f]{64}$; channels
// non-empty; each channel has non-empty address/protocol and at least one of sends/receives.
//
// Consequent: success -> ConsumedContract; failure (any antecedent clause) -> ErrParseError
// (PARSE_ERROR).
func NewConsumedContract(raw RawContract, expectedConsumer string) (ConsumedContract, error) {
	if raw.SchemaVersion != "1.0" {
		return ConsumedContract{}, fmt.Errorf("%w: schema_version %q, want \"1.0\"", ErrParseError, raw.SchemaVersion)
	}
	if raw.Consumer != expectedConsumer {
		return ConsumedContract{}, fmt.Errorf("%w: consumer %q, want %q", ErrParseError, raw.Consumer, expectedConsumer)
	}
	if !capturedHashPattern.MatchString(raw.Provenance.CapturedHash) {
		return ConsumedContract{}, fmt.Errorf("%w: provenance.captured_hash %q does not match ^sha256:[0-9a-f]{64}$", ErrParseError, raw.Provenance.CapturedHash)
	}
	if len(raw.Channels) == 0 {
		return ConsumedContract{}, fmt.Errorf("%w: channels must be non-empty", ErrParseError)
	}
	for i, ch := range raw.Channels {
		if ch.Address == "" {
			return ConsumedContract{}, fmt.Errorf("%w: channels[%d].address must be non-empty", ErrParseError, i)
		}
		if ch.Protocol == "" {
			return ConsumedContract{}, fmt.Errorf("%w: channels[%d].protocol must be non-empty", ErrParseError, i)
		}
		if len(ch.Sends) == 0 && len(ch.Receives) == 0 {
			return ConsumedContract{}, fmt.Errorf("%w: channels[%d] (%s) must declare at least one of sends/receives", ErrParseError, i, ch.Address)
		}
	}

	return ConsumedContract{
		consumer:   raw.Consumer,
		provenance: raw.Provenance,
		channels:   raw.Channels,
	}, nil
}

// serverRefPrefix is the only $ref root InlineServerRefs resolves — the canonical
// `channel.servers: [{$ref: '#/servers/<name>'}]` form (concept.md D2).
const serverRefPrefix = "#/servers/"

// YAMLTree is the decoded-YAML-tree shape InlineServerRefs operates on (contracts.md
// §InlineServerRefs): the generic nested map/slice/scalar shape yaml.v3 (or any YAML/JSON
// decoder) produces when unmarshaling into `any`, before the document is re-encoded and handed
// to the stock `parser.FromYAML(...).Process()` (tickets 11/12). A named type, not a struct — the
// antecedent is "any shape", not a fixed schema.
type YAMLTree = map[string]any

// InlineServerRefs substitutes `#/servers/*` `$ref`s with the referenced server body, in the
// decoded YAML tree, before the stock `lerenn/asyncapi-codegen` parser runs (contracts.md
// §InlineServerRefs, concept.md D2 — the library's `Process()` does not resolve the root
// `#/servers/` ref, the canonical `channel.servers` form both real specs in this repo's sandbox
// rely on).
//
// Antecedent: a decoded YAML tree (any shape; a malformed document surfaces later at Process(),
// not here).
//
// Consequent: the same tree with every `#/servers/*` ref inlined; a no-op when there are none
// (same consequent shape — not a distinguishable branch) or when doc has no top-level `servers`
// map to resolve against.
func InlineServerRefs(doc YAMLTree) YAMLTree {
	servers, _ := doc["servers"].(map[string]any)
	if len(servers) == 0 {
		return doc
	}
	inlineServerRefsValue(doc, servers)
	return doc
}

// inlineServerRefsValue walks v (a value from a decoded YAML tree — map[string]any, []any, or a
// scalar) and mutates every nested map/slice in place, replacing each `{$ref: "#/servers/<name>"}`
// map with a deep copy of servers[<name>]. Unknown ref targets are left as-is (surfaces later at
// Process(), not here — same "any shape" antecedent as the top-level doc).
func inlineServerRefsValue(v any, servers map[string]any) any {
	switch node := v.(type) {
	case map[string]any:
		if name, ok := serverRefName(node); ok {
			if body, ok := servers[name]; ok {
				return deepCopyYAMLValue(body)
			}
			return node
		}
		for k, val := range node {
			node[k] = inlineServerRefsValue(val, servers)
		}
		return node
	case []any:
		for i, val := range node {
			node[i] = inlineServerRefsValue(val, servers)
		}
		return node
	default:
		return v
	}
}

// serverRefName reports whether m is exactly a `{$ref: "#/servers/<name>"}` map and, if so,
// returns <name>.
func serverRefName(m map[string]any) (string, bool) {
	if len(m) != 1 {
		return "", false
	}
	ref, ok := m["$ref"].(string)
	if !ok || !strings.HasPrefix(ref, serverRefPrefix) {
		return "", false
	}
	return strings.TrimPrefix(ref, serverRefPrefix), true
}

// deepCopyYAMLValue returns a deep copy of v (a decoded YAML tree value) so an inlined server
// body is never aliased across multiple `$ref` sites.
func deepCopyYAMLValue(v any) any {
	switch node := v.(type) {
	case map[string]any:
		cp := make(map[string]any, len(node))
		for k, val := range node {
			cp[k] = deepCopyYAMLValue(val)
		}
		return cp
	case []any:
		cp := make([]any, len(node))
		for i, val := range node {
			cp[i] = deepCopyYAMLValue(val)
		}
		return cp
	default:
		return v
	}
}

// BuildSpecLoader is the slice's late-bound loader-strategy factory (contracts.md
// §BuildSpecLoader, ticket 13): selects and constructs exactly one SpecLoader implementation from
// provider — FileSpecLoader when SpecPath is set, HTTPSpecLoader when SpecURL is set. Must be
// called by the head AFTER NewConfig, so timeout is always the real, already-validated
// cfg.Settings.Timeout — never a hardcoded constant (EMULATION.md S15 guards exactly this: a
// loader built early with a constant timeout makes the TIMEOUT_ERROR scenario unreachable).
// httpClient is only used (encapsulated into the returned HTTPSpecLoader) when the HTTP branch is
// selected — it is never exposed to the head.
//
// Antecedent: exactly one of provider.SpecPath/provider.SpecURL set (guaranteed by NewConfig's
// antecedent).
// Consequent: a SpecLoader — FileSpecLoader when SpecPath set, HTTPSpecLoader when SpecURL set.
// No failure branch (construction only).
func BuildSpecLoader(provider Provider, timeout time.Duration, httpClient *http.Client) SpecLoader {
	if provider.SpecPath != "" {
		return NewFileSpecLoader(provider.SpecPath)
	}
	return NewHTTPSpecLoader(provider.SpecURL, timeout, httpClient)
}

// DeriveProviderChannels is the slice's provider-spec projection (contracts.md
// §DeriveProviderChannels, concept.md D3): builds `address -> {protocol, send, receive}` from the
// parsed, typed provider spec, expanding AsyncAPI 3.0's two mandatory defaults — both required, or
// the projection under-reports what the provider actually does (D3):
//
//  1. an operation with no `messages` means ALL of its channel's messages (allChannelMessageKeys);
//  2. an operation's `reply` injects the OPPOSITE action on the reply channel, with the reply's own
//     `messages` (or, again, all of the reply channel's messages when omitted).
//
// Message identity is the `channel.messages` map key, never `Message.Name` (D8 — `Process()`
// overwrites `Name` with a generated Go identifier; EMULATION.md §0 point 2/S3b).
//
// Antecedent: spec was produced by a successful Process() (typed, refs resolved).
// Consequent: ProviderChannels, total over any valid ProviderSpec — no failure branch. spec.Doc
// not holding the expected typed model (should not happen given the antecedent) degrades to an
// empty projection rather than a panic — still total.
func DeriveProviderChannels(spec ProviderSpec) ProviderChannels {
	doc, ok := spec.Doc.(*asyncapiv3.Specification)
	if !ok || doc == nil {
		return ProviderChannels{}
	}

	channelKeys := indexChannelKeys(doc.Channels)
	result := make(ProviderChannels)

	opNames := make([]string, 0, len(doc.Operations))
	for name := range doc.Operations {
		opNames = append(opNames, name)
	}
	sort.Strings(opNames) // deterministic iteration (module-tree.md/EMULATION.md discipline)

	for _, name := range opNames {
		op := doc.Operations[name]
		if op == nil || op.Channel == nil {
			continue
		}

		ch := op.Channel.Follow()
		addr := channelAddress(ch, channelKeys)
		addProviderDirection(result, addr, channelProtocol(ch), op.Action, operationMessageKeys(op, ch))
		mergeProviderMessages(result, addr, channelProviderMessages(ch, doc.DefaultContentType))

		if op.Reply == nil || op.Reply.Channel == nil {
			continue
		}
		replyCh := op.Reply.Channel.Follow()
		replyAddr := channelAddress(replyCh, channelKeys)
		addProviderDirection(result, replyAddr, channelProtocol(replyCh), oppositeAction(op.Action), replyMessageKeys(op.Reply, replyCh))
		mergeProviderMessages(result, replyAddr, channelProviderMessages(replyCh, doc.DefaultContentType))
	}

	return result
}

// indexChannelKeys maps each top-level channel object to the key it is registered under in
// spec.Channels — the D4 fallback identity ("channel key in `channels`") for a channel whose
// `address` is omitted.
func indexChannelKeys(channels map[string]*asyncapiv3.Channel) map[*asyncapiv3.Channel]string {
	keys := make(map[*asyncapiv3.Channel]string, len(channels))
	for key, ch := range channels {
		keys[ch] = key
	}
	return keys
}

// channelAddress is ch's projection key: its `address` when declared, else the channel's map key
// in `channels` (D4).
func channelAddress(ch *asyncapiv3.Channel, keys map[*asyncapiv3.Channel]string) string {
	if ch.Address != "" {
		return ch.Address
	}
	return keys[ch]
}

// channelProtocol reads the protocol off ch's first bound server. A channel that (invalidly) binds
// no server projects an empty protocol — DeriveProviderChannels has no failure branch to reject it.
func channelProtocol(ch *asyncapiv3.Channel) string {
	for _, srv := range ch.Servers {
		if srv == nil {
			continue
		}
		if p := srv.Follow().Protocol; p != "" {
			return p
		}
	}
	return ""
}

// oppositeAction is the reply-channel action D3 injects: the direction opposite the operation
// carrying the `reply`.
func oppositeAction(a asyncapiv3.OperationAction) asyncapiv3.OperationAction {
	if a.IsSend() {
		return asyncapiv3.OperationActionIsReceive
	}
	return asyncapiv3.OperationActionIsSend
}

// operationMessageKeys resolves op's message keys on ch: op.Messages when declared (matched back
// to ch.Messages's map keys, D8), else ALL of ch's message keys — D3's first default.
func operationMessageKeys(op *asyncapiv3.Operation, ch *asyncapiv3.Channel) []string {
	if len(op.Messages) == 0 {
		return allChannelMessageKeys(ch)
	}
	return matchMessageKeys(op.Messages, ch)
}

// replyMessageKeys resolves reply's message keys on ch, mirroring operationMessageKeys — D3's
// second default applies the same "no messages = all channel messages" rule to a reply.
func replyMessageKeys(reply *asyncapiv3.OperationReply, ch *asyncapiv3.Channel) []string {
	if len(reply.Messages) == 0 {
		return allChannelMessageKeys(ch)
	}
	return matchMessageKeys(reply.Messages, ch)
}

// allChannelMessageKeys returns every message key declared on ch, sorted for deterministic output.
func allChannelMessageKeys(ch *asyncapiv3.Channel) []string {
	keys := make([]string, 0, len(ch.Messages))
	for key := range ch.Messages {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// matchMessageKeys resolves msgs (an operation's or reply's `messages` references) back to their
// `channel.messages` map keys on ch (D8): a `$ref` into `channels/<ch>/messages/<key>` resolves,
// unfollowed, to the exact *Message value stored at that key, so pointer identity — not
// `Message.Name` — is the correct match.
func matchMessageKeys(msgs []*asyncapiv3.Message, ch *asyncapiv3.Channel) []string {
	byPointer := make(map[*asyncapiv3.Message]string, len(ch.Messages))
	for key, m := range ch.Messages {
		byPointer[m] = key
	}

	keys := make([]string, 0, len(msgs))
	for _, m := range msgs {
		if m == nil {
			continue
		}
		if key, ok := byPointer[m]; ok {
			keys = append(keys, key)
			continue
		}
		if key, ok := byPointer[m.Follow()]; ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

// addProviderDirection folds one operation's contribution into result: the address's
// ProviderChannel is created on first touch and every subsequent contribution merges into it
// (module-tree.md — a channel is populated once for each direction it carries, potentially from
// several operations plus a `reply` injection). msgKeys with a zero length is a legitimate
// contribution shape (an operation whose channel declares no messages at all) — it still marks the
// direction as touched, per the "total, no failure branch" consequent.
func addProviderDirection(result ProviderChannels, address, protocol string, action asyncapiv3.OperationAction, msgKeys []string) {
	if address == "" {
		return
	}

	pc, exists := result[address]
	if !exists {
		pc = ProviderChannel{Protocol: protocol}
	} else if pc.Protocol == "" && protocol != "" {
		pc.Protocol = protocol
	}

	switch {
	case action.IsSend():
		pc.Send = mergeSortedUnique(pc.Send, msgKeys)
	case action.IsReceive():
		pc.Receive = mergeSortedUnique(pc.Receive, msgKeys)
	}

	result[address] = pc
}

// mergeSortedUnique unions existing and add, deduplicated and sorted — deterministic regardless of
// which operation contributed which key first.
func mergeSortedUnique(existing, add []string) []string {
	seen := make(map[string]bool, len(existing)+len(add))
	out := make([]string, 0, len(existing)+len(add))
	for _, v := range append(append([]string{}, existing...), add...) {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}

// channelProviderMessages projects EVERY message declared on ch's own `messages` map into
// ProviderMessage form (address-fix for the gap flagged at ticket-18: DeriveProviderChannels
// previously carried only message-key sets, leaving CompareMessage's R5-R9 structurally
// unreachable through CompareContracts). Built once per channel touched, regardless of which
// direction/operation referenced which key — D3's two defaults (no `messages`, `reply`) already
// expand to the full key set upstream (operationMessageKeys/replyMessageKeys), so giving every
// channel message a schema here, unconditionally, carries schemas through both defaults too.
func channelProviderMessages(ch *asyncapiv3.Channel, defaultContentType string) map[string]ProviderMessage {
	if ch == nil || len(ch.Messages) == 0 {
		return nil
	}
	out := make(map[string]ProviderMessage, len(ch.Messages))
	for key, m := range ch.Messages {
		out[key] = toProviderMessage(m, defaultContentType)
	}
	return out
}

// toProviderMessage converts one provider *asyncapiv3.Message (refs already resolved by
// Process(), D8/Follow()) into CompareMessage's ProviderMessage shape: effective contentType (R8
// — `message.contentType`, else the spec's `defaultContentType`), correlationId.location (R9),
// and payload/headers schemas (R5-R7).
func toProviderMessage(m *asyncapiv3.Message, defaultContentType string) ProviderMessage {
	if m == nil {
		return ProviderMessage{}
	}
	resolved := m.Follow()
	if resolved == nil {
		return ProviderMessage{}
	}

	contentType := resolved.ContentType
	if contentType == "" {
		contentType = defaultContentType
	}

	var correlationIDLocation string
	if resolved.CorrelationID != nil {
		correlationIDLocation = resolved.CorrelationID.Location
	}

	return ProviderMessage{
		ContentType:           contentType,
		CorrelationIDLocation: correlationIDLocation,
		Payload:               toProviderSchema(resolved.Payload),
		Headers:               toProviderSchema(resolved.Headers),
	}
}

// toProviderSchema recursively converts one provider *asyncapiv3.Schema (refs already resolved by
// Process(), Follow()) into CompareMessage's ProviderSchema shape — type/format/required at this
// level, then properties/items, mirroring compareSchemaNode's own recursive shape on the consumer
// side (SchemaNode).
func toProviderSchema(s *asyncapiv3.Schema) *ProviderSchema {
	if s == nil {
		return nil
	}
	resolved := s.Follow()
	if resolved == nil {
		return nil
	}

	out := &ProviderSchema{
		Type:     resolved.Type,
		Format:   resolved.Format,
		Required: append([]string(nil), resolved.Required...),
	}

	if len(resolved.Properties) > 0 {
		out.Properties = make(map[string]*ProviderSchema, len(resolved.Properties))
		for key, prop := range resolved.Properties {
			out.Properties[key] = toProviderSchema(prop)
		}
	}
	if resolved.Items != nil {
		out.Items = toProviderSchema(resolved.Items)
	}

	return out
}

// mergeProviderMessages folds messages (one channel's full provider-message catalog,
// channelProviderMessages) into result[address].Messages — created on first touch, same
// create-or-merge discipline as addProviderDirection. A key already present is left untouched
// (every contribution for one address originates from the same channel's own `messages` map, so
// there is nothing to reconcile — first write wins, deterministic regardless of which operation
// touched the address first).
func mergeProviderMessages(result ProviderChannels, address string, messages map[string]ProviderMessage) {
	if address == "" || len(messages) == 0 {
		return
	}

	pc, exists := result[address]
	if !exists {
		pc = ProviderChannel{}
	}
	if pc.Messages == nil {
		pc.Messages = make(map[string]ProviderMessage, len(messages))
	}
	for key, msg := range messages {
		if _, ok := pc.Messages[key]; !ok {
			pc.Messages[key] = msg
		}
	}
	result[address] = pc
}

// NewComparison is the slice's valid-by-construction Comparison constructor (contracts.md
// §NewComparison, module-tree.md §3 note 1 — the design's sanctioned uniting-constructor
// exception): unites the three already-valid parts (Config, ConsumedContract, ProviderChannels)
// into one valid Comparison, performing the one cross-artifact check that can only run here.
//
// Antecedent: every address in cfg.Consumer.Channels is present among contract.Channels[].Address.
//
// Consequent: success -> Comparison; failure -> ErrConfigInvalid (CONFIG_ERROR) — still before any
// report is written.
func NewComparison(cfg Config, contract ConsumedContract, pchans ProviderChannels) (Comparison, error) {
	contractAddresses := make(map[string]bool, len(contract.channels))
	for _, ch := range contract.channels {
		contractAddresses[ch.Address] = true
	}

	for _, addr := range cfg.consumer.Channels {
		if !contractAddresses[addr] {
			return Comparison{}, fmt.Errorf("%w: consumer.channels address %q not present in consumed-contract channels", ErrConfigInvalid, addr)
		}
	}

	return Comparison{
		config:    cfg,
		contract:  contract,
		pchannels: pchans,
	}, nil
}

// Direction identifiers used by ChannelComparison/ResolvedDirection and ResolveChannelDirection
// (contracts.md §ResolveChannelDirection). Values mirror the consumed-contract's own vocabulary
// (Channel.Sends/Channel.Receives) — this module operates on the consumer's side of the
// send/receive inversion (TASK.md "Отличие async от sync"; EMULATION.md §0), never the provider's
// asyncapiv3.OperationAction.
const (
	DirectionSend    = "send"
	DirectionReceive = "receive"
)

// ChannelComparison is ResolveChannelDirection's private input DTO (contracts.md
// §ResolveChannelDirection): one consumer channel, one of its declared directions, plus the
// provider's projected counterpart at that address (ProviderChannels, DeriveProviderChannels's
// output). Not in the frozen message catalog (contracts.md §1) — CompareContracts (ticket 18)
// constructs one ChannelComparison per (address, direction) the consumer actually declares when it
// folds over consumer.channels.
type ChannelComparison struct {
	// Address is the channel identity already matched against the provider projection by address
	// (R1, TASK.md decision #3) — the D4 channel-key fallback is DeriveProviderChannels's own
	// concern; this module only ever sees the resolved address string.
	Address string
	// Direction is the consumer's declared direction under check: DirectionSend (ch.Sends) or
	// DirectionReceive (ch.Receives).
	Direction string
	// Protocol is the consumer's declared protocol for Address (consumed-contract.schema.json
	// channel.protocol), compared against Provider.Protocol by R2.
	Protocol string
	// Messages are the consumer's messages declared on Direction (ch.Sends or ch.Receives). R4
	// checks each Message.Name against the provider's message-key set on the counterpart
	// direction.
	Messages []Message
	// Provider is the provider's projected channel at Address. Meaningless (zero value) when
	// ProviderFound is false — R1 hierarchical-stops before it would be read.
	Provider ProviderChannel
	// ProviderFound reports whether Address exists in the provider's projection at all (R1).
	ProviderFound bool
}

// ResolvedDirection is ResolveChannelDirection's success-shaped output: the (address, direction)
// under check plus the consumer messages whose key matched a provider message key on the
// counterpart direction (R4 survivors) — ready to hand to CompareMessage (ticket 17, contracts.md
// §CompareMessage) one message at a time. A direction that hierarchical-stopped at R1 or R3 carries
// a nil Messages — there is nothing resolved to compare.
type ResolvedDirection struct {
	Address   string
	Direction string
	Messages  []Message
}

// ResolveChannelDirection is the slice's channel/direction resolver (contracts.md
// §ResolveChannelDirection): R1-R4, TASK.md's channel-level rules ("Ядро сверки" — Уровень
// канала). Hierarchical stop (EMULATION.md §3, ticket-16): no channel found (R1) skips R2/R3/R4
// entirely; no counterpart direction found (R3) skips R4. Otherwise rules do not short-circuit
// each other — R2 (protocol) never blocks R3/R4 from running, and R4 evaluates every one of
// ch.Messages, folding one violation per absent key rather than stopping at the first (TASK.md §5,
// "правила не короткозамыкают — нарушения сворачиваются").
//
// Direction inversion (TASK.md "Отличие async от sync"; EMULATION.md §0): a consumer DirectionSend
// is checked against the provider's Receive column (the provider listens for what the consumer
// publishes) and DirectionReceive against Send — see counterpartMessageKeys.
//
// Antecedent: a ChannelComparison for one address, one declared direction.
//
// Consequent: resolved -> the message set to hand to CompareMessage (matched-key survivors only);
// violations -> zero or more of {CodeChannelNotInProvider, CodeProtocolMismatch,
// CodeDirectionNotInProvider, CodeMessageNotInProvider}.
func ResolveChannelDirection(ch ChannelComparison) (ResolvedDirection, []Violation) {
	resolved := ResolvedDirection{Address: ch.Address, Direction: ch.Direction}

	if !ch.ProviderFound {
		return resolved, []Violation{{
			Code:     CodeChannelNotInProvider,
			Message:  fmt.Sprintf("no channel with address %q at the provider", ch.Address),
			Subject:  ch.Address,
			Location: ch.Address,
		}}
	}

	var violations []Violation

	if ch.Protocol != ch.Provider.Protocol {
		violations = append(violations, Violation{
			Code:     CodeProtocolMismatch,
			Message:  fmt.Sprintf("channel %q protocol %q does not match provider protocol %q", ch.Address, ch.Protocol, ch.Provider.Protocol),
			Subject:  ch.Address,
			Location: ch.Address,
			Context:  map[string]string{"consumer": ch.Protocol, "provider": ch.Provider.Protocol},
		})
	}

	counterpart := counterpartMessageKeys(ch.Direction, ch.Provider)
	if len(counterpart) == 0 {
		subject := channelDirectionSubject(ch.Address, ch.Direction)
		violations = append(violations, Violation{
			Code:     CodeDirectionNotInProvider,
			Message:  fmt.Sprintf("provider has no counterpart operation for %s on %q", ch.Direction, ch.Address),
			Subject:  subject,
			Location: subject,
		})
		return resolved, violations
	}

	counterpartSet := make(map[string]bool, len(counterpart))
	for _, key := range counterpart {
		counterpartSet[key] = true
	}

	resolved.Messages = make([]Message, 0, len(ch.Messages))
	for _, m := range ch.Messages {
		if counterpartSet[m.Name] {
			resolved.Messages = append(resolved.Messages, m)
			continue
		}
		subject := channelDirectionMessageSubject(ch.Address, ch.Direction, m.Name)
		violations = append(violations, Violation{
			Code:     CodeMessageNotInProvider,
			Message:  fmt.Sprintf("message %q not among provider's %s messages on %q", m.Name, ch.Direction, ch.Address),
			Subject:  subject,
			Location: subject,
		})
	}

	return resolved, violations
}

// counterpartMessageKeys returns pc's message-key set for the direction OPPOSITE direction — the
// inversion TASK.md's "Отличие async от sync" describes: a consumer DirectionSend is checked
// against the provider's Receive column (the provider listens for what the consumer publishes),
// and DirectionReceive against Send.
func counterpartMessageKeys(direction string, pc ProviderChannel) []string {
	if direction == DirectionSend {
		return pc.Receive
	}
	return pc.Send
}

// channelDirectionSubject is D11's async subject shape with the direction segment resolved:
// "<address> <direction>" (R3 — the channel is known, the counterpart direction is not).
func channelDirectionSubject(address, direction string) string {
	return address + " " + direction
}

// channelDirectionMessageSubject is D11's full async subject shape: "<address> <direction>
// <message key>" (R4 — channel and direction resolved, the named message key is not).
func channelDirectionMessageSubject(address, direction, messageKey string) string {
	return address + " " + direction + " " + messageKey
}

// MessageComparison is CompareMessage's private input DTO (contracts.md §CompareMessage): one
// message pair matched by the `channel.messages` map key (D8) — the consumer's declared Message
// (domain.go) and the provider's projected counterpart — plus the resolved (address, direction)
// under check. A consumer DirectionSend runs R5 (contravariant); DirectionReceive runs R6
// (covariant) — TASK.md "Отличие async от sync". Not in the frozen message catalog —
// CompareContracts (ticket 18) constructs one per ResolveChannelDirection (ticket 16) survivor.
type MessageComparison struct {
	Address    string
	Direction  string
	MessageKey string
	Consumer   Message
	Provider   ProviderMessage
}

// CompareMessage is the slice's message-level comparator (contracts.md §CompareMessage): R5-R9,
// TASK.md's "Ядро сверки" — Уровень сообщения, over both `payload` and `headers` alike. Rules do
// not short-circuit each other (TASK.md §5, "правила не короткозамыкают — нарушения
// сворачиваются") — every distinguishable violation in one run is collected.
//
// R5/R6/R7 share one recursive walk (compareSchemaNode) over the common object/array structure of
// consumer and provider: at every level reached by both sides, R5 (send) or R6 (receive) checks
// the field-set variance and R7 checks type/format; recursion continues into a nested object
// property or array `items` only when BOTH sides have it there — a field missing on one side stops
// the walk on that branch (same hierarchical-stop discipline as ResolveChannelDirection's R1/R3):
// once absence itself has been reported, there is nothing more specific left to compare.
//
// R8/R9 are message-level, not payload/headers-level, and fire only when the consumer declared the
// value under check (D9) — "то, что он не пинил, поставщик волен иметь".
//
// Antecedent: a resolved MessageComparison (message key matched on both sides,
// ResolveChannelDirection).
// Consequent: zero or more of {CodeMissingRequiredSentField, CodeReadsFieldNotProvided,
// CodeTypeMismatch, CodeContentTypeMismatch, CodeCorrelationIDMismatch}.
func CompareMessage(m MessageComparison) []Violation {
	subject := channelDirectionMessageSubject(m.Address, m.Direction, m.MessageKey)

	var violations []Violation
	violations = append(violations, compareSchemaNode(m.Direction, subject, "payload", m.Consumer.Payload, m.Provider.Payload)...)
	violations = append(violations, compareSchemaNode(m.Direction, subject, "headers", m.Consumer.Headers, m.Provider.Headers)...)

	if m.Consumer.ContentType != "" && m.Consumer.ContentType != m.Provider.ContentType {
		violations = append(violations, Violation{
			Code:     CodeContentTypeMismatch,
			Message:  fmt.Sprintf("message %q effective content type %q does not match provider's %q", m.MessageKey, m.Consumer.ContentType, m.Provider.ContentType),
			Subject:  subject,
			Location: subject,
			Context:  map[string]string{"consumer": m.Consumer.ContentType, "provider": m.Provider.ContentType},
		})
	}

	if m.Consumer.CorrelationIDLocation != "" && m.Consumer.CorrelationIDLocation != m.Provider.CorrelationIDLocation {
		violations = append(violations, Violation{
			Code:     CodeCorrelationIDMismatch,
			Message:  fmt.Sprintf("message %q correlationId.location %q does not match provider's %q", m.MessageKey, m.Consumer.CorrelationIDLocation, m.Provider.CorrelationIDLocation),
			Subject:  subject,
			Location: subject,
			Context:  map[string]string{"consumer": m.Consumer.CorrelationIDLocation, "provider": m.Provider.CorrelationIDLocation},
		})
	}

	return violations
}

// compareSchemaNode is CompareMessage's shared R5/R6/R7 walk (see CompareMessage's doc) over one
// level of a payload/headers schema, at path (e.g. "payload", "payload.data", "payload.tags.items").
func compareSchemaNode(direction, subject, path string, consumer *SchemaNode, provider *ProviderSchema) []Violation {
	if consumer == nil && provider == nil {
		return nil
	}

	var violations []Violation

	switch direction {
	case DirectionSend:
		violations = append(violations, missingRequiredSentFields(subject, path, provider, consumer)...)
	case DirectionReceive:
		violations = append(violations, readFieldsNotProvided(subject, path, consumer, provider)...)
	}

	if consumer != nil && provider != nil {
		violations = append(violations, typeFormatMismatches(subject, path, consumer, provider)...)

		if consumer.Items != nil && provider.Items != nil {
			violations = append(violations, compareSchemaNode(direction, subject, path+".items", consumer.Items, provider.Items)...)
		}

		for _, key := range sortedSchemaNodeKeys(consumer.Properties) {
			providerChild, ok := provider.Properties[key]
			if !ok {
				continue // absence already reported by R5/R6 at this level — nothing more specific to recurse into
			}
			violations = append(violations, compareSchemaNode(direction, subject, path+"."+key, consumer.Properties[key], providerChild)...)
		}
	}

	return violations
}

// missingRequiredSentFields is R5 (TASK.md "Ядро сверки" — Уровень сообщения): contravariant for
// `sends` — `required(provider) ⊆ fields(consumer.sends)` at this schema level.
func missingRequiredSentFields(subject, path string, provider *ProviderSchema, consumer *SchemaNode) []Violation {
	if provider == nil || len(provider.Required) == 0 {
		return nil
	}

	sent := make(map[string]bool)
	if consumer != nil {
		for k := range consumer.Properties {
			sent[k] = true
		}
	}

	required := append([]string(nil), provider.Required...)
	sort.Strings(required)

	var violations []Violation
	for _, field := range required {
		if sent[field] {
			continue
		}
		loc := path + "." + field
		violations = append(violations, Violation{
			Code:     CodeMissingRequiredSentField,
			Message:  fmt.Sprintf("%s is required by the provider but not present among what the consumer declares it sends", loc),
			Subject:  subject,
			Location: subject + " " + loc,
			Context:  map[string]string{"field": field},
		})
	}
	return violations
}

// readFieldsNotProvided is R6 (TASK.md "Ядро сверки" — Уровень сообщения): covariant for
// `receives` — `fields(consumer.receives) ⊆ properties(provider)` at this schema level. Catches
// provider field removal (EMULATION.md S5 — the schema is open by default, so only a set
// comparison, not instance validation, catches an absent field).
func readFieldsNotProvided(subject, path string, consumer *SchemaNode, provider *ProviderSchema) []Violation {
	if consumer == nil || len(consumer.Properties) == 0 {
		return nil
	}

	offered := make(map[string]bool)
	if provider != nil {
		for k := range provider.Properties {
			offered[k] = true
		}
	}

	var violations []Violation
	for _, field := range sortedSchemaNodeKeys(consumer.Properties) {
		if offered[field] {
			continue
		}
		loc := path + "." + field
		violations = append(violations, Violation{
			Code:     CodeReadsFieldNotProvided,
			Message:  fmt.Sprintf("%s is read by the consumer but not offered by the provider", loc),
			Subject:  subject,
			Location: subject + " " + loc,
			Context:  map[string]string{"field": field},
		})
	}
	return violations
}

// typeFormatMismatches is R7 (TASK.md "Ядро сверки" — Уровень сообщения): a shared field's `type`
// and `format` must match at this schema level; compareSchemaNode's caller recurses this check
// into nested objects and array `items` (module doc).
func typeFormatMismatches(subject, path string, consumer *SchemaNode, provider *ProviderSchema) []Violation {
	var violations []Violation
	loc := subject + " " + path

	if consumer.Type != provider.Type {
		violations = append(violations, Violation{
			Code:     CodeTypeMismatch,
			Message:  fmt.Sprintf("%s: type diverges between consumer and provider", path),
			Subject:  subject,
			Location: loc,
			Details:  fmt.Sprintf("expected %q, got %q", consumer.Type, provider.Type),
			Context:  map[string]string{"consumer": consumer.Type, "provider": provider.Type, "field": "type"},
		})
	}
	if consumer.Format != provider.Format {
		violations = append(violations, Violation{
			Code:     CodeTypeMismatch,
			Message:  fmt.Sprintf("%s: format diverges between consumer and provider", path),
			Subject:  subject,
			Location: loc,
			Details:  fmt.Sprintf("expected %q, got %q", consumer.Format, provider.Format),
			Context:  map[string]string{"consumer": consumer.Format, "provider": provider.Format, "field": "format"},
		})
	}
	return violations
}

// sortedSchemaNodeKeys returns props's keys sorted — deterministic iteration (module-tree.md /
// EMULATION.md discipline), same rationale as DeriveProviderChannels's sorted walks.
func sortedSchemaNodeKeys(props map[string]*SchemaNode) []string {
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// CompareContracts is the slice's per-run fold (contracts.md §CompareContracts, module-tree.md §4
// head pseudocode — "CORE: R1..R9 fold, never short-circuits across channels"): for every address
// in c.config.consumer.Channels (sorted — determinism, TASK.md DoD: "одни и те же входные байты ⇒
// тот же вердикт... в т.ч. упорядоченный обход каналов"), builds one ChannelComparison per declared
// direction (send/receive) and calls ResolveChannelDirection (ticket 16); for every message it
// resolves, builds a MessageComparison and calls CompareMessage (ticket 17). Every violation from
// every channel/direction/message is accumulated — no channel's violations ever suppress or replace
// another's (EMULATION.md S2 is the regression anchor: a single shared-server protocol mutation
// yields one PROTOCOL_MISMATCH per affected channel, not one collapsed record).
//
// Comparison's ProviderChannels projection (DeriveProviderChannels, ticket 14) carries each
// channel's provider messages (schema, effective contentType, correlationId location) keyed the
// same way as the message-KEY sets per direction, so resolveAndCompareDirection hands
// CompareMessage the REAL ProviderMessage, not a zero value — R5-R9 are reachable end-to-end
// through this fold (fix for the gap flagged at ticket-18), on top of already being proven
// correct at CompareMessage's own unit level (contracts.md §4). Ticket 18's own formula still only
// names R1-R4-territory branches: S2's protocol mismatch and S0's uncovered_channels — R5-R9's
// per-rule branches remain CompareMessage's unit-test formula, not re-enumerated here.
//
// Also computes uncovered_channels[] — provider channels outside consumer.channels, informational
// only, never affecting Violations/compatible (EMULATION.md S0 regression anchor) — and carries
// forward ConsumerName/Provenance for FoldReport (ticket 19).
//
// Antecedent: a valid Comparison (NewComparison's own antecedent already holds by construction).
// Consequent: Outcome{Violations, UncoveredChannels, ConsumerName, Provenance} — never an Error;
// incompatible is a legitimate Outcome value, not a failure (use-case.md, note after Extensions
// 6a-6i).
func CompareContracts(c Comparison) Outcome {
	channelsByAddress := make(map[string]Channel, len(c.contract.channels))
	for _, ch := range c.contract.channels {
		channelsByAddress[ch.Address] = ch
	}

	addresses := append([]string(nil), c.config.consumer.Channels...)
	sort.Strings(addresses) // sorted channel order — determinism (TASK.md DoD)

	selected := make(map[string]bool, len(addresses))
	var violations []Violation
	for _, addr := range addresses {
		selected[addr] = true

		ch, ok := channelsByAddress[addr]
		if !ok {
			// NewComparison's antecedent guarantees every consumer.channels address is present
			// among contract.channels — unreachable in a valid Comparison.
			continue
		}
		pc, found := c.pchannels[addr]

		if len(ch.Sends) > 0 {
			violations = append(violations, resolveAndCompareDirection(addr, DirectionSend, ch.Protocol, ch.Sends, pc, found)...)
		}
		if len(ch.Receives) > 0 {
			violations = append(violations, resolveAndCompareDirection(addr, DirectionReceive, ch.Protocol, ch.Receives, pc, found)...)
		}
	}

	return Outcome{
		Violations:        violations,
		UncoveredChannels: uncoveredChannels(c.pchannels, selected),
		ConsumerName:      c.contract.consumer,
		Provenance:        c.contract.provenance,
	}
}

// resolveAndCompareDirection is CompareContracts's per-(address, direction) step: builds the
// ChannelComparison, calls ResolveChannelDirection, then folds CompareMessage over every message it
// resolves — accumulating every violation from both calls without short-circuiting. Each resolved
// message's REAL provider counterpart (pc.Messages[m.Name], DeriveProviderChannels/ticket 14) is
// handed to CompareMessage — ResolveChannelDirection's R4 already proved the key exists in the
// provider's key set on the counterpart direction, so pc.Messages is expected to carry a matching
// entry; a lookup miss degrades to a zero ProviderMessage (still total, no panic) rather than a
// failure branch.
func resolveAndCompareDirection(address, direction, protocol string, messages []Message, pc ProviderChannel, providerFound bool) []Violation {
	resolved, violations := ResolveChannelDirection(ChannelComparison{
		Address:       address,
		Direction:     direction,
		Protocol:      protocol,
		Messages:      messages,
		Provider:      pc,
		ProviderFound: providerFound,
	})

	for _, m := range resolved.Messages {
		violations = append(violations, CompareMessage(MessageComparison{
			Address:    address,
			Direction:  direction,
			MessageKey: m.Name,
			Consumer:   m,
			Provider:   pc.Messages[m.Name],
		})...)
	}

	return violations
}

// uncoveredChannels is CompareContracts's informational projection (contracts.md §CompareContracts,
// EMULATION.md S0): every pchans address the consumer did NOT select, sorted — never read by any
// verdict/exit computation (EMULATION.md's "информационное не течёт в вердикт").
func uncoveredChannels(pchans ProviderChannels, selected map[string]bool) []string {
	var uncovered []string
	for addr := range pchans {
		if !selected[addr] {
			uncovered = append(uncovered, addr)
		}
	}
	sort.Strings(uncovered)
	return uncovered
}

// Report build-time constants (report.schema.json `validator`/`interaction`/`schema_version` —
// each a `const` in the frozen schema, so this service only ever emits these three literals).
const (
	reportSchemaVersion = "1.1"
	reportValidator     = "pinout-asyncapi"
	reportInteraction   = "async"
)

// FoldReport is the slice's terminal step (contracts.md §FoldReport): assembles the canon-1.1
// Report (report.schema.json) from CompareContracts's Outcome. clock.Now() is called here EXACTLY
// ONCE — the whole slice's only clock read (D10, concept.md: the core never reads the system clock
// directly).
//
// Antecedent: a valid Outcome (CompareContracts's own consequent).
// Consequent: Report, total — no failure branch.
func FoldReport(outcome Outcome, clock Clock) Report {
	errs := outcome.Violations
	if errs == nil {
		// report.schema.json's `errors` is a required array — never emit JSON `null` for "no
		// violations" (encoding/json marshals a nil slice as `null`, not `[]`).
		errs = []Violation{}
	}

	uncovered := outcome.UncoveredChannels
	if uncovered == nil {
		uncovered = []string{}
	}

	return Report{
		SchemaVersion:     reportSchemaVersion,
		Validator:         reportValidator,
		Interaction:       reportInteraction,
		Consumer:          ReportConsumer{Name: outcome.ConsumerName},
		GeneratedAt:       clock.Now().Format(time.RFC3339),
		Compatible:        len(outcome.Violations) == 0,
		Provenance:        outcome.Provenance,
		Errors:            errs,
		UncoveredChannels: uncovered,
	}
}

// BuildReportWriter is the slice's whether/where-to-persist factory (contracts.md
// §BuildReportWriter, ticket 21): constructs the ReportWriter, encapsulating whether/where to
// persist so the head never branches on settings.SaveJSONReport itself (Store-style) —
// FileReportWriter's zero-Path value IS the no-op writer (io_report.go), so returning it verbatim
// when SaveJSONReport is false is all this factory needs to do.
//
// Antecedent: —.
// Consequent: a ReportWriter — no-op internally (Path == "") when settings.SaveJSONReport ==
// false, else a FileReportWriter targeting settings.JSONReportFile. No failure branch
// (construction only).
func BuildReportWriter(settings Settings) ReportWriter {
	if !settings.SaveJSONReport {
		return FileReportWriter{}
	}
	return FileReportWriter{Path: settings.JSONReportFile}
}
