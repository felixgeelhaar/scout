package agent

import "testing"

func TestLooksLikeNaturalLanguage(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"the login button", true},
		{"search input field", true},
		{"submit form button", true},
		{"click me now", true},
		{"#login", false},
		{".btn-primary", false},
		{"button[type=submit]", false},
		{"div > span", false},
		{"a:first-child", false},
		{"button", false},
		{"input+label", false},
		{"h1~p", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := looksLikeNaturalLanguage(tt.input)
			if got != tt.want {
				t.Errorf("looksLikeNaturalLanguage(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
