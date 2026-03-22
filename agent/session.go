// Package agent provides a high-level, agent-optimized API for browser automation.
//
// Unlike the core browse package which follows Gin's middleware/handler pattern for
// human developers, the agent package is designed for AI agents and programmatic callers.
// It provides:
//   - Stateful sessions with automatic page lifecycle management
//   - Structured JSON-serializable results (not plain strings)
//   - Built-in retry and auto-wait on all operations
//   - High-level compound actions (FillForm, ExtractTable, ClickAndWait)
//   - Snapshot-based page state for agent context windows
package agent

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	browse "github.com/felixgeelhaar/scout"
)

// Session manages a stateful browser automation session for an agent.
// All methods are goroutine-safe via an internal mutex.
type Session struct {
	mu            sync.Mutex
	browser       browse.Browser
	page          *browse.Page
	timeout       time.Duration
	contentOpts   ContentOptions
	network       *networkState
	diffInstalled bool
	closed        bool
	tabs          *tabManager
	recording     *recording
	history       []HistoryEntry
}

// SessionConfig configures a new Session.
type SessionConfig struct {
	Headless        bool
	Timeout         time.Duration
	UserAgent       string
	Viewport        [2]int // [width, height], zero means default
	AllowPrivateIPs bool   // Allow navigation to private/loopback IPs
	RemoteCDP       string // WebSocket URL for remote Chrome (skips local launch)
}

// NewSession creates and launches a new browser session.
func NewSession(cfg SessionConfig) (*Session, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	opts := []browse.Option{
		browse.WithHeadless(cfg.Headless),
		browse.WithTimeout(cfg.Timeout),
	}
	if cfg.UserAgent != "" {
		opts = append(opts, browse.WithUserAgent(cfg.UserAgent))
	}
	if cfg.Viewport[0] > 0 && cfg.Viewport[1] > 0 {
		opts = append(opts, browse.WithViewport(cfg.Viewport[0], cfg.Viewport[1]))
	}
	if cfg.AllowPrivateIPs {
		opts = append(opts, browse.WithAllowPrivateIPs(true))
	}
	if cfg.RemoteCDP != "" {
		opts = append(opts, browse.WithRemoteCDP(cfg.RemoteCDP))
	}

	engine := browse.New(opts...)
	if err := engine.Launch(); err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	return &Session{
		browser:     engine,
		timeout:     cfg.Timeout,
		contentOpts: DefaultContentOptions(),
	}, nil
}

// NewSessionFromBrowser creates a session from an existing Browser implementation.
// Use this to inject a mock browser for testing or to reuse a pre-configured Engine.
func NewSessionFromBrowser(b browse.Browser, cfg SessionConfig) *Session {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &Session{
		browser:     b,
		timeout:     cfg.Timeout,
		contentOpts: DefaultContentOptions(),
	}
}

// Close shuts down the browser and releases all resources.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	if s.page != nil {
		_ = s.page.Close()
		s.page = nil
	}
	return s.browser.Close()
}

// ensurePage creates a page if none exists.
func (s *Session) ensurePage() error {
	if s.closed {
		return fmt.Errorf("session is closed")
	}
	if s.page != nil {
		return nil
	}
	page, err := s.browser.NewPage()
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}
	s.page = page
	return nil
}

// Navigate loads a URL and returns structured page info.
func (s *Session) Navigate(url string) (*PageResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Close existing page and create new one directly at target URL (skips about:blank)
	if s.page != nil {
		_ = s.page.Close()
	}
	page, err := s.browser.NewPageAt(url)
	if err != nil {
		return nil, fmt.Errorf("failed to navigate to %s: %w", url, err)
	}
	s.page = page
	s.diffInstalled = false

	// Wait for page to fully load
	if err := page.WaitLoad(); err != nil {
		// Non-fatal — page may be interactive but not fully loaded
		_ = err
	}

	s.recordAction(Action{Type: "navigate", Value: url})
	s.addHistory("navigate", "", url, "")
	return s.pageResult()
}

// Snapshot returns the current page state without performing any action.
func (s *Session) Snapshot() (*PageResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensurePage(); err != nil {
		return nil, err
	}
	return s.pageResult()
}

