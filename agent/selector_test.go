package agent

import "testing"

func TestTextSelectorRegex(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{"single quotes", `button:text('Login')`, "Login", true},
		{"double quotes", `button:text("Login")`, "Login", true},
		{"with spaces", `a:text('Sign In')`, "Sign In", true},
		{"no match", "button.primary", "", false},
		{"empty text", `:text('')`, "", false},
		{"complex selector prefix", `div.nav a:text('Home')`, "Home", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := textSelectorRe.FindStringSubmatch(tt.input)
			if tt.ok {
				if len(matches) < 2 {
					t.Fatalf("expected match for %q, got none", tt.input)
				}
				if matches[1] != tt.want {
					t.Errorf("got %q, want %q", matches[1], tt.want)
				}
			} else {
				if len(matches) > 1 {
					t.Errorf("expected no match for %q, got %q", tt.input, matches[1])
				}
			}
		})
	}
}

func TestHasTextSelectorRegex(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{"single quotes", `div:has-text('Hello')`, "Hello", true},
		{"double quotes", `div:has-text("Hello")`, "Hello", true},
		{"with tag", `span:has-text('World')`, "World", true},
		{"no match", "div.has-text", "", false},
		{"no match plain", "button", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := hasTextSelectorRe.FindStringSubmatch(tt.input)
			if tt.ok {
				if len(matches) < 2 {
					t.Fatalf("expected match for %q, got none", tt.input)
				}
				if matches[1] != tt.want {
					t.Errorf("got %q, want %q", matches[1], tt.want)
				}
			} else {
				if len(matches) > 1 {
					t.Errorf("expected no match for %q, got %q", tt.input, matches[1])
				}
			}
		})
	}
}

func TestLooksLikeNaturalLanguage_EdgeCases(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"", false},
		{" ", true},
		{"a b", true},
		{"click the button", true},
		{"click the #button", false},
		{"click the .button", false},
		{"a[href] link", false},
		{"a > b", false},
		{"a + b", false},
		{"a ~ b", false},
		{"foo:bar baz", false},
		{"two words", true},
		{"  spaces  everywhere  ", true},
		{"no-spaces", false},
		{"single", false},
		{"the big red button on the page", true},
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
