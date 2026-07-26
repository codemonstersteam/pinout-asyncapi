// Package validate implements the pinout-asyncapi validate slice (slice-01-validate):
// UC1 "consumer runs pinout-asyncapi validate against a provider AsyncAPI 3.0 spec".
//
// domain.go declares the slice's shared value types (message catalog, contracts.md §1) and the
// Clock port. It carries no behavior — see docs/design/slice-01-validate/module-tree.md §6 for
// the full file layout and docs/design/slice-01-validate/contracts.md for module contracts.
package validate

import "time"

// Clock is the slice's only source of the current time (D10: the core never reads system time
// directly). Declared here — not in head.go — so both FoldReport (logic.go, ticket 19) and
// head.go's Deps (ticket 22) can reference it without a forward reference.
type Clock interface {
	Now() time.Time
}

// Invocation is the CLI ingress door's unvalidated request (cli.Parse's output): one config file
// path from argv. Not yet checked for existence/readability — that happens at ConfigStore.Load.
type Invocation struct {
	ConfigPath string
}

// RawConfig is config.yaml decoded as YAML/JSON, pre-validation (config.schema.json). Produced by
// ConfigStore.Load; consumed only by NewConfig, which is solely responsible for turning it into a
// valid-by-construction Config.
type RawConfig struct {
	Consumer Consumer    `yaml:"consumer" json:"consumer"`
	Provider Provider    `yaml:"provider" json:"provider"`
	Settings RawSettings `yaml:"settings" json:"settings"`
}

// RawSettings is the pre-validation, pre-defaulting shape of config.yaml's optional `settings`
// object. SaveJSONReport is a pointer so NewConfig can distinguish "not set" (apply the
// schema default, true) from an explicit `false`; every other field defaults on its zero value.
type RawSettings struct {
	LogLevel       string `yaml:"log_level" json:"log_level"`
	SaveJSONReport *bool  `yaml:"save_json_report" json:"save_json_report"`
	JSONReportFile string `yaml:"json_report_file" json:"json_report_file"`
	Timeout        int    `yaml:"timeout" json:"timeout"`
	IgnoreWarnings bool   `yaml:"ignore_warnings" json:"ignore_warnings"`
}

// Config is the validated, valid-by-construction shape of config.yaml (config.schema.json).
// Unexported fields — built only via NewConfig (logic.go, ticket 07); no naked composite literal
// of this type may appear outside logic.go.
type Config struct {
	consumer Consumer
	provider Provider
	settings Settings
}

// Consumer is the consumer-under-test sub-object of Config (and reused, pre-validation, by
// RawConfig — its field shapes never change between raw and validated form).
type Consumer struct {
	Name                 string   `yaml:"name" json:"name"`
	ConsumedContractPath string   `yaml:"consumed_contract_path" json:"consumed_contract_path"`
	Channels             []string `yaml:"channels" json:"channels"`
}

// Provider is the provider-spec-source sub-object of Config (and reused, pre-validation, by
// RawConfig): exactly one of SpecPath/SpecURL is set once validated (NewConfig's antecedent).
type Provider struct {
	Name     string `yaml:"name" json:"name"`
	SpecPath string `yaml:"spec_path" json:"spec_path"`
	SpecURL  string `yaml:"spec_url" json:"spec_url"`
}

// Settings is the validated, defaulted run-settings sub-object of Config.
type Settings struct {
	LogLevel       string
	SaveJSONReport bool
	JSONReportFile string
	Timeout        time.Duration
	IgnoreWarnings bool
}

// RawContract is the consumed-contract artifact decoded as YAML/JSON, pre-validation
// (consumed-contract.schema.json). Produced by ContractStore.Load; consumed only by
// NewConsumedContract.
type RawContract struct {
	SchemaVersion string     `yaml:"schema_version" json:"schema_version"`
	Consumer      string     `yaml:"consumer" json:"consumer"`
	Provenance    Provenance `yaml:"provenance" json:"provenance"`
	Channels      []Channel  `yaml:"channels" json:"channels"`
}

// ConsumedContract is the validated, valid-by-construction shape of the consumed-contract
// artifact. Unexported fields — built only via NewConsumedContract (logic.go, ticket 09).
type ConsumedContract struct {
	consumer   string
	provenance Provenance
	channels   []Channel
}

// Channel is one consumer-declared channel (consumed-contract.schema.json $defs/channel); reused,
// pre-validation, by RawContract.
type Channel struct {
	Address  string    `yaml:"address" json:"address"`
	Protocol string    `yaml:"protocol" json:"protocol"`
	Sends    []Message `yaml:"sends" json:"sends"`
	Receives []Message `yaml:"receives" json:"receives"`
}

// Message is one message on one direction of one Channel (consumed-contract.schema.json
// $defs/message). ContentType/CorrelationIDLocation empty means "not declared" (D9, R8/R9) — the
// zero value is the meaningful "absent" state, not an invalid one.
type Message struct {
	Name                  string      `yaml:"name" json:"name"`
	ContentType           string      `yaml:"content_type" json:"content_type"`
	CorrelationIDLocation string      `yaml:"correlation_id_location" json:"correlation_id_location"`
	Payload               *SchemaNode `yaml:"payload" json:"payload"`
	Headers               *SchemaNode `yaml:"headers" json:"headers"`
}

