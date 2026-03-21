package browse

import (
	"fmt"
	"net"
	"net/url"
)

// allowedSchemes for navigation. Blocks file://, javascript:, data:, chrome:// etc.
var allowedSchemes = map[string]bool{"http": true, "https": true}

// URLValidator controls URL validation for navigation.
// Set AllowPrivateIPs to true to permit loopback/private IP navigation (e.g., for testing).
type URLValidator struct {
	AllowPrivateIPs bool
}

// DefaultURLValidator blocks private IPs and non-http(s) schemes.
var DefaultURLValidator = URLValidator{AllowPrivateIPs: false}

// Validate checks that a URL is safe for navigation.
// Blocks non-http(s) schemes and private/loopback IPs (unless AllowPrivateIPs is set).
func (v URLValidator) Validate(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("browse: invalid URL: %w", err)
	}
	if !allowedSchemes[u.Scheme] {
		return fmt.Errorf("browse: blocked URL scheme %q (only http/https allowed)", u.Scheme)
	}
	if !v.AllowPrivateIPs {
		host := u.Hostname()
		if ip := net.ParseIP(host); ip != nil {
			if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
				return fmt.Errorf("browse: blocked navigation to private/loopback IP %s", ip)
			}
		}
	}
	return nil
}
