package browse

import (
	"context"
	"sync"
)

// Context carries the page state, middleware chain, and data for a single task execution.
type Context struct {
	ctx      context.Context
	cancel   context.CancelFunc
	page     *Page
	taskName string
	handlers HandlersChain
	index    int
	mu       sync.RWMutex
	keys     map[string]any
	errors   []error
	aborted  bool
}

// NewTestContext creates a Context without a page for testing middleware chains.
func NewTestContext(taskName string, handlers HandlersChain) *Context {
	return newContext(nil, taskName, handlers)
}

func newContext(page *Page, taskName string, handlers HandlersChain) *Context {
	ctx, cancel := context.WithCancel(context.Background())
	return &Context{
		ctx:      ctx,
		cancel:   cancel,
		page:     page,
		taskName: taskName,
		handlers: handlers,
		index:    -1,
		keys:     make(map[string]any),
	}
}

// GoContext returns the underlying context.Context for cancellation propagation.
// Use this in middleware that wraps fortify or other context-aware libraries.
func (c *Context) GoContext() context.Context {
	return c.ctx
}

// SaveIndex returns the current handler index so it can be restored for retry.
// Used by resilience middleware (retry, timeout, circuit breaker, bulkhead).
func (c *Context) SaveIndex() int {
	return c.index
}

// RestoreIndex resets the handler chain position for re-execution.
// Used by resilience middleware to replay the downstream handler chain.
func (c *Context) RestoreIndex(idx int) {
	c.index = idx
	c.aborted = false
	c.mu.Lock()
	c.errors = nil
	c.mu.Unlock()
}

// --- Chain control ---

// Next calls the next handler in the middleware chain.
func (c *Context) Next() {
	c.index++
	for c.index < len(c.handlers) {
		if c.aborted {
			return
		}
		c.handlers[c.index](c)
		c.index++
	}
}

// Abort stops the middleware chain from continuing.
func (c *Context) Abort() {
	c.aborted = true
}

// AbortWithError stops the chain and records an error.
func (c *Context) AbortWithError(err error) {
	c.addError(err)
	c.Abort()
}

// IsAborted returns whether the chain has been aborted.
func (c *Context) IsAborted() bool {
	return c.aborted
}

// --- Data passing ---

// Set stores a key-value pair on the context.
func (c *Context) Set(key string, value any) {
	c.mu.Lock()
	c.keys[key] = value
	c.mu.Unlock()
}

// Get retrieves a value by key. The second return value indicates existence.
func (c *Context) Get(key string) (any, bool) {
	c.mu.RLock()
	val, ok := c.keys[key]
	c.mu.RUnlock()
	return val, ok
}

