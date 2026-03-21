package browse

import "testing"

func TestNewCreatesEngine(t *testing.T) {
	e := New()
	if e == nil {
		t.Fatal("New() returned nil")
	}
	if len(e.middleware) != 0 {
		t.Errorf("New() should have no middleware, got %d", len(e.middleware))
	}
}

func TestDefaultHasMiddleware(t *testing.T) {
	e := Default()
	if e == nil {
		t.Fatal("Default() returned nil")
	}
	// Logger + Recovery = 2 middleware
	if len(e.middleware) != 2 {
		t.Errorf("Default() should have 2 middleware, got %d", len(e.middleware))
	}
}

func TestOptions(t *testing.T) {
	e := New(
		WithHeadless(false),
		WithViewport(1920, 1080),
	)
	if e.opts.headless != false {
		t.Error("expected headless=false")
	}
	if e.opts.width != 1920 || e.opts.height != 1080 {
		t.Errorf("expected 1920x1080, got %dx%d", e.opts.width, e.opts.height)
	}
}
