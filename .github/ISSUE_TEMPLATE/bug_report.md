---
name: Bug Report
about: Report a bug in scout / browse-go
title: ''
labels: bug
assignees: ''
---

## Description

A clear description of what the bug is.

## Steps to Reproduce

```go
// Minimal code to reproduce the issue
engine := browse.New(browse.WithHeadless(true))
engine.MustLaunch()
defer engine.Close()
// ...
```

## Expected Behavior

What you expected to happen.

## Actual Behavior

What actually happened. Include error messages or output.

## Environment

- OS: [e.g. macOS 15, Ubuntu 24.04]
- Go version: [e.g. 1.23]
- Chrome version: [e.g. 131]
- Scout version: [output of `scout version`]
- Usage: [library / CLI / MCP server]
