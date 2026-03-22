package agent

import (
	"encoding/json"
	"testing"
)

func TestNewTabManager(t *testing.T) {
	tm := newTabManager()
	if tm == nil {
		t.Fatal("newTabManager returned nil")
	}
	if tm.tabs == nil {
		t.Error("tabs map should be initialized")
	}
	if len(tm.tabs) != 0 {
		t.Errorf("tabs should be empty, got %d", len(tm.tabs))
	}
	if tm.active != "" {
		t.Errorf("active should be empty, got %q", tm.active)
	}
}

func TestTabInfo_JSON(t *testing.T) {
	ti := TabInfo{
		Name:   "main",
		URL:    "https://example.com",
		Title:  "Example",
		Active: true,
	}
	data, err := json.Marshal(ti)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got TabInfo
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got != ti {
		t.Errorf("roundtrip: got %+v, want %+v", got, ti)
	}
}

func TestTabInfo_JSON_Inactive(t *testing.T) {
	ti := TabInfo{
		Name:   "background",
		URL:    "https://other.com",
		Active: false,
	}
	data, err := json.Marshal(ti)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got TabInfo
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Active != false {
		t.Error("Active should be false")
	}
}
