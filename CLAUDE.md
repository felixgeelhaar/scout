# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

scout is a Gin-like browser automation library for Go using a pure Chrome DevTools Protocol (CDP) implementation over WebSocket. No rod, no chromedp. It has two API layers: a core `browse` package (Engine/Context/Group/HandlerFunc for developers) and an `agent` package (Session-based, structured-output API for AI agents). Includes an MCP server binary at `cmd/scout`.

## Commands

```bash
# Build & verify
go build ./...
go vet ./...
golangci-lint run --timeout 2m . ./cmd/... ./middleware/... ./internal/...  # excludes examples/

# Tests (Chrome required for integration tests)
go test -short ./...                              # unit tests only, no Chrome
go test ./...                                     # all tests (unit + integration)
go test -run TestIntegrationClick ./...            # single test
go test -run TestIntegration -timeout 120s ./...   # all integration tests
go test -v -race -timeout 300s ./agent/...         # agent package with race detector

# Coverage
make cover-check   # runs tests + coverctl policy enforcement

# Pre-commit hook (gofmt, vet, lint, unit tests, coverctl, nox baseline)
make hooks         # install
bash scripts/pre-commit.sh  # run manually
```

## Architecture

**Two API layers, one CDP engine:**

The root `browse` package follows Gin's patterns — `Engine` manages browser lifecycle, `Context` carries page state through a `HandlerFunc` middleware chain, `Group` organizes tasks with shared middleware, `Selection`/`SelectionAll` wrap DOM elements. The `agent` package wraps all of this into a single `Session` type with structured JSON-serializable responses, auto-wait, content distillation, and mutex-protected concurrency safety.

**CDP data flow:**

`Page.call(method, params)` → `Conn.CallSessionCtx(ctx, sessionID, method, params)` — every CDP command is scoped to a session ID and carries a `context.Context` for cancellation. Events flow back through `Conn.dispatchEvent` which filters by `sessionID` before invoking handlers. `Page.Close()` cancels its context, removes all session-scoped event handlers, and closes the CDP target.

**Key internal contracts:**

- `Page.getRootNodeID()` caches the DOM document root node ID. It is invalidated when `Navigate()` is called (sets `rootNodeID = 0`). This halves CDP round-trips for `QuerySelector`/`QuerySelectorAll`.
- `Page.Navigate()` validates URLs via `URLValidator` — blocks non-http(s) schemes and private IPs by default. Tests must use `WithAllowPrivateIPs(true)`.
- Resilience middleware (Retry, Timeout, CircuitBreaker, Bulkhead) uses `c.SaveIndex()`/`c.RestoreIndex()` to replay the downstream handler chain. `RestoreIndex` clears `errors` and `aborted` but preserves `keys` — data set by prior handlers survives retries.
- `agent.Session` holds a `sync.Mutex` and locks on every public method. Internal helpers (`ensurePage`, `observeInternal`, `pageResult`, `discoverFormInternal`) are called with the lock held — they must not re-lock.
- The `internal/wait` package provides the polling implementation. `Page.WaitLoad()` and `Page.WaitForSelector()` delegate to `wait.ForLoad()` and `wait.ForSelector()`.
- MCP eval tool is gated behind `SCOUT_ENABLE_EVAL=1` env var due to arbitrary code execution risk.
- MCP server uses lazy session creation — browser starts on first tool use, not at startup. `configure` tool changes settings without restart.
- Playwright-style selectors (`:text('...')`, `:has-text('...')`) are translated to JS text-content lookup via `agent/selector.go`.
- `annotated_screenshot` returns element list only by default (no base64 image). Set `include_image: true` for the image.
- Action replay: `recordAction()` is called inside Navigate/Click/Type when `s.recording != nil`. Playbooks validate expected outcomes.
- Multi-tab: `tabManager` tracks named pages. Default page becomes "default" tab when `OpenTab` is first called.
- DOM diff classification: `classifyDiff()` categorizes mutations as modal_appeared, form_error, notification, loading_complete, etc.
- Action cost: `estimateLinkCost`/`estimateButtonCost` tag elements as high/medium/low in Observe responses.

**Screenshot compression:** `ScreenshotWithOptions` with `MaxSize` set progressively re-captures as JPEG with lower quality (80→60→40→20) and smaller scale (1.0→0.75→0.5→0.25) until the image fits under the byte limit. `agent.Session.Screenshot()` defaults to a 5MB limit.

## Key Dependencies

| Package | Role |
|---------|------|
| `felixgeelhaar/bolt` | Logger/Recovery middleware (zero-alloc structured logging) |
| `felixgeelhaar/fortify` | Retry, Timeout, CircuitBreaker, RateLimit, Bulkhead middleware |
| `felixgeelhaar/statekit` | Task lifecycle state machine (pending→running→success/failed/aborted) |
| `felixgeelhaar/mcp-go` | MCP server framework for `cmd/scout` |
| `gorilla/websocket` | CDP WebSocket transport |

## Lint Configuration

golangci-lint v2 config (`.golangci.yml`): `tests: false` excludes test files from linting. The `examples/` directory is excluded via `exclude-dirs`. Lint targets must be explicit: `. ./cmd/... ./middleware/... ./internal/...` (not `./...` which includes examples).