// GetString returns the string value for key, or "" if not found.
func (c *Context) GetString(key string) string {
	v, ok := c.Get(key)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// TaskName returns the name of the currently executing task.
func (c *Context) TaskName() string {
	return c.taskName
}

// Errors returns all errors recorded on this context.
func (c *Context) Errors() []error {
	return c.errors
}

func (c *Context) addError(err error) {
	c.mu.Lock()
	c.errors = append(c.errors, err)
	c.mu.Unlock()
}

// --- Navigation ---

// Navigate loads the given URL and waits for the page to be ready.
func (c *Context) Navigate(url string) error {
	return c.page.Navigate(url)
}

// MustNavigate calls Navigate and panics on error.
func (c *Context) MustNavigate(url string) *Context {
	if err := c.Navigate(url); err != nil {
		panic(err)
	}
	return c
}

// WaitLoad waits for the page load event.
func (c *Context) WaitLoad() error {
	return c.page.WaitLoad()
}

// WaitStable waits until the page DOM is stable.
func (c *Context) WaitStable() error {
	return c.page.WaitStable(0)
}

// WaitSelector waits until an element matching the selector appears in the DOM.
func (c *Context) WaitSelector(selector string) error {
	return c.page.WaitForSelector(selector)
}

// WaitNavigation waits for a navigation to complete after performing an action.
// Call this after clicking a link that triggers a page load.
func (c *Context) WaitNavigation() error {
	return c.page.WaitLoad()
}

// URL returns the current page URL.
func (c *Context) URL() string {
	u, _ := c.page.URL()
	return u
}

// Cookies returns all cookies for the current page.
func (c *Context) Cookies() ([]Cookie, error) {
	return c.page.Cookies()
}

// SetCookie sets a cookie on the current page.
func (c *Context) SetCookie(cookie Cookie) error {
	return c.page.SetCookie(cookie)
}

// FillForm fills multiple form fields at once.
// The map keys are CSS selectors, values are the text to input.
func (c *Context) FillForm(fields map[string]string) error {
	for selector, value := range fields {
		el := c.El(selector)
		if err := el.Input(value); err != nil {
			return err
		}
	}
	return nil
}

// --- Element selection ---

// El returns a Selection for the first element matching the CSS selector.
func (c *Context) El(selector string) *Selection {
	nodeID, err := c.page.QuerySelector(selector)
	if err != nil {
		return &Selection{err: err}
	}
	return &Selection{page: c.page, nodeID: nodeID, selector: selector}
}

// ElAll returns a SelectionAll for all elements matching the CSS selector.
func (c *Context) ElAll(selector string) *SelectionAll {
	nodeIDs, err := c.page.QuerySelectorAll(selector)
	if err != nil {
		return &SelectionAll{err: err}
	}
	selections := make([]*Selection, len(nodeIDs))
	for i, nid := range nodeIDs {
		selections[i] = &Selection{page: c.page, nodeID: nid, selector: selector}
	}
	return &SelectionAll{selections: selections, selector: selector}
}

// HasEl checks whether at least one element matches the selector.
func (c *Context) HasEl(selector string) bool {
	_, err := c.page.QuerySelector(selector)
	return err == nil
}

// --- Page-level operations ---

// Screenshot captures the full page as PNG bytes.
func (c *Context) Screenshot() ([]byte, error) {
	return c.page.Screenshot()
}

// ScreenshotTo captures a screenshot and writes it to the given file path.
func (c *Context) ScreenshotTo(path string) error {
	img, err := c.Screenshot()
	if err != nil {
		return err
	}
	return writeFile(path, img)
}

// ScreenshotFullPage captures the entire scrollable page as PNG bytes.
func (c *Context) ScreenshotFullPage() ([]byte, error) {
	return c.page.ScreenshotFullPage()
}

// ScreenshotElement captures a screenshot of a single element.
func (c *Context) ScreenshotElement(selector string) ([]byte, error) {
	nodeID, err := c.page.QuerySelector(selector)
	if err != nil {
		return nil, err
	}
	return c.page.ScreenshotElement(nodeID)
}

// PDF generates a PDF of the current page with default options.
func (c *Context) PDF() ([]byte, error) {
	return c.page.PDF()
}

// PDFTo generates a PDF and writes it to the given file path.
func (c *Context) PDFTo(path string) error {
	data, err := c.page.PDF()
	if err != nil {
		return err
	}
	return writeFile(path, data)
}

// StartRecording begins capturing screencast frames for video.
// Returns a Recorder that must be stopped and saved.
func (c *Context) StartRecording(opts RecorderOptions) (*Recorder, error) {
	rec, err := NewRecorder(c.page, opts)
	if err != nil {
		return nil, err
	}
	if err := rec.Start(); err != nil {
		return nil, err
	}
	return rec, nil
}

// Eval executes JavaScript on the page and returns the result.
func (c *Context) Eval(js string) (any, error) {
	return c.page.Evaluate(js)
}

// HTML returns the full page HTML.
func (c *Context) HTML() (string, error) {
	return c.page.HTML()
}

// Page returns the underlying Page for advanced usage.
func (c *Context) Page() *Page {
	return c.page
}
