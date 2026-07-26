// register.go is the slice's composition root for the concrete Deps ProcessValidate (head.go,
// ticket-22) needs to run for real (module-tree.md §6, ticket-23). Wiring only — no domain logic:
// it constructs one production Clock (systemClock, D10 — the only production `time.Now()` read
// in the whole slice) and a plain *http.Client (the domain timeout, cfg.Settings.Timeout, is
// applied per-request inside HTTPSpecLoader.Load via context — never a transport-level constant
// hardcoded here, module-tree.md §3 row 10). cmd/app/main.go calls NewDeps() to obtain the value
// it hands to ProcessValidate.
package validate

import (
	"net/http"
	"time"
)

// systemClock is the slice's sole production Clock implementation (domain.go's Clock port,
// D10). The only place in the whole slice that reads the system clock directly.
type systemClock struct{}

// Now returns the current wall-clock time.
func (systemClock) Now() time.Time { return time.Now() }

// NewDeps constructs the concrete Deps for ProcessValidate: a real systemClock and a plain
// *http.Client. No provider timeout is set here at the transport level — BuildSpecLoader
// (logic.go) late-binds the real, validated cfg.Settings.Timeout into a per-request context
// inside HTTPSpecLoader.Load, so this Client stays a sane, timeout-free default.
func NewDeps() Deps {
	return Deps{
		Clock:      systemClock{},
		HTTPClient: &http.Client{},
	}
}
