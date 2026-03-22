package agent

import (
	"encoding/json"
	"fmt"
	"time"
)

// DismissCookieBanner attempts to find and dismiss common cookie consent banners.
// Tries common selectors and text patterns. Returns whether a banner was found and dismissed.
func (s *Session) DismissCookieBanner() (*CookieDismissResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensurePage(); err != nil {
		return nil, err
	}

	js := `(function() {
		// Common accept button selectors (ordered by specificity)
		const selectors = [
			// ID-based
			'#accept-cookies', '#cookie-accept', '#onetrust-accept-btn-handler',
			'#CybotCookiebotDialogBodyLevelButtonLevelOptinAllowAll',
			'#truste-consent-button', '#didomi-notice-agree-button',
			'#cookiescript_accept', '#cookie_action_close_header',
			// Class-based
			'.cookie-accept', '.accept-cookies', '.js-cookie-accept',
			'.cc-accept', '.cc-btn.cc-allow', '.cc-dismiss',
			'.cookie-consent-accept', '.cookie-notice-accept',
			'.gdpr-accept', '.consent-accept',
			// Data attributes
			'[data-cookie-accept]', '[data-consent="accept"]',
			'[data-action="accept"]', '[data-testid="cookie-accept"]',
			// aria labels
			'[aria-label="Accept cookies"]', '[aria-label="Accept all cookies"]',
			'[aria-label="Accept all"]', '[aria-label="Allow all"]',
		];

		// Try direct selectors first
		for (const sel of selectors) {
			const btn = document.querySelector(sel);
			if (btn && btn.offsetParent !== null) {
				btn.click();
				return JSON.stringify({found: true, method: 'selector', selector: sel, text: btn.textContent.trim().slice(0, 50)});
			}
		}

		// Try text-based search on buttons and links
		const textPatterns = [
			/^accept\s*(all)?\s*(cookies)?$/i,
			/^(i\s+)?agree$/i,
			/^allow\s*(all)?\s*(cookies)?$/i,
			/^got\s+it$/i,
			/^ok(ay)?$/i,
			/^consent$/i,
			/^accept\s*&?\s*close$/i,
			/^(i\s+)?understand$/i,
			/^continue$/i,
		];

		const clickables = document.querySelectorAll('button, a, [role="button"], input[type="button"], input[type="submit"]');
		for (const el of clickables) {
			const text = el.textContent.trim();
			if (text.length > 50) continue;
			for (const pattern of textPatterns) {
				if (pattern.test(text) && el.offsetParent !== null) {
					el.click();
					return JSON.stringify({found: true, method: 'text', text: text});
				}
			}
		}

		// Check if there's a cookie banner at all
		const bannerSelectors = [
			'#cookie-banner', '#cookie-consent', '#cookie-notice',
			'.cookie-banner', '.cookie-consent', '.cookie-notice',
			'#onetrust-banner-sdk', '#CybotCookiebotDialog',
			'[class*="cookie"]', '[id*="cookie"]',
			'[class*="consent"]', '[id*="consent"]',
			'[class*="gdpr"]',
		];
		for (const sel of bannerSelectors) {
			if (document.querySelector(sel)) {
				return JSON.stringify({found: true, method: 'none', banner: sel, text: 'Banner found but no accept button detected'});
			}
		}

		return JSON.stringify({found: false});
	})()`

	result, err := s.page.Evaluate(js)
	if err != nil {
		return nil, err
	}

	str, _ := result.(string)
	var r CookieDismissResult
	_ = json.Unmarshal([]byte(str), &r)

	if r.Found && r.Method != "none" {
		time.Sleep(300 * time.Millisecond) // wait for banner animation
	}

	return &r, nil
}

// CookieDismissResult describes the outcome of cookie banner dismissal.
type CookieDismissResult struct {
	Found    bool   `json:"found"`
	Method   string `json:"method,omitempty"`   // "selector", "text", "none"
	Selector string `json:"selector,omitempty"` // which selector matched
	Text     string `json:"text,omitempty"`     // button text that was clicked
	Banner   string `json:"banner,omitempty"`   // banner selector if found but not dismissed
}

// NavigateAndDismissCookies navigates to a URL and auto-dismisses any cookie banner.
func (s *Session) NavigateAndDismissCookies(url string) (*PageResult, error) {
	result, err := s.Navigate(url)
	if err != nil {
		return nil, err
	}
	_, _ = s.DismissCookieBanner()
	return result, nil
}

// AutoDismissCookies wraps Navigate to always dismiss cookie banners.
func (s *Session) autoNavigate(url string) (*PageResult, error) {
	result, err := s.Navigate(url)
	if err != nil {
		return nil, err
	}

	// Best-effort cookie dismissal
	dismissResult, _ := s.DismissCookieBanner()
	if dismissResult != nil && dismissResult.Found {
		_ = fmt.Sprintf("Cookie banner dismissed: %s", dismissResult.Text)
	}

	return result, nil
}
