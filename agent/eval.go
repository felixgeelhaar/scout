package agent

import (
	"os"
	"strconv"
)

// EvalEnabled reports whether arbitrary JavaScript evaluation is opted into
// via SCOUT_ENABLE_EVAL. Eval is disabled by default: it runs in the page
// context and can read cookies, call fetch, and otherwise act as the origin.
func EvalEnabled() bool {
	v := os.Getenv("SCOUT_ENABLE_EVAL")
	if v == "" {
		return false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false
	}
	return b
}
