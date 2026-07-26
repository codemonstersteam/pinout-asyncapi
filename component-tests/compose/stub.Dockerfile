# Real-protocol HTTP stub for the provider's spec_url (scenarios 7 HTTP_ERROR / 8 TIMEOUT_ERROR).
# A genuine long-running HTTP server (never an in-code mock), reached by the SUT over the
# real protocol via the compose service name "provider-stub". Standalone module — no
# dependency on the SUT's go.mod.
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY component-tests/stub/go.mod ./
COPY component-tests/stub/main.go ./
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -ldflags '-s -w' -o /out/provider-stub .

FROM alpine:3.21
COPY --from=build /out/provider-stub /provider-stub
EXPOSE 8080
# Diagnostic only (visible via `docker compose ps`) — NOT wired into any depends_on gate.
# tester waits for actual HTTP readiness itself, in-process (steps/main_test.go's
# waitForProviderStub): a health-gated depends_on here would delay this container's own
# "started" signal enough to race docker compose's --abort-on-container-exit against the
# one-shot `tool` container's near-instant clean exit (see main_test.go's comment).
HEALTHCHECK --interval=1s --timeout=1s --start-period=0s --retries=5 \
  CMD wget -q -O /dev/null http://localhost:8080/healthz || exit 1
ENTRYPOINT ["/provider-stub"]
