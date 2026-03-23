package cdp

import "encoding/json"

// CreateTarget creates a new browser tab/page and returns its target ID.
func (c *Conn) CreateTarget(url string) (string, error) {
	params := map[string]string{"url": url}
	result, err := c.Call("Target.createTarget", params)
	if err != nil {
		return "", err
	}

	var resp struct {
		TargetID string `json:"targetId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return "", err
	}
	return resp.TargetID, nil
}

// AttachToTarget attaches to a target and returns the session ID.
func (c *Conn) AttachToTarget(targetID string) (string, error) {
	params := map[string]any{
		"targetId": targetID,
		"flatten":  true,
	}
	result, err := c.Call("Target.attachToTarget", params)
	if err != nil {
		return "", err
	}

	var resp struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return "", err
	}
	return resp.SessionID, nil
}

// GetTargets returns the list of available targets (pages/tabs).
func (c *Conn) GetTargets() ([]TargetInfo, error) {
	result, err := c.Call("Target.getTargets", nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		TargetInfos []TargetInfo `json:"targetInfos"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, err
	}
	return resp.TargetInfos, nil
}

// TargetInfo describes a CDP target.
type TargetInfo struct {
	TargetID string `json:"targetId"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	URL      string `json:"url"`
}

// CloseTarget closes a browser tab/page.
func (c *Conn) CloseTarget(targetID string) error {
	params := map[string]string{"targetId": targetID}
	_, err := c.Call("Target.closeTarget", params)
	return err
}
