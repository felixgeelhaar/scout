# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.8.0] - 2026-03-21

### Added
- MCP structured content: `OutputSchema` on observe tool for typed responses
- MCP channels: navigate pushes page info to `scout.navigation` channel
- MCP elicitation: available via `ElicitFromContext` for interactive prompts
- MCP dynamic tool registration: `NotifyToolListChanged` after navigate
- MCP progress reporting: navigate reports launch/load/done steps
- MCP tool annotations: `ReadOnly`/`OpenWorld`/`ClosedWorld`/`Idempotent` on all 46 tools
- Interaction tools: hover, double_click, right_click, select_option, scroll_to, scroll_by, focus, drag_drop
- Multi-tab coordination: open_tab, switch_tab, close_tab, list_tabs
- DOM diff classification: modal_appeared, form_error, notification, loading_complete, content_loaded
- Shadow DOM traversal: `QuerySelectorPiercing` crosses shadow boundaries
- Action cost estimation: links/buttons tagged high/medium/low in observe responses
- Action replay: start_recording, stop_recording, save_playbook, replay_playbook
- Playwright :text() selector support (translates to JS text-content lookup)
- Runtime configure tool (switch headless/visible without restart)
- Lazy session creation (browser starts on first tool use)

### Fixed
- Annotated screenshot no longer returns 147KB base64 by default (element list only)

### Changed
- CLI defaults to visible browser (--headless to hide)
- Upgraded mcp-go from v1.7.0 to v1.9.0

## [0.1.0] - 2026-03-21

### Added
- Gin-like browser automation API: Engine, Context, Group, HandlerFunc, Selection
- Pure CDP implementation over WebSocket (no rod/chromedp dependency)
- Agent package with structured JSON output for AI agents
- 29 MCP tools via `scout mcp serve`
- Full CLI: `scout navigate`, `observe`, `markdown`, `screenshot`, `pdf`, `extract`, `eval`, `form discover`, `frameworks`
- Middleware: Stealth, Retry, Timeout, CircuitBreaker, RateLimit, Bulkhead, Auth, Network
- Content distillation: Markdown, ReadableText, AccessibilityTree
- DOM diff tracking between observations
- Network request/response capture
- Semantic form filling (auto-matches labels to inputs)
- Token-budget-aware extraction
- Visual grounding (annotated screenshots with numbered labels)
- Persistent browser profiles (cookie/localStorage serialization)
- Screenshot auto-compression for LLM contexts (MaxSize enforcement)
- Video recording (screencast → MP4/GIF via ffmpeg)
- PDF generation
- Remote CDP connection support (Browserbase, Steel, self-hosted)
- Framework detection and state extraction (React, Vue, Angular, Svelte, Next.js, Nuxt, Remix, SvelteKit, Gatsby, Alpine, HTMX, Stimulus, Lit, Preact, Ember, Qwik, Astro, SolidJS)
- URL validation (SSRF protection)
- Task lifecycle state machine via statekit
- Structured logging via bolt
- GoReleaser + Homebrew distribution
