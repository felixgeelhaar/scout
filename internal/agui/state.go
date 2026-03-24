package agui

import (
	"encoding/json"
)

// BrowserState is the shared state object synced to the CopilotKit frontend.
type BrowserState struct {
	URL        string        `json:"url"`
	Title      string        `json:"title"`
	Screenshot string        `json:"screenshot,omitempty"` // base64 JPEG
	Elements   []ElementInfo `json:"elements,omitempty"`
	ReadyScore int           `json:"readyScore"`
	ActiveTool string        `json:"activeTool,omitempty"`
	TabCount   int           `json:"tabCount"`
}

// ElementInfo is a simplified element descriptor for the shared state.
type ElementInfo struct {
	Tag      string `json:"tag"`
	Text     string `json:"text,omitempty"`
	Selector string `json:"selector,omitempty"`
	Type     string `json:"type,omitempty"`
}

// Diff computes JSON Patch operations between the old and new state.
// Only changed fields emit patch operations to minimize bandwidth.
func Diff(old, new *BrowserState) []PatchOp {
	var ops []PatchOp

	if old.URL != new.URL {
		ops = append(ops, PatchOp{Op: "replace", Path: "/url", Value: new.URL})
	}
	if old.Title != new.Title {
		ops = append(ops, PatchOp{Op: "replace", Path: "/title", Value: new.Title})
	}
	if old.Screenshot != new.Screenshot && new.Screenshot != "" {
		ops = append(ops, PatchOp{Op: "replace", Path: "/screenshot", Value: new.Screenshot})
	}
	if old.ReadyScore != new.ReadyScore {
		ops = append(ops, PatchOp{Op: "replace", Path: "/readyScore", Value: new.ReadyScore})
	}
	if old.ActiveTool != new.ActiveTool {
		ops = append(ops, PatchOp{Op: "replace", Path: "/activeTool", Value: new.ActiveTool})
	}
	if old.TabCount != new.TabCount {
		ops = append(ops, PatchOp{Op: "replace", Path: "/tabCount", Value: new.TabCount})
	}

	// For elements, compare serialized forms (simple but effective)
	oldElems, _ := json.Marshal(old.Elements)
	newElems, _ := json.Marshal(new.Elements)
	if string(oldElems) != string(newElems) {
		ops = append(ops, PatchOp{Op: "replace", Path: "/elements", Value: new.Elements})
	}

	return ops
}
