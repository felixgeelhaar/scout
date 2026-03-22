package agent

import (
	"fmt"
	"testing"
	"time"
)

func TestJsonQuote(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", `"hello"`},
		{"", `""`},
		{`with "quotes"`, `"with \"quotes\""`},
		{"with\nnewline", `"with\nnewline"`},
		{"with\ttab", `"with\ttab"`},
		{`back\slash`, `"back\\slash"`},
		{"unicode: 日本語", `"unicode: 日本語"`},
		{`<script>alert('xss')</script>`, `"\u003cscript\u003ealert('xss')\u003c/script\u003e"`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := jsonQuote(tt.input)
			if got != tt.want {
				t.Errorf("jsonQuote(%q) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

func TestDefaultContentOptions_Values(t *testing.T) {
	opts := DefaultContentOptions()
	if opts.MaxLength != 4000 {
		t.Errorf("MaxLength: got %d", opts.MaxLength)
	}
	if opts.MaxLinks != 25 {
		t.Errorf("MaxLinks: got %d", opts.MaxLinks)
	}
	if opts.MaxInputs != 20 {
		t.Errorf("MaxInputs: got %d", opts.MaxInputs)
	}
	if opts.MaxButtons != 15 {
		t.Errorf("MaxButtons: got %d", opts.MaxButtons)
	}
	if opts.MaxItems != 50 {
		t.Errorf("MaxItems: got %d", opts.MaxItems)
	}
	if opts.MaxRows != 100 {
		t.Errorf("MaxRows: got %d", opts.MaxRows)
	}
	if opts.MaxScreenshotBytes != 5*1024*1024 {
		t.Errorf("MaxScreenshotBytes: got %d", opts.MaxScreenshotBytes)
	}
}

func TestSetContentOptions(t *testing.T) {
	s := newTestSession()
	custom := ContentOptions{
		MaxLength:  1000,
		MaxLinks:   5,
		MaxInputs:  3,
		MaxButtons: 2,
		MaxItems:   10,
		MaxRows:    20,
	}
	s.SetContentOptions(custom)
	if s.contentOpts.MaxLength != 1000 {
		t.Errorf("MaxLength: got %d", s.contentOpts.MaxLength)
	}
	if s.contentOpts.MaxLinks != 5 {
		t.Errorf("MaxLinks: got %d", s.contentOpts.MaxLinks)
	}
}

func TestTraceBeforeAction_NotTracing(t *testing.T) {
	s := newTestSession()
	start, before := s.traceBeforeAction("click", "#btn", "", "")
	if !start.IsZero() {
		t.Error("start should be zero when not tracing")
	}
	if before != nil {
		t.Error("before should be nil when not tracing")
	}
}

func TestTraceAfterAction_NotTracing(t *testing.T) {
	s := newTestSession()
	s.traceAfterAction(time.Now(), nil, "click", "#btn", "", "", nil)
	if s.trace != nil {
		t.Error("trace should remain nil when not tracing")
	}
}

func TestTraceAfterAction_Tracing(t *testing.T) {
	s := newTestSession()
	s.tracing = true
	s.trace = &traceState{startTime: time.Now()}

	start := time.Now()
	s.traceAfterAction(start, nil, "navigate", "", "", "https://example.com", nil)

	if len(s.trace.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(s.trace.events))
	}
	ev := s.trace.events[0]
	if ev.Action != "navigate" {
		t.Errorf("action: got %q", ev.Action)
	}
	if ev.URL != "https://example.com" {
		t.Errorf("url: got %q", ev.URL)
	}
	if ev.Error != "" {
		t.Errorf("error should be empty, got %q", ev.Error)
	}
}

func TestTraceAfterAction_WithError(t *testing.T) {
	s := newTestSession()
	s.tracing = true
	s.trace = &traceState{startTime: time.Now()}

	start := time.Now()
	s.traceAfterAction(start, nil, "click", "#btn", "", "", fmt.Errorf("element not found"))

	if len(s.trace.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(s.trace.events))
	}
	if s.trace.events[0].Error != "element not found" {
		t.Errorf("error: got %q", s.trace.events[0].Error)
	}
}

func TestTraceAfterAction_WithScreenshots(t *testing.T) {
	s := newTestSession()
	s.tracing = true
	s.trace = &traceState{startTime: time.Now()}

	start := time.Now()
	before := []byte("before-screenshot-data")
	s.traceAfterAction(start, before, "click", "#btn", "", "", nil)

	if len(s.trace.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(s.trace.events))
	}
	if s.trace.events[0].BeforeImg != "before-screenshot-data" {
		t.Errorf("BeforeImg: got %q", s.trace.events[0].BeforeImg)
	}
}
