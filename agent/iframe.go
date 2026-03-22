package agent

import (
	"encoding/json"
	"fmt"
)

// FrameInfo describes an iframe discovered on the page.
type FrameInfo struct {
	FrameID  string `json:"frame_id"`
	URL      string `json:"url"`
	Name     string `json:"name,omitempty"`
	Selector string `json:"selector,omitempty"`
}

// SwitchToFrame switches the session's execution context to the iframe matching
// the given CSS selector. Subsequent Evaluate, Click, Type, etc. calls will
// operate inside the iframe until SwitchToMainFrame is called.
func (s *Session) SwitchToFrame(selector string) (*PageResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensurePage(); err != nil {
		return nil, err
	}

	frameTree, err := s.page.Call("Page.getFrameTree", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get frame tree: %w", err)
	}

	var tree struct {
		FrameTree struct {
			Frame struct {
				ID string `json:"id"`
			} `json:"frame"`
			ChildFrames []struct {
				Frame struct {
					ID   string `json:"id"`
					URL  string `json:"url"`
					Name string `json:"name"`
				} `json:"frame"`
			} `json:"childFrames"`
		} `json:"frameTree"`
	}
	if err := json.Unmarshal(frameTree, &tree); err != nil {
		return nil, fmt.Errorf("failed to parse frame tree: %w", err)
	}

	selectorJSON, _ := json.Marshal(selector)
	js := fmt.Sprintf(`(function() {
		const iframe = document.querySelector(%s);
		if (!iframe || iframe.tagName !== 'IFRAME') return '';
		return iframe.src || '';
	})()`, selectorJSON)

	result, err := s.page.Evaluate(js)
	if err != nil {
		return nil, fmt.Errorf("failed to find iframe %s: %w", selector, err)
	}
	iframeSrc, _ := result.(string)

	var targetFrameID string
	for _, child := range tree.FrameTree.ChildFrames {
		if child.Frame.URL == iframeSrc || child.Frame.Name == selector {
			targetFrameID = child.Frame.ID
			break
		}
	}

	if targetFrameID == "" {
		nameJS := fmt.Sprintf(`(function() {
			const iframe = document.querySelector(%s);
			if (!iframe) return '';
			return iframe.name || iframe.id || '';
		})()`, selectorJSON)
		nameResult, _ := s.page.Evaluate(nameJS)
		frameName, _ := nameResult.(string)

		for _, child := range tree.FrameTree.ChildFrames {
			if frameName != "" && child.Frame.Name == frameName {
				targetFrameID = child.Frame.ID
				break
			}
		}
	}

	if targetFrameID == "" {
		return nil, fmt.Errorf("iframe %q not found in frame tree", selector)
	}

	worldResult, err := s.page.Call("Page.createIsolatedWorld", map[string]any{
		"frameId":             targetFrameID,
		"grantUniveralAccess": true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create execution context for frame: %w", err)
	}

	var world struct {
		ExecutionContextID int64 `json:"executionContextId"`
	}
	if err := json.Unmarshal(worldResult, &world); err != nil {
		return nil, fmt.Errorf("failed to parse execution context: %w", err)
	}

	s.frameID = targetFrameID
	s.frameContextID = world.ExecutionContextID

	url, _ := s.evaluateInFrame(`window.location.href`)
	title, _ := s.evaluateInFrame(`document.title`)
	urlStr, _ := url.(string)
	titleStr, _ := title.(string)

	s.addHistory("switch_to_frame", selector, "", "")
	return &PageResult{URL: urlStr, Title: titleStr}, nil
}

// SwitchToMainFrame resets the session back to the main frame context.
func (s *Session) SwitchToMainFrame() (*PageResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensurePage(); err != nil {
		return nil, err
	}

	s.frameID = ""
	s.frameContextID = 0

	s.addHistory("switch_to_main_frame", "", "", "")
	return s.pageResult()
}

// evaluateInFrame executes JS in the current frame's execution context.
// Must be called with s.mu held.
func (s *Session) evaluateInFrame(expression string) (any, error) {
	if s.frameContextID == 0 {
		return s.page.Evaluate(expression)
	}

	params := map[string]any{
		"expression":    expression,
		"contextId":     s.frameContextID,
		"returnByValue": true,
		"awaitPromise":  true,
	}
	result, err := s.page.Call("Runtime.evaluate", params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Result struct {
			Type  string          `json:"type"`
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
		return nil, fmt.Errorf("js error in frame: %s", resp.ExceptionDetails.Text)
	}

	var val any
	if len(resp.Result.Value) == 0 {
		return nil, nil
	}
	if err := json.Unmarshal(resp.Result.Value, &val); err != nil {
		return nil, nil //nolint:nilerr
	}
	return val, nil
}

// InFrame returns true if the session is currently targeting an iframe.
func (s *Session) InFrame() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.frameID != ""
}
