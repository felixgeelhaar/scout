package main

import "go.klarlabs.de/mcp"

// curatedMCPTools is the default MCP surface (~22 tools). Everything else is
// registered only with --advanced / SCOUT_MCP_ADVANCED=1. eval is never in
// this set; it is added separately when SCOUT_ENABLE_EVAL=1.
var curatedMCPTools = []string{
	"configure",
	"status",
	"reset",
	"navigate",
	"observe",
	"observe_diff",
	"click",
	"click_text",
	"click_label",
	"type",
	"fill_form_semantic",
	"submit_form",
	"discover_form",
	"extract",
	"extract_table",
	"markdown",
	"screenshot",
	"annotated_screenshot",
	"wait_for",
	"has_element",
	"dismiss_cookies",
	"batch",
}

type mcpServeOptions struct {
	Advanced bool
	Eval     bool
}

func applyMCPToolFilter(srv *mcp.Server, opts mcpServeOptions) {
	if opts.Advanced {
		return
	}
	allow := make(map[string]struct{}, len(curatedMCPTools)+1)
	for _, n := range curatedMCPTools {
		allow[n] = struct{}{}
	}
	if opts.Eval {
		allow["eval"] = struct{}{}
	}
	for _, t := range srv.Tools() {
		if _, ok := allow[t.Name]; !ok {
			srv.RemoveTool(t.Name)
		}
	}
}

func mcpToolNames(srv *mcp.Server) []string {
	tools := srv.Tools()
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name
	}
	return names
}
