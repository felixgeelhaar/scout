# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in Scout (scout), please report it responsibly.

**Do not open a public GitHub issue for security vulnerabilities.**

Instead, please email security concerns to the maintainer or use [GitHub Security Advisories](https://github.com/klarlabs-studio/scout/security/advisories/new).

## Security Model

Scout launches and controls a Chrome browser process. Be aware of these security considerations:

### URL Validation
- Navigation and **subresource** requests (XHR, fetch, images, scripts, WebSockets) are re-validated against the URL policy
- Non-http(s) schemes are blocked (`file://`, `javascript:`; `data:` is blocked for documents, allowed for subresources like images). Browser-internal schemes (`chrome:`, `chrome-extension:`, `devtools:`) are allowed so Chrome itself can load.
- Private/loopback/link-local IPs (including `169.254.169.254`) are blocked by default (opt-in via `WithAllowPrivateIPs` / `configure allow_private_ips`)
- `data:` and `blob:` subresources and `about:blank` are allowed so pages can still render

### MCP Server
- Default tool surface is a curated set (~22 tools). Pass `--advanced` or `SCOUT_MCP_ADVANCED=1` for the full set.
- The `eval` tool (arbitrary JavaScript execution) is not registered unless `SCOUT_ENABLE_EVAL=1`
- The same flag gates eval in the chat UI (`scout ui serve`); the CLI `scout eval` command is an explicit one-shot the operator typed
- Observe/extract/markdown/network results are wrapped with `_untrusted_page_content` so models treat page text as data
- All tool inputs are validated via typed structs

### File Operations
- Agent-supplied write paths (`screenshot` `output_path`, playbooks, traces, recordings) reject path traversal, credential locations, and system directories
- Relative capture paths resolve under the OS temp dir and cannot escape it
- Files are written with `0600` permissions
- Temp directories for recordings use OS-provided secure temp paths
- `upload_file` refuses well-known credential files and directories (`.ssh`, `.aws`, `.env`, `credentials.json`, …)

### Browser Process
- Chrome is launched with security-hardening flags
- Stealth middleware patches automation detection markers
- WebSocket CDP connection is localhost-only by default

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest  | Yes       |

## Dependencies

We monitor dependencies via `nox scan` for known vulnerabilities. Run `make nox` to check locally.
