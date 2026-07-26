// io_spec.go holds SpecLoader (the strategy interface) and its two implementations — the
// provider-spec fetch pipes (module-tree.md §6): FileSpecLoader.Load (this ticket) and
// HTTPSpecLoader.Load (ticket 12, appended to this same file). io: none for the file strategy —
// a pure pipe, no validation beyond what the parser itself enforces, not unit-tested (its two
// failure branches are component scenarios 5 & 6).
package validate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/lerenn/asyncapi-codegen/pkg/asyncapi/parser"
	"gopkg.in/yaml.v3"
)

// SpecLoader is the late-bound strategy BuildSpecLoader (logic.go) selects between
// (module-tree.md §6, §3 "one integration per I/O module"): FileSpecLoader when
// provider.spec_path is set, HTTPSpecLoader when provider.spec_url is set.
type SpecLoader interface {
	Load() (ProviderSpec, error)
}

// FileSpecLoader reads the provider AsyncAPI 3.0 spec from a local file (contracts.md
// §FileSpecLoader.Load). Deps = — (OS filesystem encapsulated in the zero-value-shaped object;
// its one field is the path captured at construction, not an external dependency).
type FileSpecLoader struct {
	path string
}

// NewFileSpecLoader constructs a FileSpecLoader for path (captured at construction — Load takes
// no data argument, contracts.md §FileSpecLoader.Load "Input: void").
func NewFileSpecLoader(path string) FileSpecLoader {
	return FileSpecLoader{path: path}
}

// Load is the pipe (contracts.md §FileSpecLoader.Load): reads the spec file, applies
// InlineServerRefs (ticket 10, D2 — the parser doesn't resolve the root `#/servers/` ref), then
// hands the (re-encoded) tree to the pinned `lerenn/asyncapi-codegen` parser's
// `parser.FromYAML(...).Process()` (TASK.md "Ограничения").
//
// Antecedent: —.
// Consequent: success -> ProviderSpec; failure: file missing/unreadable -> ErrFileNotFound
// (FILE_NOT_FOUND); doesn't parse as AsyncAPI 3.0 -> ErrParseError (PARSE_ERROR).
func (l FileSpecLoader) Load() (ProviderSpec, error) {
	data, err := os.ReadFile(l.path)
	if err != nil {
		return ProviderSpec{}, fmt.Errorf("%w: read %s: %v", ErrFileNotFound, l.path, err)
	}

	spec, err := parseProviderSpec(data, l.path)
	if err != nil {
		return ProviderSpec{}, err
	}
	return spec, nil
}

// providerTokenEnv is the env var HTTPSpecLoader.Load reads the bearer token from (TASK.md §6:
// "ТОЛЬКО из env, никогда из конфига/git"). Never logged, never placed in a domain type.
const providerTokenEnv = "PINOUT_PROVIDER_TOKEN"

// HTTPSpecLoader fetches the provider AsyncAPI 3.0 spec over HTTP (contracts.md
// §HTTPSpecLoader.Load). Deps = *http.Client (encapsulated); URL + timeout captured at
// construction — Load takes no data argument ("Input: void").
type HTTPSpecLoader struct {
	url     string
	timeout time.Duration
	client  *http.Client
}

// NewHTTPSpecLoader constructs an HTTPSpecLoader for url, bounded by timeout, using client for
// the underlying request (contracts.md §BuildSpecLoader — timeout is late-bound from the
// already-validated Config, never a constant).
func NewHTTPSpecLoader(url string, timeout time.Duration, client *http.Client) HTTPSpecLoader {
	return HTTPSpecLoader{url: url, timeout: timeout, client: client}
}

// Load is the pipe (contracts.md §HTTPSpecLoader.Load): GETs l.url with an Authorization: Bearer
// $PINOUT_PROVIDER_TOKEN header when that env var is set, bounded by l.timeout via context; on a
// 2xx response, hands the body through the shared parseProviderSpec pipe (decode ->
// InlineServerRefs -> re-encode -> parser.FromYAML(...).Process()).
//
// Antecedent: —.
// Consequent: success -> ProviderSpec; failure: exceeds l.timeout -> ErrTimeoutError
// (TIMEOUT_ERROR); unreachable/non-2xx -> ErrHTTPError (HTTP_ERROR); body doesn't parse ->
// ErrParseError (PARSE_ERROR).
func (l HTTPSpecLoader) Load() (ProviderSpec, error) {
	ctx, cancel := context.WithTimeout(context.Background(), l.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.url, nil)
	if err != nil {
		return ProviderSpec{}, fmt.Errorf("%w: build request for %s: %v", ErrHTTPError, l.url, err)
	}
	if token := os.Getenv(providerTokenEnv); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := l.client.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return ProviderSpec{}, fmt.Errorf("%w: fetch %s: exceeded %s", ErrTimeoutError, l.url, l.timeout)
		}
		return ProviderSpec{}, fmt.Errorf("%w: fetch %s: %v", ErrHTTPError, l.url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ProviderSpec{}, fmt.Errorf("%w: fetch %s: status %d", ErrHTTPError, l.url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return ProviderSpec{}, fmt.Errorf("%w: read %s: exceeded %s", ErrTimeoutError, l.url, l.timeout)
		}
		return ProviderSpec{}, fmt.Errorf("%w: read %s: %v", ErrHTTPError, l.url, err)
	}

	spec, err := parseProviderSpec(data, l.url)
	if err != nil {
		return ProviderSpec{}, err
	}
	return spec, nil
}

// parseProviderSpec is the shared antecedent-free parse pipe both FileSpecLoader.Load and
// HTTPSpecLoader.Load (ticket 12) drive once fetch-specific bytes are in hand: decode ->
// InlineServerRefs -> re-encode -> parser.FromYAML(...).Process(). Any decode/re-encode/parse/
// process failure is ErrParseError (PARSE_ERROR) — the spec body doesn't parse as AsyncAPI 3.0.
func parseProviderSpec(data []byte, subject string) (ProviderSpec, error) {
	var tree YAMLTree
	if err := yaml.Unmarshal(data, &tree); err != nil {
		return ProviderSpec{}, fmt.Errorf("%w: decode %s: %v", ErrParseError, subject, err)
	}

	tree = InlineServerRefs(tree)

	resolved, err := yaml.Marshal(tree)
	if err != nil {
		return ProviderSpec{}, fmt.Errorf("%w: re-encode %s: %v", ErrParseError, subject, err)
	}

	doc, err := parser.FromYAML(parser.FromYAMLParams{Data: resolved})
	if err != nil {
		return ProviderSpec{}, fmt.Errorf("%w: parse %s: %v", ErrParseError, subject, err)
	}
	if err := doc.Process(); err != nil {
		return ProviderSpec{}, fmt.Errorf("%w: process %s: %v", ErrParseError, subject, err)
	}

	return ProviderSpec{Doc: doc}, nil
}
