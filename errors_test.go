package browse

import (
	"errors"
	"testing"
)

func TestTimeoutError(t *testing.T) {
	e := &TimeoutError{Operation: "click", Selector: "#btn"}
	s := e.Error()
	if s != `browse: timeout waiting for click on "#btn"` {
		t.Errorf("unexpected: %s", s)
	}

	e2 := &TimeoutError{Operation: "page load"}
	s2 := e2.Error()
	if s2 != "browse: timeout during page load" {
		t.Errorf("unexpected: %s", s2)
	}
}

func TestElementNotFoundError(t *testing.T) {
	e := &ElementNotFoundError{Selector: "#missing"}
	s := e.Error()
	if s != `browse: element not found: "#missing"` {
		t.Errorf("unexpected: %s", s)
	}
}

func TestNavigationError(t *testing.T) {
	inner := errors.New("net error")
	e := &NavigationError{URL: "http://bad", Err: inner}
	s := e.Error()
	if s != `browse: navigation to "http://bad" failed: net error` {
		t.Errorf("unexpected: %s", s)
	}
	if !errors.Is(e, inner) {
		t.Error("Unwrap should return inner error")
	}
}
