// Package browse provides a Gin-like API for browser automation using pure CDP over WebSocket.
//
// browse-go applies Gin's middleware/context/group patterns to browser automation,
// giving Go developers a familiar, composable way to script browser interactions.
//
//	engine := browse.Default(browse.WithHeadless(true))
//	engine.MustLaunch()
//	defer engine.Close()
//
//	engine.Task("search", func(c *browse.Context) {
//	    c.MustNavigate("https://example.com")
//	    c.El("input[name=q]").MustInput("hello")
//	    c.El("button[type=submit]").MustClick()
//	})
//
//	engine.Run("search")
package browse

// Browser is the interface for browser lifecycle management.
// Engine implements this. The agent package depends on this interface
// rather than on *Engine directly, enabling testing without a real browser.
type Browser interface {
	NewPage() (*Page, error)
	NewPageAt(url string) (*Page, error)
	Close() error
}

// HandlerFunc defines the handler function signature for middleware and tasks.
type HandlerFunc func(*Context)

// HandlersChain is a slice of HandlerFunc used to build middleware chains.
type HandlersChain []HandlerFunc

// New creates a new Engine with no middleware attached.
func New(opts ...Option) *Engine {
	e := &Engine{
		opts:   applyOptions(opts),
		groups: make(map[string]*Group),
		tasks:  make(map[string]*taskEntry),
	}
	return e
}

// Default creates a new Engine with Logger and Recovery middleware attached.
func Default(opts ...Option) *Engine {
	e := New(opts...)
	e.Use(Logger(), Recovery())
	return e
}
