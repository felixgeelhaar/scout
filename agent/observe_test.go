package agent

import "testing"

func TestEstimateLinkCost(t *testing.T) {
	tests := []struct {
		href string
		want string
	}{
		{"", "low"},
		{"#", "low"},
		{"#section", "low"},
		{"javascript:void(0)", "medium"},
		{"javascript:doSomething()", "medium"},
		{"https://example.com", "high"},
		{"/about", "high"},
		{"http://localhost:3000/page", "high"},
		{"/path/to/page?q=1", "high"},
		{"mailto:test-user", "high"},
	}
	for _, tt := range tests {
		t.Run(tt.href, func(t *testing.T) {
			got := estimateLinkCost(tt.href)
			if got != tt.want {
				t.Errorf("estimateLinkCost(%q) = %q, want %q", tt.href, got, tt.want)
			}
		})
	}
}

func TestEstimateButtonCost(t *testing.T) {
	tests := []struct {
		name    string
		btnType string
		text    string
		want    string
	}{
		{"submit type", "submit", "Go", "high"},
		{"submit text", "button", "Submit Form", "high"},
		{"sign in text", "button", "Sign In", "high"},
		{"login text", "button", "Login", "high"},
		{"register text", "button", "Register Now", "high"},
		{"create text", "button", "Create Account", "high"},
		{"delete text", "button", "Delete Item", "high"},
		{"save text", "button", "Save Changes", "high"},
		{"checkout text", "button", "Checkout", "high"},
		{"next text", "button", "Next", "medium"},
		{"continue text", "button", "Continue", "medium"},
		{"load more text", "button", "Load More", "medium"},
		{"search text", "button", "Search", "medium"},
		{"toggle text", "button", "Toggle", "low"},
		{"close text", "button", "Close", "low"},
		{"expand text", "button", "Expand", "low"},
		{"empty text", "button", "", "low"},
		{"cancel text", "button", "Cancel", "low"},
		{"submit type overrides text", "submit", "Cancel", "high"},
		{"case insensitive", "button", "SIGN IN", "high"},
		{"mixed case", "button", "Log In", "low"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimateButtonCost(tt.btnType, tt.text)
			if got != tt.want {
				t.Errorf("estimateButtonCost(%q, %q) = %q, want %q", tt.btnType, tt.text, got, tt.want)
			}
		})
	}
}

func TestStrVal(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]any
		key  string
		want string
	}{
		{"existing string", map[string]any{"k": "v"}, "k", "v"},
		{"missing key", map[string]any{"k": "v"}, "other", ""},
		{"non-string value", map[string]any{"k": 123}, "k", ""},
		{"nil value", map[string]any{"k": nil}, "k", ""},
		{"empty map", map[string]any{}, "k", ""},
		{"empty string", map[string]any{"k": ""}, "k", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strVal(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("strVal(%v, %q) = %q, want %q", tt.m, tt.key, got, tt.want)
			}
		})
	}
}
