package agent

import "testing"

func TestRateMetric(t *testing.T) {
	tests := []struct {
		name   string
		value  float64
		good   float64
		poor   float64
		expect string
	}{
		{"good LCP", 1500, 2500, 4000, "good"},
		{"needs-improvement LCP", 3000, 2500, 4000, "needs-improvement"},
		{"poor LCP", 5000, 2500, 4000, "poor"},
		{"boundary good", 2500, 2500, 4000, "good"},
		{"boundary poor", 4000, 2500, 4000, "poor"},
		{"good CLS", 0.05, 0.1, 0.25, "good"},
		{"poor CLS", 0.3, 0.1, 0.25, "poor"},
		{"good INP", 100, 200, 500, "good"},
		{"poor INP", 600, 200, 500, "poor"},
		{"zero value", 0, 2500, 4000, "good"},
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

func TestOverallRating(t *testing.T) {
	tests := []struct {
		name    string
		ratings []string
		expect  string
	}{
		{"all good", []string{"good", "good", "good"}, "good"},
		{"one needs-improvement", []string{"good", "needs-improvement", "good"}, "needs-improvement"},
		{"one poor", []string{"good", "needs-improvement", "poor"}, "poor"},
		{"all poor", []string{"poor", "poor", "poor"}, "poor"},
		{"empty", []string{}, "good"},
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