func (s *Session) pageResult() (*PageResult, error) {
	url, _ := s.page.URL()
	title, _ := s.page.Evaluate(`document.title`)
	titleStr, _ := title.(string)

	return &PageResult{
		URL:   url,
		Title: titleStr,
	}, nil
}

// Click clicks an element and returns the updated page state.
func (s *Session) Click(selector string) (*PageResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensurePage(); err != nil {
		return nil, err
	}

	if err := s.waitAndResolve(selector); err != nil {
		return nil, err
	}

	nodeID, err := s.querySelector(selector)
	if err != nil {
		return nil, err
	}
	sel := browse.NewSelection(s.page, nodeID, selector)
	if err := sel.Click(); err != nil {
		return nil, err
	}

	// Wait for any resulting navigation or DOM update
	_ = s.page.WaitStable(300 * time.Millisecond)

	s.recordAction(Action{Type: "click", Selector: selector})
	s.addHistory("click", selector, "", "")
	return s.pageResult()
}

// ClickAndWait clicks an element and waits for a full page navigation.
func (s *Session) ClickAndWait(selector string) (*PageResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensurePage(); err != nil {
		return nil, err
	}

	nodeID, err := s.querySelector(selector)
	if err != nil {
		return nil, err
	}
	sel := browse.NewSelection(s.page, nodeID, selector)
	if err := sel.Click(); err != nil {
		return nil, err
	}

	if err := s.page.WaitLoad(); err != nil {
		return nil, err
	}

	return s.pageResult()
}

// Type types text into an input element.
func (s *Session) Type(selector, text string) (*ElementResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensurePage(); err != nil {
		return nil, err
	}

	if err := s.waitAndResolve(selector); err != nil {
		return nil, err
	}

	nodeID, err := s.querySelector(selector)
	if err != nil {
		return nil, err
	}
	sel := browse.NewSelection(s.page, nodeID, selector)
	if err := sel.Input(text); err != nil {
		return nil, err
	}

	val, _ := sel.Value()
	s.recordAction(Action{Type: "type", Selector: selector, Value: text})
	s.addHistory("type", selector, "", text)
	return &ElementResult{
		Selector: selector,
		Value:    val,
		Action:   "typed",
	}, nil
}

// FillForm fills multiple form fields and returns their resulting values.
func (s *Session) FillForm(fields map[string]string) (*FormResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensurePage(); err != nil {
		return nil, err
	}

	result := &FormResult{
		Fields: make([]FieldResult, 0, len(fields)),
	}

	for selector, value := range fields {
		nodeID, err := s.querySelector(selector)
		if err != nil {
			result.Fields = append(result.Fields, FieldResult{
				Selector: selector,
				Error:    err.Error(),
			})
			continue
		}
		sel := browse.NewSelection(s.page, nodeID, selector)
		if err := sel.Input(value); err != nil {
			result.Fields = append(result.Fields, FieldResult{
				Selector: selector,
				Error:    err.Error(),
			})
			continue
		}
		actual, _ := sel.Value()
		result.Fields = append(result.Fields, FieldResult{
			Selector: selector,
			Value:    actual,
			Success:  true,
		})
	}

	result.Success = true
	for _, f := range result.Fields {
		if !f.Success {
			result.Success = false
			break
		}
	}

	return result, nil
}

// Extract returns the text content of an element.
func (s *Session) Extract(selector string) (*ElementResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensurePage(); err != nil {
		return nil, err
	}

	if err := s.waitAndResolve(selector); err != nil {
		return nil, err
	}

	nodeID, err := s.querySelector(selector)
	if err != nil {
		return nil, err
	}
	sel := browse.NewSelection(s.page, nodeID, selector)
	text, err := sel.Text()
	if err != nil {
		return nil, err
	}

	return &ElementResult{
		Selector: selector,
		Text:     text,
		Action:   "extracted",
	}, nil
}

