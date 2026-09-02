package agent

import "testing"

func TestEvalEnabled(t *testing.T) {
	cases := []struct {
		name string
		val  string
		set  bool
		want bool
	}{
		{"unset", "", false, false},
		{"true", "true", true, true},
		{"1", "1", true, true},
		{"0", "0", true, false},
		{"false", "false", true, false},
		{"garbage", "yesplease", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.set {
				t.Setenv("SCOUT_ENABLE_EVAL", tc.val)
			} else {
				t.Setenv("SCOUT_ENABLE_EVAL", "")
			}
			if got := EvalEnabled(); got != tc.want {
				t.Errorf("EvalEnabled() = %v, want %v", got, tc.want)
			}
		})
	}
}
