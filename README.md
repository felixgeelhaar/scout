# browse-go

Gin-like browser automation for Go. No rod, no chromedp — pure CDP over WebSocket with middleware composition, grouped tasks, and an agent-optimized API.

```go
engine := browse.Default(browse.WithHeadless(true))
engine.MustLaunch()
defer engine.Close()

engine.Task("search", func(c *browse.Context) {
    c.MustNavigate("https://example.com")
    c.El("input[name=q]").MustInput("hello")
    c.El("button[type=submit]").MustClick()
    fmt.Println(c.El("#result").MustText())
})

engine.Run("search")
```

## Install

### As an MCP Server (for AI agents)

```bash
# Homebrew (macOS/Linux)
brew install felixgeelhaar/tap/browse-mcp

# Or download binary directly
curl -fsSL https://raw.githubusercontent.com/felixgeelhaar/browse-go/main/install.sh | bash

# Or go install
go install github.com/felixgeelhaar/browse-go/cmd/browse-mcp@latest
```

Then configure in your MCP client:

```bash
# Claude Code
claude mcp add browse -- browse-mcp

# Claude Desktop — add to ~/Library/Application Support/Claude/claude_desktop_config.json
# Cursor — add to ~/.cursor/mcp.json
```

```json
{
  "mcpServers": {
    "browse": {
      "command": "browse-mcp"
    }
  }
}
```

### As a Go Library

```bash
go get github.com/felixgeelhaar/browse-go
```

Requires Chrome or Chromium installed on the system (or use `WithRemoteCDP` for remote browsers).

## Core Concepts

browse-go maps Gin patterns to browser automation:

| Gin | browse-go | Purpose |
|-----|-----------|---------|
| `gin.Engine` | `browse.Engine` | Browser lifecycle, global middleware |
| `gin.Context` | `browse.Context` | Page state, actions, data passing |
| `gin.HandlerFunc` | `browse.HandlerFunc` | Middleware and task handlers |
| `c.Next()` / `c.Abort()` | Same | Middleware chain flow control |
| `c.Set()` / `c.Get()` | Same | Pass data between handlers |
| `r.Group("/path")` | `e.Group("name")` | Grouped tasks with shared middleware |

## Middleware

