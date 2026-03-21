package browse

import (
	"encoding/json"
	"fmt"
)

// Selection wraps a single page element with a fluent, chainable API.
type Selection struct {
	page     *Page
	nodeID   int64
	selector string
	err      error
}

// NewSelection creates a Selection for the given page and node ID.
func NewSelection(page *Page, nodeID int64, selector string) *Selection {
	return &Selection{page: page, nodeID: nodeID, selector: selector}
}

// Err returns the accumulated error, if any.
func (s *Selection) Err() error {
	return s.err
}

// --- Interaction ---

// Click clicks the element.
func (s *Selection) Click() error {
	if s.err != nil {
		return s.err
	}
	// Get element center coordinates via DOM.getBoxModel
	result, err := s.page.call("DOM.getBoxModel", map[string]any{
		"nodeId": s.nodeID,
	})
	if err != nil {
		return fmt.Errorf("browse: click failed: %w", err)
	}

	var box struct {
		Model struct {
			Content []float64 `json:"content"` // [x1,y1, x2,y2, x3,y3, x4,y4]
		} `json:"model"`
	}
	if err := json.Unmarshal(result, &box); err != nil {
		return fmt.Errorf("browse: click failed to parse box model: %w", err)
	}

	if len(box.Model.Content) < 8 {
		return fmt.Errorf("browse: click failed: invalid box model for %q", s.selector)
	}

	// Calculate center from quad points
	x := (box.Model.Content[0] + box.Model.Content[2] + box.Model.Content[4] + box.Model.Content[6]) / 4
	y := (box.Model.Content[1] + box.Model.Content[3] + box.Model.Content[5] + box.Model.Content[7]) / 4

	// Dispatch mouse events
	for _, typ := range []string{"mousePressed", "mouseReleased"} {
		_, err := s.page.call("Input.dispatchMouseEvent", map[string]any{
			"type":       typ,
			"x":          x,
			"y":          y,
			"button":     "left",
			"clickCount": 1,
		})
		if err != nil {
			return fmt.Errorf("browse: click %s failed: %w", typ, err)
		}
	}
	return nil
}

// MustClick clicks the element and panics on error.
func (s *Selection) MustClick() *Selection {
	if err := s.Click(); err != nil {
		panic(err)
	}
	return s
}

// Input focuses the element, clears it, and types the given text.
func (s *Selection) Input(text string) error {
	if s.err != nil {
		return s.err
	}

	// Clear existing value via JS, then focus
	objectID, err := s.page.ResolveNode(s.nodeID)
	if err != nil {
		return fmt.Errorf("browse: input resolve failed: %w", err)
	}
	_, err = s.callFunctionOn(objectID, `function() { this.value = ''; this.focus(); }`)
	if err != nil {
		return fmt.Errorf("browse: input clear/focus failed: %w", err)
	}

	// Type text using insertText
	_, err = s.page.call("Input.insertText", map[string]any{
		"text": text,
	})
	if err != nil {
		return fmt.Errorf("browse: input text failed: %w", err)
	}
	return nil
}

// MustInput types text and panics on error.
func (s *Selection) MustInput(text string) *Selection {
	if err := s.Input(text); err != nil {
		panic(err)
	}
	return s
}

// Clear clears the element's value.
func (s *Selection) Clear() error {
	if s.err != nil {
		return s.err
	}
	return s.Input("")
}

// Hover moves the mouse over the element.
func (s *Selection) Hover() error {
	if s.err != nil {
		return s.err
	}

	result, err := s.page.call("DOM.getBoxModel", map[string]any{
		"nodeId": s.nodeID,
	})
	if err != nil {
		return err
	}

	var box struct {
		Model struct {
			Content []float64 `json:"content"`
		} `json:"model"`
	}
	if err := json.Unmarshal(result, &box); err != nil || len(box.Model.Content) < 8 {
		return fmt.Errorf("browse: hover failed: invalid box model")
	}

	x := (box.Model.Content[0] + box.Model.Content[2] + box.Model.Content[4] + box.Model.Content[6]) / 4
	y := (box.Model.Content[1] + box.Model.Content[3] + box.Model.Content[5] + box.Model.Content[7]) / 4

	_, err = s.page.call("Input.dispatchMouseEvent", map[string]any{
		"type": "mouseMoved",
		"x":    x,
		"y":    y,
	})
	return err
}

// --- Extraction ---

// Text returns the element's text content.
func (s *Selection) Text() (string, error) {
	if s.err != nil {
		return "", s.err
	}
	objectID, err := s.page.ResolveNode(s.nodeID)
	if err != nil {
		return "", err
	}
	result, err := s.callFunctionOn(objectID, `function() { return this.textContent; }`)
	if err != nil {
		return "", err
	}
	str, _ := result.(string)
	return str, nil
}

// MustText returns the element's text content, panicking on error.
func (s *Selection) MustText() string {
	t, err := s.Text()
	if err != nil {
		panic(err)
	}
	return t
}

