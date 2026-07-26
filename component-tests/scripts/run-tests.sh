#!/usr/bin/env bash
# Запуск компонентных тестов template-go-cli ВНУТРИ Docker (изоляция; не `go test` с хоста).
# tool стейджит бинарь в общий том и выходит; tester гоняет godog против него.
set -euo pipefail
cd "$(dirname "$0")/.."
CF=(-f docker-compose.test.yml)
cleanup() { docker compose "${CF[@]}" down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT
echo "==> building images..."; docker compose "${CF[@]}" build
# Two phases, not one `up --abort-on-container-exit --exit-code-from tester`: bringing every
# service up in a single `up` races docker compose's own container/network bookkeeping for
# provider-stub against tester's first DNS lookup of it — reproducibly (not a rare flake)
# "no such host" against Docker's embedded resolver (127.0.0.11) for the tester container's
# entire lifetime once hit, no retry/backoff window recovers it. Bringing provider-stub (+ the
# one-shot tool stager) up FIRST, then starting tester as its own `run` afterwards, guarantees
# provider-stub is fully network-registered before tester ever resolves its name — this is also
# what a plain `docker compose run tester` against an already-up stack does reliably, verified
# manually while diagnosing this exact ordering issue.
echo "==> starting SUT (tool) + provider-stub..."; docker compose "${CF[@]}" up -d provider-stub tool
echo "==> running tests..."; docker compose "${CF[@]}" run --rm tester