Built-in middleware powered by [bolt](https://github.com/felixgeelhaar/bolt) and [fortify](https://github.com/felixgeelhaar/fortify):

```go
engine := browse.Default() // Logger + Recovery

// Resilience (fortify)
engine.Use(middleware.Timeout(30 * time.Second))
engine.Use(middleware.Retry(middleware.RetryConfig{MaxAttempts: 3}))
engine.Use(middleware.CircuitBreaker(middleware.CircuitBreakerConfig{ConsecutiveFailures: 5}))
engine.Use(middleware.RateLimit(middleware.RateLimitConfig{Rate: 10}))
engine.Use(middleware.Bulkhead(middleware.BulkheadConfig{MaxConcurrent: 5}))

// Anti-detection
engine.Use(middleware.Stealth()) // navigator.webdriver, plugins, WebGL, etc.

// Auth
engine.Use(middleware.BearerAuth("token"))
engine.Use(middleware.BasicAuth("user", "pass"))
engine.Use(middleware.CookieAuth(browse.Cookie{Name: "session", Value: "abc"}))

// Utilities
engine.Use(middleware.ScreenshotOnError("./errors"))
engine.Use(middleware.SlowMotion(500 * time.Millisecond))
engine.Use(middleware.Viewport(1920, 1080))
engine.Use(middleware.BlockResources("Image", "Font", "Stylesheet"))
```

## Agent Package

The `agent` package provides a high-level, session-based API optimized for AI agents. All responses are structured JSON, content is auto-truncated for LLM context windows, and every operation auto-waits.

```go
session, _ := agent.NewSession(agent.SessionConfig{Headless: true})
defer session.Close()

// Navigate and observe
session.Navigate("https://example.com")
obs, _ := session.Observe()       // structured: links, inputs, buttons, text
md, _ := session.Markdown()       // compact markdown, not raw HTML
tree, _ := session.AccessibilityTree() // semantic element tree

// DOM diff — only see what changed (saves 50-80% tokens)
session.Click("#submit")
_, diff, _ := session.ObserveDiff()
// diff.Added: [{Tag:"div", ID:"success", Text:"Saved!"}]

// Semantic form filling — no CSS selectors needed
session.FillFormSemantic(map[string]string{
    "Email":    "user@example.com",
    "Password": "secret",
})

// Network API capture — read XHR responses directly
session.EnableNetworkCapture("/api/")
session.Navigate("https://app.example.com")
captured := session.CapturedRequests("/api/users")
// [{URL:"/api/users", Status:200, ResponseBody:"{...}"}]

// Token-budget-aware observation
obs, _ = session.ObserveWithBudget(500) // fits in ~500 tokens

// Persistent sessions
session.SaveProfile("session.json")  // cookies + localStorage
session.LoadProfile("session.json")  // restore on next run

// Screenshots auto-compressed for LLM contexts (default 5MB limit)
data, _ := session.Screenshot()
```

## MCP Server

Single-binary MCP server — no Node.js, no Python:

```bash
go build -o browse-mcp ./cmd/browse-mcp
```

21 tools: `navigate`, `observe`, `observe_diff`, `observe_with_budget`, `click`, `type`, `fill_form`, `fill_form_semantic`, `discover_form`, `extract`, `extract_all`, `extract_table`, `screenshot`, `pdf`, `markdown`, `readable_text`, `accessibility_tree`, `has_element`, `wait_for`, `enable_network_capture`, `network_requests`.

```json
{
  "mcpServers": {
    "browse": {
      "command": "./browse-mcp"
    }
  }
}
```

## Options

```go
browse.New(
    browse.WithHeadless(true),
    browse.WithTimeout(30 * time.Second),
    browse.WithViewport(1280, 720),
    browse.WithSlowMotion(100 * time.Millisecond),
    browse.WithUserAgent("custom-agent/1.0"),
    browse.WithProxy("http://proxy:8080"),
    browse.WithRemoteCDP("ws://browserbase.example.com/connect/..."),
    browse.WithPoolSize(5),           // concurrent task execution
    browse.WithAllowPrivateIPs(true), // for testing with localhost
)
```

## Content Distillation

Pages return megabytes of HTML. browse-go provides 5 levels of content extraction:

| Method | Size | Best for |
|--------|------|----------|
| `HTML()` | 5-20MB | Never use for agents |
| `Observe()` | ~2-5KB | Deciding what to click/fill |
| `Markdown()` | ~2-8KB | Reading page content |
| `ReadableText()` | ~1-4KB | Main article/body text only |
| `AccessibilityTree()` | ~1-4KB | Compact semantic DOM |

All methods respect configurable limits:

```go
session.SetContentOptions(agent.ContentOptions{
    MaxLength:          4000,
    MaxLinks:           25,
    MaxScreenshotBytes: 5 * 1024 * 1024,
})
```

## Architecture

```
browse-go/
├── browse.go, engine.go, context.go    # Gin-like API
├── page.go, selection.go               # CDP page & element interaction
├── recorder.go                        # Video recording (screencast → MP4/GIF)
├── lifecycle.go                       # statekit task state machine
├── middleware.go                      # bolt logger + recovery
├── middleware/                        # stealth, fortify resilience, auth, network
├── agent/                             # AI agent API: session, observe, diff, forms, network, profiles
├── internal/cdp/                      # WebSocket CDP client (context-aware, session-scoped)
├── internal/launcher/                 # Chrome process management
├── internal/wait/                     # Auto-wait utilities
├── cmd/browse-mcp/                    # MCP server binary (21 tools)
└── examples/                          # login, scrape, demo
```

## License

MIT