// Attr returns the value of the given attribute.
func (s *Selection) Attr(name string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	objectID, err := s.page.ResolveNode(s.nodeID)
	if err != nil {
		return "", err
	}
	result, err := s.callFunctionOn(objectID, fmt.Sprintf(`function() { return this.getAttribute(%q); }`, name))
	if err != nil {
		return "", err
	}
	if result == nil {
		return "", nil
	}
	str, _ := result.(string)
	return str, nil
}

// Visible reports whether the element is visible.
func (s *Selection) Visible() (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	objectID, err := s.page.ResolveNode(s.nodeID)
	if err != nil {
		return false, err
	}
	result, err := s.callFunctionOn(objectID, `function() {
		const style = window.getComputedStyle(this);
		return style.display !== 'none' && style.visibility !== 'hidden' && style.opacity !== '0';
	}`)
	if err != nil {
		return false, err
	}
	b, _ := result.(bool)
	return b, nil
}

// Screenshot captures a screenshot of this element only.
func (s *Selection) Screenshot() ([]byte, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.page.ScreenshotElement(s.nodeID)
}

// Value returns the element's value property (for inputs).
func (s *Selection) Value() (string, error) {
	if s.err != nil {
		return "", s.err
	}
	objectID, err := s.page.ResolveNode(s.nodeID)
	if err != nil {
		return "", err
	}
	result, err := s.callFunctionOn(objectID, `function() { return this.value; }`)
	if err != nil {
		return "", err
	}
	str, _ := result.(string)
	return str, nil
}

// --- Waiting ---

// WaitVisible waits until the element is visible.
func (s *Selection) WaitVisible() *Selection {
	if s.err != nil {
		return s
	}
	objectID, err := s.page.ResolveNode(s.nodeID)
	if err != nil {
		s.err = err
		return s
	}
	timeout := s.page.timeout.Milliseconds()
	_, err = s.callFunctionOn(objectID, fmt.Sprintf(`function() {
		return new Promise((resolve, reject) => {
			const timer = setTimeout(() => reject(new Error('timeout waiting for visible')), %d);
			const check = () => {
				const style = window.getComputedStyle(this);
				if (style.display !== 'none' && style.visibility !== 'hidden') {
					clearTimeout(timer);
					resolve(true);
				} else {
					requestAnimationFrame(check);
				}
			};
			check();
		});
	}`, timeout))
	if err != nil {
		s.err = err
	}
	return s
}

// WaitStable waits until the element's position is stable.
func (s *Selection) WaitStable() *Selection {
	if s.err != nil {
		return s
	}
	objectID, err := s.page.ResolveNode(s.nodeID)
	if err != nil {
		s.err = err
		return s
	}
	timeout := s.page.timeout.Milliseconds()
	_, err = s.callFunctionOn(objectID, fmt.Sprintf(`function() {
		return new Promise((resolve, reject) => {
			const timer = setTimeout(() => reject(new Error('timeout waiting for stable')), %d);
			let prev = this.getBoundingClientRect();
			const check = () => {
				const curr = this.getBoundingClientRect();
				if (prev.x === curr.x && prev.y === curr.y && prev.width === curr.width && prev.height === curr.height) {
					clearTimeout(timer);
					resolve(true);
				} else {
					prev = curr;
					requestAnimationFrame(check);
				}
			};
			requestAnimationFrame(check);
		});
	}`, timeout))
	if err != nil {
		s.err = err
	}
	return s
}

// WaitEnabled waits until the element is enabled.
func (s *Selection) WaitEnabled() *Selection {
	if s.err != nil {
		return s
	}
	objectID, err := s.page.ResolveNode(s.nodeID)
	if err != nil {
		s.err = err
		return s
	}
	timeout := s.page.timeout.Milliseconds()
	_, err = s.callFunctionOn(objectID, fmt.Sprintf(`function() {
		return new Promise((resolve, reject) => {
			const timer = setTimeout(() => reject(new Error('timeout waiting for enabled')), %d);
			const check = () => {
				if (!this.disabled) {
					clearTimeout(timer);
					resolve(true);
				} else {
					requestAnimationFrame(check);
				}
			};
			check();
		});
	}`, timeout))
	if err != nil {
		s.err = err
	}
	return s
}

func (s *Selection) callFunctionOn(objectID, fn string) (any, error) {
	result, err := s.page.call("Runtime.callFunctionOn", map[string]any{
		"functionDeclaration": fn,
		"objectId":            objectID,
		"returnByValue":       true,
		"awaitPromise":        true,
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text string `json:"text"`
		} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, err
	}
	if resp.ExceptionDetails != nil {
		return nil, fmt.Errorf("browse: js error: %s", resp.ExceptionDetails.Text)
	}

	var val any
	if len(resp.Result.Value) == 0 {
		return nil, nil // undefined result
	}
	if err := json.Unmarshal(resp.Result.Value, &val); err != nil {
		return nil, nil //nolint:nilerr // unmarshal of undefined/void JS results is expected
	}
	return val, nil
}
