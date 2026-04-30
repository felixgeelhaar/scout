package agent

import (
	"sync"
	"testing"
)

func newTestSession() *Session {
	return &Session{
		contentOpts: DefaultContentOptions(),
	}
}

func TestAddHistory_Basic(t *testing.T) {
	s := newTestSession()
	s.addHistory("navigate", "", "https://example.com", "")

	if len(s.history) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(s.history))
	}
	if s.history[0].Action != "navigate" {
		t.Errorf("action: got %q, want %q", s.history[0].Action, "navigate")
	}
	if s.history[0].URL != "https://example.com" {
		t.Errorf("url: got %q", s.history[0].URL)
	}
	if s.history[0].Timestamp == "" {
		t.Error("timestamp should not be empty")
	}
}

func TestAddHistory_RingBuffer(t *testing.T) {
	s := newTestSession()
	for i := 0; i < 25; i++ {
		s.addHistory("click", "#btn", "", "")
	}
	if len(s.history) != 20 {
		t.Errorf("expected 20 entries (ring buffer), got %d", len(s.history))
	}
}

func TestAddHistory_PreservesLast20(t *testing.T) {
	s := newTestSession()
	for i := 0; i < 30; i++ {
		s.addHistory("type", "#input", "", "value-"+string(rune('a'+i)))
	}
	if len(s.history) != 20 {
		t.Fatalf("expected 20, got %d", len(s.history))
	}
	first := s.history[0]
	if first.Result != "value-"+string(rune('a'+10)) {
		t.Errorf("first entry result: got %q, want %q", first.Result, "value-"+string(rune('a'+10)))
	}
}

func TestSessionHistory_ReturnsLastN(t *testing.T) {
	s := newTestSession()
	for i := 0; i < 5; i++ {
		s.addHistory("click", "#btn", "", "")
	}

	tests := []struct {
		n    int
		want int
	}{
		{0, 0},
		{-1, 0},
		{1, 1},
		{3, 3},
		{5, 5},
		{10, 5},
	}
	for _, tt := range tests {
		got := s.SessionHistory(tt.n)
		gotLen := len(got)
		if tt.want == 0 && got != nil {
			gotLen = len(got)
		}
		if gotLen != tt.want {
			t.Errorf("SessionHistory(%d): got %d entries, want %d", tt.n, gotLen, tt.want)
		}
	}
}

func TestSessionHistory_ReturnsNilForEmpty(t *testing.T) {
	s := newTestSession()
	got := s.SessionHistory(5)
	if got != nil {
		t.Errorf("expected nil for empty history, got %v", got)
	}
}

func TestSessionHistory_ReturnsCopy(t *testing.T) {
	s := newTestSession()
	s.addHistory("click", "#a", "", "")
	s.addHistory("click", "#b", "", "")

	result := s.SessionHistory(2)
	result[0].Action = "modified"

	original := s.SessionHistory(2)
	if original[0].Action == "modified" {
		t.Error("SessionHistory should return a copy, not a reference")
	}
}

func TestAddHistory_AllFields(t *testing.T) {
	s := newTestSession()
	s.addHistory("type", "#email", "https://example.com/login", "test-user")

	entry := s.history[0]
	if entry.Action != "type" {
		t.Errorf("Action: got %q", entry.Action)
	}
	if entry.Selector != "#email" {
		t.Errorf("Selector: got %q", entry.Selector)
	}
	if entry.URL != "https://example.com/login" {
		t.Errorf("URL: got %q", entry.URL)
	}
	if entry.Result != "test-user" {
		t.Errorf("Result: got %q", entry.Result)
	}
}

func TestSessionHistory_ConcurrentAccess(t *testing.T) {
	s := newTestSession()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.SessionHistory(5)
		}()
	}
	wg.Wait()
}
