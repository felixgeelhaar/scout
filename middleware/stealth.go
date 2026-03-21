package middleware

import (
	browse "github.com/felixgeelhaar/scout"
)

// Stealth returns middleware that applies anti-detection patches to avoid bot fingerprinting.
// It overrides common automation signals that websites use to detect headless Chrome:
//   - navigator.webdriver = false
//   - chrome.runtime injection
//   - Permissions API normalization
//   - Plugin/language spoofing
//   - WebGL vendor/renderer masking
//
// Apply this middleware globally via engine.Use(middleware.Stealth()).
func Stealth() browse.HandlerFunc {
	return func(c *browse.Context) {
		page := c.Page()
		if page == nil {
			c.Next()
			return
		}

		// Inject stealth patches before any page load
		_, _ = page.Call("Page.addScriptToEvaluateOnNewDocument", map[string]any{
			"source": stealthJS,
		})

		c.Next()
	}
}

const stealthJS = `
// 1. Override navigator.webdriver
Object.defineProperty(navigator, 'webdriver', {
	get: () => false,
	configurable: true
});

// 2. Mock chrome.runtime to pass chrome.runtime check
if (!window.chrome) window.chrome = {};
if (!window.chrome.runtime) {
	window.chrome.runtime = {
		connect: function() {},
		sendMessage: function() {},
		id: undefined
	};
}

// 3. Override Permissions API to deny 'notifications' query gracefully
if (navigator.permissions) {
	const originalQuery = navigator.permissions.query.bind(navigator.permissions);
	navigator.permissions.query = function(parameters) {
		if (parameters.name === 'notifications') {
			return Promise.resolve({ state: Notification.permission });
		}
		return originalQuery(parameters);
	};
}

// 4. Override navigator.plugins to appear non-empty
Object.defineProperty(navigator, 'plugins', {
	get: () => {
		const plugins = [
			{ name: 'Chrome PDF Plugin', filename: 'internal-pdf-viewer', description: 'Portable Document Format' },
			{ name: 'Chrome PDF Viewer', filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai', description: '' },
			{ name: 'Native Client', filename: 'internal-nacl-plugin', description: '' }
		];
		plugins.length = 3;
		return plugins;
	},
	configurable: true
});

// 5. Override navigator.languages
Object.defineProperty(navigator, 'languages', {
	get: () => ['en-US', 'en'],
	configurable: true
});

// 6. Mask WebGL vendor/renderer
const getParameter = WebGLRenderingContext.prototype.getParameter;
WebGLRenderingContext.prototype.getParameter = function(parameter) {
	if (parameter === 37445) return 'Intel Inc.';
	if (parameter === 37446) return 'Intel Iris OpenGL Engine';
	return getParameter.call(this, parameter);
};

// 7. Fix broken iframe contentWindow
const origAttachShadow = Element.prototype.attachShadow;
Element.prototype.attachShadow = function(init) {
	if (init && init.mode) return origAttachShadow.call(this, init);
	return origAttachShadow.call(this, { mode: 'open' });
};

// 8. Remove automation-related properties
delete navigator.__proto__.webdriver;

// 9. Fix window.outerWidth/outerHeight (headless has 0)
if (window.outerWidth === 0) {
	Object.defineProperty(window, 'outerWidth', { get: () => window.innerWidth });
	Object.defineProperty(window, 'outerHeight', { get: () => window.innerHeight + 85 });
}

// 10. Fix missing screen properties in headless
if (screen.availWidth === 0) {
	Object.defineProperty(screen, 'availWidth', { get: () => screen.width });
	Object.defineProperty(screen, 'availHeight', { get: () => screen.height - 40 });
}
`
