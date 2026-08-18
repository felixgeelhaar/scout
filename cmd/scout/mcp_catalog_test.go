package main

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"go.klarlabs.de/mcp"
)

func TestCuratedMCPTools_NoDuplicatesAndNamedBatch(t *testing.T) {
	seen := map[string]struct{}{}
	for _, n := range curatedMCPTools {
		if n == "execute_batch" {
			t.Fatal("curated set must use batch, not execute_batch")
		}
		if _, ok := seen[n]; ok {
			t.Errorf("duplicate curated tool %q", n)
		}
		seen[n] = struct{}{}
	}
	if _, ok := seen["batch"]; !ok {
		t.Fatal("curated set must include batch")
	}
	if len(curatedMCPTools) < 20 || len(curatedMCPTools) > 24 {
		t.Errorf("curated set size %d; want ~22", len(curatedMCPTools))
	}
}

func TestMCPToolRegistrationsInSource(t *testing.T) {
	src, err := os.ReadFile("mcp.go")
	if err != nil {
		t.Fatal(err)
	}
	matches := regexp.MustCompile(`srv\.Tool\("([^"]+)"\)`).FindAllStringSubmatch(string(src), -1)
	if len(matches) == 0 {
		t.Fatal("no srv.Tool registrations found")
	}
	seen := map[string]struct{}{}
	var names []string
	for _, m := range matches {
		n := m[1]
		if _, ok := seen[n]; ok {
			t.Errorf("duplicate srv.Tool(%q)", n)
		}
		seen[n] = struct{}{}
		names = append(names, n)
	}
	if _, ok := seen["execute_batch"]; ok {
		t.Error("tool is named batch, not execute_batch")
	}
	if _, ok := seen["batch"]; !ok {
		t.Error("missing batch tool")
	}
	if _, ok := seen["eval"]; !ok {
		t.Error("eval must be registered in source (gated at runtime)")
	}
	for _, n := range curatedMCPTools {
		if _, ok := seen[n]; !ok {
			t.Errorf("curated tool %q is not registered in mcp.go", n)
		}
	}
	// 88 always-on tools + eval (runtime gated) = 89 Tool() calls.
	if len(names) != 89 {
		t.Errorf("registered Tool() count = %d, want 89 (88 + eval); names=%v", len(names), names)
	}
}

func TestApplyMCPToolFilter_CuratedVsAdvanced(t *testing.T) {
	register := func(srv *mcp.Server, names ...string) {
		for _, n := range names {
			name := n
			srv.Tool(name).Description("t").Handler(func(context.Context, struct{}) (string, error) {
				return name, nil
			})
		}
	}

	t.Run("curated drops advanced tools", func(t *testing.T) {
		srv := mcp.NewServer(mcp.ServerInfo{Name: "t", Version: "0"})
		register(srv, "observe", "click", "pdf", "web_vitals", "batch")
		applyMCPToolFilter(srv, mcpServeOptions{})
		got := mcpToolNames(srv)
		if slices.Contains(got, "pdf") || slices.Contains(got, "web_vitals") {
			t.Errorf("advanced tools survived filter: %v", got)
		}
		if !slices.Contains(got, "observe") || !slices.Contains(got, "batch") {
			t.Errorf("curated tools missing: %v", got)
		}
	})

	t.Run("advanced keeps everything", func(t *testing.T) {
		srv := mcp.NewServer(mcp.ServerInfo{Name: "t", Version: "0"})
		register(srv, "observe", "pdf", "eval")
		applyMCPToolFilter(srv, mcpServeOptions{Advanced: true})
		if len(mcpToolNames(srv)) != 3 {
			t.Errorf("advanced filter should keep all, got %v", mcpToolNames(srv))
		}
	})

	t.Run("eval kept on curated only when enabled", func(t *testing.T) {
		srv := mcp.NewServer(mcp.ServerInfo{Name: "t", Version: "0"})
		register(srv, "observe", "eval")
		applyMCPToolFilter(srv, mcpServeOptions{Eval: true})
		got := mcpToolNames(srv)
		if !slices.Contains(got, "eval") {
			t.Errorf("eval should remain when enabled: %v", got)
		}

		srv2 := mcp.NewServer(mcp.ServerInfo{Name: "t", Version: "0"})
		register(srv2, "observe", "eval")
		applyMCPToolFilter(srv2, mcpServeOptions{})
		if slices.Contains(mcpToolNames(srv2), "eval") {
			t.Error("eval must be dropped when disabled")
		}
	})
}

func TestWriteCapture_RejectsTraversal(t *testing.T) {
	if _, err := writeCapture("../../../etc/cron.d/x", "screenshot", "png", []byte("x")); err == nil {
		t.Fatal("expected traversal error")
	}
	if _, err := writeCapture("/etc/passwd", "screenshot", "png", []byte("x")); err == nil {
		t.Fatal("expected system path error")
	}
}

func TestWriteCapture_ModeIs0600(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shot.png")
	got, err := writeCapture(path, "screenshot", "png", []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(got)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("perm = %o, want 0600", info.Mode().Perm())
	}
}
