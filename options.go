package browse

import "time"

// options holds configuration for the Engine.
type options struct {
	headless        bool
	timeout         time.Duration
	slowmo          time.Duration
	width           int
	height          int
	userAgent       string
	proxyServer     string
	poolSize        int
	allowPrivateIPs bool
	remoteCDP       string
}

func defaultOptions() options {
	return options{
		headless: true,
		timeout:  30 * time.Second,
		width:    1280,
		height:   720,
	}
}

// Option configures an Engine.
type Option func(*options)

func applyOptions(opts []Option) options {
	o := defaultOptions()
	for _, fn := range opts {
		fn(&o)
	}
	return o
}

// WithHeadless sets whether the browser runs in headless mode.
func WithHeadless(h bool) Option {
	return func(o *options) { o.headless = h }
}

// WithTimeout sets the default timeout for browser operations.
func WithTimeout(d time.Duration) Option {
	return func(o *options) { o.timeout = d }
}

// WithSlowMotion adds artificial delay between actions.
func WithSlowMotion(d time.Duration) Option {
	return func(o *options) { o.slowmo = d }
}

// WithViewport sets the browser viewport dimensions.
func WithViewport(width, height int) Option {
	return func(o *options) {
		o.width = width
		o.height = height
	}
}

// WithUserAgent sets a custom User-Agent string for all pages.
func WithUserAgent(ua string) Option {
	return func(o *options) { o.userAgent = ua }
}

// WithProxy routes browser traffic through the specified proxy server.
// Format: "http://host:port" or "socks5://host:port".
func WithProxy(proxy string) Option {
	return func(o *options) { o.proxyServer = proxy }
}

// WithPoolSize sets the number of reusable pages in the page pool.
// When > 0, RunAll executes tasks concurrently up to this limit.
// Default 0 means sequential execution with no pooling.
func WithPoolSize(n int) Option {
	return func(o *options) { o.poolSize = n }
}

// WithRemoteCDP connects to an already-running Chrome instance via WebSocket URL
// instead of launching a local browser. Use with Browserbase, Steel, or self-hosted Chrome.
//
//	engine := browse.New(browse.WithRemoteCDP("ws://localhost:9222/devtools/browser/..."))
func WithRemoteCDP(wsURL string) Option {
	return func(o *options) { o.remoteCDP = wsURL }
}

// WithAllowPrivateIPs permits navigation to private/loopback IP addresses.
// By default, navigation to private IPs is blocked to prevent SSRF.
// Enable for testing or internal network automation.
func WithAllowPrivateIPs(allow bool) Option {
	return func(o *options) { o.allowPrivateIPs = allow }
}
