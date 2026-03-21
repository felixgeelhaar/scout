package agent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var textSelectorRe = regexp.MustCompile(`:text\(['"](.+?)['"]\)`)
var hasTextSelectorRe = regexp.MustCompile(`:has-text\(['"](.+?)['"]\)`)

// resolveSelector translates Playwright-style selectors to JS-based element lookup
// when standard CSS selectors won't work. Falls back to the original selector if
// it looks like valid CSS.
func (s *Session) resolveSelector(selector string) (int64, error) {
	// Try standard CSS first
	nodeID, err := s.page.QuerySelector(selector)
	if err == nil {
		return nodeID, nil
	}

	// Check for Playwright :text('...') syntax
	if matches := textSelectorRe.FindStringSubmatch(selector); len(matches) > 1 {
		text := matches[1]
		tag := strings.TrimSuffix(selector[:textSelectorRe.FindStringIndex(selector)[0]], " ")
		if tag == "" {
			tag = "*"
		}
		return s.findByText(tag, text)
	}

	// Check for :has-text('...')
	if matches := hasTextSelectorRe.FindStringSubmatch(selector); len(matches) > 1 {
		text := matches[1]
		tag := strings.TrimSuffix(selector[:hasTextSelectorRe.FindStringIndex(selector)[0]], " ")
		if tag == "" {
			tag = "*"
		}
		return s.findByText(tag, text)
	}

	// Return original error
	return 0, err
}

// findByText finds an element by tag and text content via JS.
func (s *Session) findByText(tag, text string) (int64, error) {
	tagJSON, _ := json.Marshal(tag)
	textJSON, _ := json.Marshal(text)

	// Use XPath to find by text, then resolve to nodeId via DOM.querySelector workaround
	js := fmt.Sprintf(`(function() {
		const tag = %s;
		const text = %s;
		const elements = document.querySelectorAll(tag === '*' ? 'a,button,span,div,p,li,h1,h2,h3,h4,label,input,td,th' : tag);
		for (const el of elements) {
			if (el.textContent.trim() === text || el.textContent.trim().includes(text)) {
				// Generate a unique selector for this element
				el.setAttribute('data-scout-found', 'true');
				return true;
			}
		}
		return false;
	})()`, tagJSON, textJSON)

	result, err := s.page.Evaluate(js)
	if err != nil {
		return 0, err
	}
	if b, ok := result.(bool); !ok || !b {
		return 0, fmt.Errorf("agent: no element found with text %q", text)
	}

	// Now query the marked element
	nodeID, err := s.page.QuerySelector("[data-scout-found]")
	if err != nil {
		return 0, err
	}

	// Clean up the marker
	_, _ = s.page.Evaluate(`document.querySelector('[data-scout-found]')?.removeAttribute('data-scout-found')`)

	return nodeID, nil
}
