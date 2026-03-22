package agent

import "testing"

func TestRateMetric_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		value  float64
		good   float64
		poor   float64
		expect string
	}{
		{"exactly between thresholds", 3000, 2500, 4000, "needs-improvement"},
		{"just above good", 2501, 2500, 4000, "needs-improvement"},
		{"just below poor", 3999, 2500, 4000, "needs-improvement"},
		{"negative value", -1, 2500, 4000, "good"},
		{"very large value", 100000, 2500, 4000, "poor"},
		{"zero thresholds", 0, 0, 0, "good"},
		{"equal thresholds", 5, 5, 5, "good"},
		{"CLS boundary good", 0.1, 0.1, 0.25, "good"},
		{"CLS boundary poor", 0.25, 0.1, 0.25, "poor"},
		{"CLS between", 0.15, 0.1, 0.25, "needs-improvement"},
		{"INP boundary good", 200, 200, 500, "good"},
		{"INP boundary poor", 500, 200, 500, "poor"},
		{"INP between", 350, 200, 500, "needs-improvement"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rateMetric(tt.value, tt.good, tt.poor)
			if got != tt.expect {
				t.Errorf("rateMetric(%v, %v, %v) = %q, want %q", tt.value, tt.good, tt.poor, got, tt.expect)
			}
		})
	}
}

func TestOverallRating_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		ratings []string
		expect  string
	}{
		{"single good", []string{"good"}, "good"},
		{"single poor", []string{"poor"}, "poor"},
		{"single needs-improvement", []string{"needs-improvement"}, "needs-improvement"},
		{"poor short-circuits", []string{"poor", "good", "good"}, "poor"},
		{"needs-improvement propagates", []string{"good", "good", "needs-improvement"}, "needs-improvement"},
		{"multiple needs-improvement", []string{"needs-improvement", "needs-improvement"}, "needs-improvement"},
		{"poor first", []string{"poor", "needs-improvement", "good"}, "poor"},
		{"all needs-improvement", []string{"needs-improvement", "needs-improvement", "needs-improvement"}, "needs-improvement"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := overallRating(tt.ratings...)
			if got != tt.expect {
				t.Errorf("overallRating(%v) = %q, want %q", tt.ratings, got, tt.expect)
			}
		})
	}
}
