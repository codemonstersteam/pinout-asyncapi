package validate

import "errors"

// Sentinel errors — one per error.code that is NOT a domain verdict (contracts.md §2). Each rises
// untransformed through the pipe (module-tree.md §4) to cli.ResolveExitCode, the only place an
// error class is mapped to an exit code.
var (
	// ErrConfigInvalid is CONFIG_ERROR (exit 2): ConfigStore.Load (unreadable/undecodable
	// config.yaml), NewConfig (config.schema.json breach), or NewComparison (a consumer channel
	// address absent from the consumed-contract). Never appears in Report.Errors — detected
	// before FoldReport runs.
	ErrConfigInvalid = errors.New("CONFIG_ERROR")

	// ErrFileNotFound is FILE_NOT_FOUND (exit 3): ContractStore.Load or FileSpecLoader.Load found
	// the artifact missing/unreadable.
	ErrFileNotFound = errors.New("FILE_NOT_FOUND")

	// ErrParseError is PARSE_ERROR (exit 3): NewConsumedContract (schema/consumer-name/hash
	// breach) or FileSpecLoader.Load/HTTPSpecLoader.Load (body doesn't parse as AsyncAPI 3.0).
	ErrParseError = errors.New("PARSE_ERROR")

	// ErrHTTPError is HTTP_ERROR (exit 3): HTTPSpecLoader.Load found spec_url unreachable or
	// returning a non-2xx status.
	ErrHTTPError = errors.New("HTTP_ERROR")

	// ErrTimeoutError is TIMEOUT_ERROR (exit 3): HTTPSpecLoader.Load exceeded settings.timeout.
	ErrTimeoutError = errors.New("TIMEOUT_ERROR")
)

// Violation.Code constants for the nine verdict rules R1-R9 (contracts.md §2,
// report.schema.json errors[].code enum). Raised by ResolveChannelDirection (R1-R4) and
// CompareMessage (R5-R9); never by an I/O module.
const (
	// CodeChannelNotInProvider is R1: a consumer channel address absent from the provider spec.
	CodeChannelNotInProvider = "CHANNEL_NOT_IN_PROVIDER"

	// CodeProtocolMismatch is R2: consumer and provider protocol differ at the same address.
	CodeProtocolMismatch = "PROTOCOL_MISMATCH"

	// CodeDirectionNotInProvider is R3: the provider has no counterpart operation for the
	// consumer's declared direction.
	CodeDirectionNotInProvider = "DIRECTION_NOT_IN_PROVIDER"

	// CodeMessageNotInProvider is R4: the consumer names a message key absent from the provider
	// direction's `messages` map.
	CodeMessageNotInProvider = "MESSAGE_NOT_IN_PROVIDER"

	// CodeMissingRequiredSentField is R5: a field the provider requires on `receive` is absent
	// from the consumer's `sends` payload/headers (contravariant check).
	CodeMissingRequiredSentField = "MISSING_REQUIRED_SENT_FIELD"

	// CodeReadsFieldNotProvided is R6: the consumer's `receives` payload/headers reads a field
	// the provider's `send` message does not offer (covariant check).
	CodeReadsFieldNotProvided = "READS_FIELD_NOT_PROVIDED"

	// CodeTypeMismatch is R7: a shared field's type/format differs between consumer and provider
	// (checked recursively over objects/arrays).
	CodeTypeMismatch = "TYPE_MISMATCH"

	// CodeContentTypeMismatch is R8: the consumer declared a contentType that differs from the
	// provider's effective content type. Checked only when the consumer declared one.
	CodeContentTypeMismatch = "CONTENT_TYPE_MISMATCH"

	// CodeCorrelationIDMismatch is R9: the consumer declared a correlationId.location that
	// differs from the provider's. Checked only when the consumer declared one.
	CodeCorrelationIDMismatch = "CORRELATION_ID_MISMATCH"
)
