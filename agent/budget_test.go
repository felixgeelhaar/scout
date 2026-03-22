package agent

import "testing"

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty string", "", 0},
		{"single char", "a", 1},
		{"four chars", "abcd", 1},
		{"five chars", "abcde", 2},
		{"eight chars", "abcdefgh", 2},
		{"twelve chars", "abcdefghijkl", 3},
		{"short sentence", "Hello, World!", 4},
		{"longer text", "The quick brown fox jumps over the lazy dog", 11},
		{"one byte", "x", 1},
		{"three bytes", "xyz", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.input)
			if got != tt.want {
				t.Errorf("EstimateTokens(%q) = %d, want %d (len=%d)", tt.input, got, tt.want, len(tt.input))
			}
		})
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{0, 0, 0},
		{-1, 1, -1},
		{100, 100, 100},
		{-5, -3, -5},
	}
	for _, tt := range tests {
		got := min(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestDefaultContentOptions(t *testing.T) {
	opts := DefaultContentOptions()
	if opts.MaxLength != 4000 {
		t.Errorf("MaxLength: got %d, want 4000", opts.MaxLength)
	}
	if opts.MaxLinks != 25 {
		t.Errorf("MaxLinks: got %d, want 25", opts.MaxLinks)
	}
	if opts.MaxInputs != 20 {
		t.Errorf("MaxInputs: got %d, want 20", opts.MaxInputs)
	}
	if opts.MaxButtons != 15 {
		t.Errorf("MaxButtons: got %d, want 15", opts.MaxButtons)
	}
	if opts.MaxItems != 50 {
		t.Errorf("MaxItems: got %d, want 50", opts.MaxItems)
	}
	if opts.MaxRows != 100 {
		t.Errorf("MaxRows: got %d, want 100", opts.MaxRows)
	}
	if opts.MaxScreenshotBytes != 5*1024*1024 {
		t.Errorf("MaxScreenshotBytes: got %d, want %d", opts.MaxScreenshotBytes, 5*1024*1024)
	}
}