// ExtractAll returns text content from all matching elements.
func (s *Session) ExtractAll(selector string) (*ExtractAllResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensurePage(); err != nil {
		return nil, err
	}

	nodeIDs, err := s.page.QuerySelectorAll(selector)
	if err != nil {
		return nil, err
	}

	maxItems := s.contentOpts.MaxItems
	if maxItems == 0 {
		maxItems = 50
	}

	items := make([]string, 0, len(nodeIDs))
	for _, nid := range nodeIDs {
		if len(items) >= maxItems {
			break
		}
		sel := browse.NewSelection(s.page, nid, selector)
		text, err := sel.Text()
		if err != nil {
			continue
		}
		items = append(items, text)
	}

	total := len(nodeIDs)
	return &ExtractAllResult{
		Selector:  selector,
		Count:     len(items),
		Total:     total,
		Truncated: total > len(items),
		Items:     items,
	}, nil
}

// ExtractTable extracts structured table data.
func (s *Session) ExtractTable(tableSelector string) (*TableResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensurePage(); err != nil {
		return nil, err
	}

	// Use JS to extract everything in one call
	js := `(function() {
		const table = document.querySelector(` + jsonQuote(tableSelector) + `);
		if (!table) return null;
		const headers = Array.from(table.querySelectorAll('th')).map(h => h.textContent.trim());
		const rows = [];
		for (const tr of table.querySelectorAll('tbody tr, tr')) {
			const cells = tr.querySelectorAll('td');
			if (cells.length === 0) continue;
			rows.push(Array.from(cells).map(c => c.textContent.trim()));
		}
		return {headers: headers, rows: rows, rowCount: rows.length, colCount: headers.length};
	})()`

	result, err := s.page.Evaluate(js)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("table %q not found", tableSelector)
	}

	m, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected table result type")
	}

	tr := &TableResult{Selector: tableSelector}

	if headers, ok := m["headers"].([]any); ok {
		for _, h := range headers {
			s, _ := h.(string)
			tr.Headers = append(tr.Headers, s)
		}
	}
	if rows, ok := m["rows"].([]any); ok {
		for _, row := range rows {
			if cols, ok := row.([]any); ok {
				r := make([]string, 0, len(cols))
				for _, c := range cols {
					s, _ := c.(string)
					r = append(r, s)
				}
				tr.Rows = append(tr.Rows, r)
			}
		}
	}
	maxRows := s.contentOpts.MaxRows
	if maxRows == 0 {
		maxRows = 100
	}
	tr.RowCount = len(tr.Rows)
	tr.ColCount = len(tr.Headers)
	if len(tr.Rows) > maxRows {
		tr.Rows = tr.Rows[:maxRows]
		tr.Truncated = true
	}

	return tr, nil
}

// HasElement checks if an element exists on the page.
func (s *Session) HasElement(selector string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return false
	}
	_, err := s.querySelector(selector)
	return err == nil
}

// WaitFor waits until an element matching the selector appears.
func (s *Session) WaitFor(selector string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensurePage(); err != nil {
		return err
	}
	return s.page.WaitForSelector(selector)
}

// Eval executes JavaScript and returns the result.
func (s *Session) Eval(js string) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensurePage(); err != nil {
		return nil, err
	}
	return s.page.Evaluate(js)
}

// Page returns the underlying Page for advanced operations.
// The caller must not hold the session mutex.
func (s *Session) Page() *browse.Page {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.page
}

// Screenshot captures the page as an image.
// Automatically compresses to fit within MaxScreenshotBytes (default 5MB).
func (s *Session) Screenshot() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensurePage(); err != nil {
		return nil, err
	}
	return s.page.ScreenshotWithOptions(browse.ScreenshotOptions{
		MaxSize: s.contentOpts.MaxScreenshotBytes,
	})
}

// PDF generates a PDF of the page.
func (s *Session) PDF() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensurePage(); err != nil {
		return nil, err
	}
	return s.page.PDF()
}

// waitAndResolve waits for an element to appear before interacting.
// Supports both CSS selectors and Playwright-style :text('...') selectors.
func (s *Session) waitAndResolve(selector string) error {
	// Try CSS first
	err := s.page.WaitForSelector(selector)
	if err == nil {
		return nil
	}
	// Try Playwright-style text selector as fallback
	_, resolveErr := s.resolveSelector(selector)
	if resolveErr == nil {
		return nil
	}
	return err // return original CSS error
}

// querySelector resolves a selector to a nodeID, supporting Playwright-style syntax.
func (s *Session) querySelector(selector string) (int64, error) {
	return s.resolveSelector(selector)
}

func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
