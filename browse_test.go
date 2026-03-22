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

func TestHandlersChainType(t *testing.T) {
	var chain HandlersChain
	if len(chain) != 0 {
		t.Error("nil chain should have length 0")
	}

	chain = append(chain, func(c *Context) {})
	chain = append(chain, func(c *Context) {})
	if len(chain) != 2 {
		t.Errorf("chain len = %d, want 2", len(chain))
	}
}

func TestHandlerFuncType(t *testing.T) {
	called := false
	var fn HandlerFunc = func(c *Context) {
		called = true
	}

	ctx := newContext(nil, "test", HandlersChain{fn})
	ctx.Next()

	if !called {
		t.Error("HandlerFunc should have been called")
	}
}

func TestCookieStruct(t *testing.T) {
	c := Cookie{
		Name:     "session",
		Value:    "abc123",
		Domain:   "example.com",
		Path:     "/",
		Expires:  1234567890,
		Secure:   true,
		HTTPOnly: true,
		SameSite: "Strict",
	}

	if c.Name != "session" {
		t.Errorf("Name = %q", c.Name)
	}
	if c.Value != "abc123" {
		t.Errorf("Value = %q", c.Value)
	}
	if c.Domain != "example.com" {
		t.Errorf("Domain = %q", c.Domain)
	}
	if c.Path != "/" {
		t.Errorf("Path = %q", c.Path)
	}
	if !c.Secure {
		t.Error("Secure should be true")
	}
	if !c.HTTPOnly {
		t.Error("HTTPOnly should be true")
	}
	if c.SameSite != "Strict" {
		t.Errorf("SameSite = %q", c.SameSite)
	}
}

func TestScreenshotOptionsDefaults(t *testing.T) {
	opts := ScreenshotOptions{}
	if opts.Format != "" {
		t.Errorf("default Format should be empty, got %q", opts.Format)
	}
	if opts.Quality != 0 {
		t.Errorf("default Quality should be 0, got %d", opts.Quality)
	}
	if opts.FullPage {
		t.Error("default FullPage should be false")
	}
	if opts.Clip != nil {
		t.Error("default Clip should be nil")
	}
	if opts.MaxSize != 0 {
		t.Errorf("default MaxSize should be 0, got %d", opts.MaxSize)
	}
	if opts.MaxWidth != 0 {
		t.Errorf("default MaxWidth should be 0, got %d", opts.MaxWidth)
	}
}

func TestClipRegionStruct(t *testing.T) {
	clip := ClipRegion{X: 10, Y: 20, Width: 100, Height: 200}
	if clip.X != 10 || clip.Y != 20 || clip.Width != 100 || clip.Height != 200 {
		t.Errorf("ClipRegion = %+v", clip)
	}
}

func TestPDFOptionsStruct(t *testing.T) {
	opts := PDFOptions{
		Landscape:       true,
		PrintBackground: true,
		Scale:           0.5,
		PaperWidth:      8.5,
		PaperHeight:     11,
		MarginTop:       0.4,
		MarginBottom:    0.4,
		MarginLeft:      0.4,
		MarginRight:     0.4,
		PageRanges:      "1-5",
	}
	if !opts.Landscape {
		t.Error("Landscape should be true")
	}
	if !opts.PrintBackground {
		t.Error("PrintBackground should be true")
	}
	if opts.Scale != 0.5 {
		t.Errorf("Scale = %f", opts.Scale)
	}
	if opts.PageRanges != "1-5" {
		t.Errorf("PageRanges = %q", opts.PageRanges)
	}
}

func TestRecorderOptionsDefaults(t *testing.T) {
	opts := RecorderOptions{}
	if opts.Format != "" {
		t.Errorf("default Format = %q, want empty", opts.Format)
	}
	if opts.Quality != 0 {
		t.Errorf("default Quality = %d, want 0", opts.Quality)
	}
	if opts.MaxWidth != 0 {
		t.Errorf("default MaxWidth = %d, want 0", opts.MaxWidth)
	}
	if opts.MaxHeight != 0 {
		t.Errorf("default MaxHeight = %d, want 0", opts.MaxHeight)
	}
}

func TestAllowedSchemes(t *testing.T) {
	if !allowedSchemes["http"] {
		t.Error("http should be allowed")
	}
	if !allowedSchemes["https"] {
		t.Error("https should be allowed")
	}
	if allowedSchemes["ftp"] {
		t.Error("ftp should not be allowed")
	}
	if allowedSchemes["file"] {
		t.Error("file should not be allowed")
	}
	if allowedSchemes[""] {
		t.Error("empty scheme should not be allowed")
	}
}