// SchemaNode mirrors the consumed-contract.schema.json $defs/schema captured field surface.
// Deliberately no `required`/`enum` — the consumed-contract states what the consumer sends/reads,
// not what it demands (schema note).
type SchemaNode struct {
	Type       string                 `yaml:"type" json:"type"`
	Format     string                 `yaml:"format" json:"format"`
	Properties map[string]*SchemaNode `yaml:"properties" json:"properties"`
	Items      *SchemaNode            `yaml:"items" json:"items"`
}

// Provenance stamps which provider spec a consumed-contract was captured from; echoed verbatim
// into Report.Provenance (canon 1.1's single source of provider identity).
type Provenance struct {
	Provider        string `yaml:"provider" json:"provider"`
	ProviderVersion string `yaml:"provider_version" json:"provider_version"`
	CapturedHash    string `yaml:"captured_hash" json:"captured_hash"`
}

// ProviderSpec is a thin, opaque wrapper around the parsed provider AsyncAPI 3.0 document (the
// lerenn/asyncapi-codegen typed model returned by parser.FromYAML(...).Process(), post
// InlineServerRefs). Nothing outside DeriveProviderChannels (io_spec.go) reads its internals; the
// concrete Doc value is populated by the spec-loading tickets that wire the parser library.
type ProviderSpec struct {
	Doc any
}

// ProviderChannel is DeriveProviderChannels's per-address projection: which message keys the
// provider sends/receives on this channel, over which protocol, plus the provider-side schema
// data (contracts.md §CompareMessage) for every message key declared on the channel — keyed the
// same way as Send/Receive (D8: `channel.messages` map key), so CompareContracts can hand
// CompareMessage the real ProviderMessage instead of a zero value (gap flagged at ticket-18).
type ProviderChannel struct {
	Protocol string
	Send     []string
	Receive  []string
	Messages map[string]ProviderMessage
}

// ProviderChannels is DeriveProviderChannels's output: address -> ProviderChannel.
type ProviderChannels map[string]ProviderChannel

// ProviderSchema is CompareMessage's provider-side schema representation for one of
// payload/headers (contracts.md §CompareMessage). Unlike the consumer's SchemaNode (above) —
// deliberately field-of-record for what the consumer sends/reads, no `required` — ProviderSchema
// also carries `required`: R5 checks what the provider demands, R6/R7 check what it offers.
// DeriveProviderChannels (logic.go, ticket 14) builds one from the provider's AsyncAPI 3.0 message
// schema, $refs already resolved by parser.Process() (ticket 12) — CompareMessage never sees a raw
// `$ref` itself.
type ProviderSchema struct {
	Type       string
	Format     string
	Required   []string
	Properties map[string]*ProviderSchema
	Items      *ProviderSchema
}

// ProviderMessage is CompareMessage's provider-side message representation (contracts.md
// §CompareMessage): the provider's effective contentType (`message.contentType`, else the spec's
// `defaultContentType` — R8), `correlationId.location` (R9), and payload/headers schemas (R5-R7).
// Populated by DeriveProviderChannels (logic.go, ticket 14) into ProviderChannel.Messages.
type ProviderMessage struct {
	ContentType           string
	CorrelationIDLocation string
	Payload               *ProviderSchema
	Headers               *ProviderSchema
}

// Comparison unites an already-valid Config, ConsumedContract and ProviderChannels into one
// valid-by-construction whole. Unexported fields — built only via NewComparison (logic.go,
// ticket 13), the design's sanctioned uniting-constructor exception (module-tree.md §3).
type Comparison struct {
	config    Config
	contract  ConsumedContract
	pchannels ProviderChannels
}

// Violation is one distinguishable rule failure (verdict codes R1-R9) or io/parse failure,
// shaped to serialize directly into Report.Errors (report.schema.json errors[] item).
//
// json tags mirror report.schema.json's errors[] item exactly: code/message/subject always
// present (required, non-empty by construction); location/details/context are declared but not
// required, so they carry `omitempty` — Context especially must, since Go's zero value (nil map)
// would otherwise marshal to JSON `null`, which fails the schema's `context: {"type": "object"}`
// (null is not an object). (Gap flagged by ticket 19, fixed here — ticket 20, the first module
// that actually serializes Report to JSON.)
type Violation struct {
	Code     string            `json:"code"`
	Message  string            `json:"message"`
	Subject  string            `json:"subject"`
	Location string            `json:"location,omitempty"`
	Details  string            `json:"details,omitempty"`
	Context  map[string]string `json:"context,omitempty"`
}

// Outcome is CompareContracts's output: the folded verdict across every consumer channel, plus
// everything FoldReport needs besides the clock. incompatible is a legitimate value here, never
// an Error (use-case.md, note after Extensions 6a-6i).
type Outcome struct {
	Violations        []Violation
	UncoveredChannels []string
	ConsumerName      string
	Provenance        Provenance
}

// ReportConsumer is Report's consumer-identity sub-object (report.schema.json `consumer`).
// Version is optional and emitted only when actually known (never a placeholder).
type ReportConsumer struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// Report is the canon-1.1 compatibility report (report.schema.json), FoldReport's output and the
// service's sole output contract.
type Report struct {
	SchemaVersion     string         `json:"schema_version"`
	Validator         string         `json:"validator"`
	Interaction       string         `json:"interaction"`
	Consumer          ReportConsumer `json:"consumer"`
	GeneratedAt       string         `json:"generated_at"`
	Compatible        bool           `json:"compatible"`
	Provenance        Provenance     `json:"provenance"`
	Errors            []Violation    `json:"errors"`
	UncoveredChannels []string       `json:"uncovered_channels"`
}
