package steps

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
)

var opts = godog.Options{
	Output:    colors.Colored(os.Stdout),
	Format:    "pretty",
	Randomize: -1, // детерминированный порядок
}

func init() { godog.BindCommandLineFlags("godog.", &opts) }

// TestFeatures гоняет все .feature из ../features. Раннер-контейнер запускает
// `go test -v -count=1 ./steps/...` после того, как бинарь застейджен в общий том.
func TestFeatures(t *testing.T) {
	opts.Paths = []string{"../features"}
	status := godog.TestSuite{
		Name:                 "component-tests",
		TestSuiteInitializer: InitializeTestSuite,
		ScenarioInitializer:  InitializeScenario,
		Options:              &opts,
	}.Run()
	if status != 0 {
		t.Fail()
	}
}

// InitializeTestSuite runs once, before any scenario. waitForProviderStub is a light safety
// net: component-tests/scripts/run-tests.sh brings provider-stub (+ the one-shot tool stager)
// fully up in its own `docker compose up -d` phase BEFORE starting tester as a separate
// `docker compose run`, specifically so provider-stub is already network-registered and
// answering by the time this process's own DNS resolver ever queries its name (a single
// combined `up --abort-on-container-exit` for every service was found to reproducibly, not
// rarely, leave tester's embedded-DNS (127.0.0.11) lookups of "provider-stub" failing for the
// container's entire lifetime — no retry/backoff window here recovered from it once hit, so
// this loop is deliberately just a short cushion for slow image pulls/cold starts, not the fix
// for that ordering issue).
func InitializeTestSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(waitForProviderStub)
}

func waitForProviderStub() {
	const (
		url      = "http://provider-stub:8080/healthz"
		attempts = 10
		delay    = 300 * time.Millisecond
	)
	client := &http.Client{Timeout: 500 * time.Millisecond}
	for i := 0; i < attempts; i++ {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(delay)
	}
	// Fall through silently — component scenarios 4c/4d will surface a clear failure of
	// their own (unexpected HTTP_ERROR) if provider-stub truly never came up; no need to
	// fail the whole suite here for what scenario 1's smoke/happy-path already proves is a
	// working binary.
}

// InitializeScenario регистрирует степы и lifecycle-хуки. Добавляй свои
// register*Steps(ctx) по мере роста набора.
func InitializeScenario(ctx *godog.ScenarioContext) {
	w := newWorld()
	ctx.Before(w.beforeScenario)
	w.registerCLISteps(ctx)
	w.registerValidateSteps(ctx)
}
