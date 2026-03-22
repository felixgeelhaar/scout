package agent

import "time"

// HistoryEntry records a single action in the session history.
type HistoryEntry struct {
	Action    string `json:"action"`
	Selector  string `json:"selector,omitempty"`
	URL       string `json:"url,omitempty"`
	Result    string `json:"result,omitempty"`
	Timestamp string `json:"timestamp"`
}

// SessionHistory returns the last N actions performed in this session.
// Provides conversation-aware context so agents don't lose track of what they've done.
func (s *Session) SessionHistory(n int) []HistoryEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	if n <= 0 || len(s.history) == 0 {
		return nil
	}
	if n > len(s.history) {
		n = len(s.history)
	}
	// Return the last N entries
	result := make([]HistoryEntry, n)
	copy(result, s.history[len(s.history)-n:])
	return result
}

// addHistory appends an entry to the session history. Caller must hold s.mu.
// Keeps at most 20 entries to avoid unbounded growth.
func (s *Session) addHistory(action, selector, url, result string) {
	entry := HistoryEntry{
		Action:    action,
		Selector:  selector,
		URL:       url,
		Result:    result,
		Timestamp: time.Now().Format("15:04:05"),
	}
	s.history = append(s.history, entry)
	if len(s.history) > 20 {
		s.history = s.history[len(s.history)-20:]
	}
}
