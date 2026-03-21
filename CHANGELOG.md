# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
