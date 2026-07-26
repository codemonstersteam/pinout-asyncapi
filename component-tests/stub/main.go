// Command provider-stub — real-protocol HTTP stub for slice-01-validate component scenarios
// 7 (HTTP_ERROR) and 8 (TIMEOUT_ERROR). It is a STUB, not an in-code mock (Fowler): a real
// HTTP server, reached by the SUT over the real protocol via the compose service name
// "provider-stub". Two routes, one per adapter branch under test:
//
//	GET /error  -> 404 Not Found (HTTPSpecLoader.Load's ErrHTTP branch; EMULATION.md S14)
//	GET /stall  -> sleeps past any reasonable settings.timeout, then answers 200
//	               (HTTPSpecLoader.Load's ErrTimeout branch; EMULATION.md S15 — fixture sets
//	               settings.timeout=1)
package main

import (
	"log"
	"net/http"
	"time"
)

// stallDelay MUST exceed the bad-timeout fixture's settings.timeout (1s) by a wide margin
// so the fetch-side deadline fires deterministically, not by a race.
const stallDelay = 5 * time.Second

func main() {
	mux := http.NewServeMux()

	// /healthz backs docker-compose.test.yml's HEALTHCHECK (stub.Dockerfile): the tester
	// container's depends_on gate waits for this to answer 200 before running any scenario,
	// so a component test never races the stub's own startup (a container having *started*
	// is not the same as its net/http listener being bound and accepting connections yet —
	// without this gate, scenario 4d/TIMEOUT_ERROR could observe a connection-refused
	// HTTP_ERROR instead, misclassifying a startup race as a wiring defect).
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/error", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	})

	mux.HandleFunc("/stall", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(stallDelay):
		case <-r.Context().Done():
			// client (SUT) gave up first — expected once its own timeout fires.
			return
		}
		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("asyncapi: 3.0.0\ninfo:\n  title: too-late\n  version: 1.0.0\nchannels: {}\noperations: {}\n"))
	})

	log.Println("provider-stub listening on :8080 (routes: /error, /stall)")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
