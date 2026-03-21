package agent

import (
	"encoding/json"
	"strings"
	"sync"
)

// networkState tracks captured network requests for a session.
type networkState struct {
	mu       sync.Mutex
	enabled  bool
	patterns []string
	requests []NetworkCapture
	pending  map[string]*NetworkCapture // requestId -> partial
	unsub    []func()
}

// EnableNetworkCapture starts capturing XHR/fetch responses matching the given URL patterns.
// Empty patterns captures all requests. Patterns are matched as substrings.
func (s *Session) EnableNetworkCapture(patterns ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensurePage(); err != nil {
		return err
	}

	if s.network == nil {
		s.network = &networkState{
			pending: make(map[string]*NetworkCapture),
		}
	}

	s.network.patterns = patterns
	s.network.enabled = true

	_, _ = s.page.Call("Network.enable", nil)

	// Subscribe to network events
	unsub1 := s.page.OnSession("Network.requestWillBeSent", func(params map[string]any) {
		s.onRequestWillBeSent(params)
	})
	unsub2 := s.page.OnSession("Network.responseReceived", func(params map[string]any) {
		s.onResponseReceived(params)
	})
	unsub3 := s.page.OnSession("Network.loadingFinished", func(params map[string]any) {
		s.onLoadingFinished(params)
	})

	s.network.unsub = append(s.network.unsub, unsub1, unsub2, unsub3)
	return nil
}

// DisableNetworkCapture stops capturing network requests.
func (s *Session) DisableNetworkCapture() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.network == nil {
		return
	}
	for _, fn := range s.network.unsub {
		fn()
	}
	s.network.enabled = false
	s.network.unsub = nil
}

// CapturedRequests returns captured network requests, optionally filtered by URL pattern.
func (s *Session) CapturedRequests(pattern string) []NetworkCapture {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.network == nil {
		return nil
	}

	s.network.mu.Lock()
	defer s.network.mu.Unlock()

	if pattern == "" {
		result := make([]NetworkCapture, len(s.network.requests))
		copy(result, s.network.requests)
		return result
	}

	var filtered []NetworkCapture
	for _, r := range s.network.requests {
		if strings.Contains(r.URL, pattern) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// ClearCapturedRequests clears all captured requests.
func (s *Session) ClearCapturedRequests() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.network != nil {
		s.network.mu.Lock()
		s.network.requests = nil
		s.network.mu.Unlock()
	}
}

func (s *Session) matchesNetworkPattern(url string) bool {
	if s.network == nil || !s.network.enabled {
		return false
	}
	if len(s.network.patterns) == 0 {
		return true
	}
	for _, p := range s.network.patterns {
		if strings.Contains(url, p) {
			return true
		}
	}
	return false
}

func (s *Session) onRequestWillBeSent(params map[string]any) {
	req, _ := params["request"].(map[string]any)
	if req == nil {
		return
	}
	reqURL, _ := req["url"].(string)
	if !s.matchesNetworkPattern(reqURL) {
		return
	}

	reqID, _ := params["requestId"].(string)
	method, _ := req["method"].(string)
	headers := extractStringMap(req, "headers")

	capture := &NetworkCapture{
		URL:            reqURL,
		Method:         method,
		RequestHeaders: headers,
	}

	s.network.mu.Lock()
	s.network.pending[reqID] = capture
	s.network.mu.Unlock()
}

func (s *Session) onResponseReceived(params map[string]any) {
	reqID, _ := params["requestId"].(string)

	s.network.mu.Lock()
	capture, ok := s.network.pending[reqID]
	s.network.mu.Unlock()

	if !ok {
		return
	}

	resp, _ := params["response"].(map[string]any)
	if resp != nil {
		if status, ok := resp["status"].(float64); ok {
			capture.Status = int(status)
		}
		capture.MimeType, _ = resp["mimeType"].(string)
		capture.ResponseHeaders = extractStringMap(resp, "headers")
	}
}

func (s *Session) onLoadingFinished(params map[string]any) {
	reqID, _ := params["requestId"].(string)

	s.network.mu.Lock()
	capture, ok := s.network.pending[reqID]
	if !ok {
		s.network.mu.Unlock()
		return
	}
	delete(s.network.pending, reqID)
	s.network.mu.Unlock()

	// Fetch response body (best-effort)
	maxBody := s.contentOpts.MaxLength
	if maxBody == 0 {
		maxBody = 4000
	}

	if s.page != nil {
		result, err := s.page.Call("Network.getResponseBody", map[string]any{
			"requestId": reqID,
		})
		if err == nil {
			var body struct {
				Body          string `json:"body"`
				Base64Encoded bool   `json:"base64Encoded"`
			}
			if err := json.Unmarshal(result, &body); err == nil && !body.Base64Encoded {
				if len(body.Body) > maxBody {
					capture.ResponseBody = body.Body[:maxBody]
					capture.Truncated = true
				} else {
					capture.ResponseBody = body.Body
				}
			}
		}
	}

	s.network.mu.Lock()
	s.network.requests = append(s.network.requests, *capture)
	s.network.mu.Unlock()
}

func extractStringMap(m map[string]any, key string) map[string]string {
	raw, ok := m[key].(map[string]any)
	if !ok {
		return nil
	}
	result := make(map[string]string, len(raw))
	for k, v := range raw {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}
	return result
}
