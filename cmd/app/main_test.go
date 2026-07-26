package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// version — стабильная команда, exit 0 + непустой вывод. Дублирует контракт smoke.
func TestVersionCmd(t *testing.T) {
	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "version") {
		t.Fatalf("ожидали строку версии, получили %q", out.String())
	}
}

// validate <config.yaml> — the live wiring's egress adapter must surface CONFIG_ERROR (exit 2)
// for a config path that doesn't exist (ConfigStore.Load's failure branch), with no report
// printed to stdout (use-case.md Minimal guarantee) and the error logged to stderr. This is a
// thin structural check on the wiring (ticket-23); the full pipe's business-logic scenarios are
// proven by internal/validate's own unit tests and by the component-test suite, not here.
func TestValidateCmd_ConfigFileNotFound(t *testing.T) {
	root := newRootCmd()
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	root.SetArgs([]string{"validate", "does-not-exist.yaml"})
	err := root.Execute()
	if err == nil {
		t.Fatal("ожидали exitError, получили nil")
	}
	var ee *exitError
	if !errors.As(err, &ee) || ee.code != 2 {
		t.Fatalf("ожидали exit code 2, получили %v", err)
	}
	if strings.TrimSpace(out.String()) != "" {
		t.Fatalf("ожидали пустой stdout на CONFIG_ERROR, получили %q", out.String())
	}
	if errb.Len() == 0 {
		t.Fatal("ожидали диагностику в stderr, получили пусто")
	}
}
