package browse

import (
	"strings"
	"testing"
)

func TestURLValidator_ValidURLs(t *testing.T) {
	v := URLValidator{AllowPrivateIPs: false}

	tests := []struct {
		name string
		url  string
	}{
		{"http scheme", "http://example.com"},
		{"https scheme", "https://example.com"},
		{"https with path", "https://example.com/page"},
		{"https with query", "https://example.com/search?q=go"},
		{"https with fragment", "https://example.com/page#section"},
		{"https with port", "https://example.com:8080/api"},
		{"http subdomain", "http://sub.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := v.Validate(tt.url); err != nil {
				t.Errorf("Validate(%q) returned error: %v", tt.url, err)
			}
		})
	}
}

func TestURLValidator_BlockedSchemes(t *testing.T) {
	v := URLValidator{AllowPrivateIPs: false}

	tests := []struct {
		name string
		url  string
	}{
		{"file scheme", "file:///etc/passwd"},
		{"javascript scheme", "javascript:alert(1)"},
		{"data scheme", "data:text/html,<h1>hi</h1>"},
		{"ftp scheme", "ftp://example.com/file"},
		{"chrome scheme", "chrome://settings"},
		{"about scheme", "about:blank"},
		{"empty scheme", "://no-scheme"}, // fails at URL parse, not scheme check
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.url)
			if err == nil {
				t.Errorf("Validate(%q) should have returned error", tt.url)
			}
		})
	}
}

func TestURLValidator_PrivateIPsBlocked(t *testing.T) {
	v := URLValidator{AllowPrivateIPs: false}

	tests := []struct {
		name string
		url  string
	}{
		{"loopback IPv4", "http://127.0.0.1"},
		{"loopback IPv4 with port", "http://127.0.0.1:8080"},
		{"private 10.x", "http://10.0.0.1"},
		{"private 172.16.x", "http://172.16.0.1"},
		{"private 192.168.x", "http://192.168.1.1"},
		{"loopback IPv6", "http://[::1]"},
		{"link-local IPv4", "http://169.254.1.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.url)
			if err == nil {
				t.Errorf("Validate(%q) should block private IP", tt.url)
			}
			if err != nil && !strings.Contains(err.Error(), "blocked navigation to private") {
				t.Errorf("Validate(%q) error should mention private IP, got: %v", tt.url, err)
			}
		})
	}
}

func TestURLValidator_PrivateIPsAllowed(t *testing.T) {
	v := URLValidator{AllowPrivateIPs: true}

	tests := []struct {
		name string
		url  string
	}{
		{"loopback", "http://127.0.0.1"},
		{"loopback with port", "http://127.0.0.1:3000"},
		{"private 10.x", "http://10.0.0.1"},
		{"private 192.168.x", "http://192.168.1.1"},
		{"loopback IPv6", "http://[::1]"},
		{"localhost by IP", "http://127.0.0.1:8080/api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := v.Validate(tt.url); err != nil {
				t.Errorf("Validate(%q) with AllowPrivateIPs=true returned error: %v", tt.url, err)
			}
		})
	}
}

func TestURLValidator_PublicIPsAlwaysAllowed(t *testing.T) {
	v := URLValidator{AllowPrivateIPs: false}

	tests := []struct {
		name string
		url  string
	}{
		{"public IP", "http://8.8.8.8"},
		{"public IP with path", "https://1.1.1.1/dns-query"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := v.Validate(tt.url); err != nil {
				t.Errorf("Validate(%q) should allow public IP, got: %v", tt.url, err)
			}
		})
	}
}

func TestURLValidator_HostnamesNotBlockedAsIPs(t *testing.T) {
	v := URLValidator{AllowPrivateIPs: false}

	tests := []struct {
		name string
		url  string
	}{
		{"localhost hostname", "http://localhost"},
		{"localhost with port", "http://localhost:3000"},
		{"domain name", "https://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := v.Validate(tt.url); err != nil {
				t.Errorf("Validate(%q) returned error: %v", tt.url, err)
			}
		})
	}
}

func TestDefaultURLValidator(t *testing.T) {
	if DefaultURLValidator.AllowPrivateIPs {
		t.Error("DefaultURLValidator should have AllowPrivateIPs=false")
	}
	if err := DefaultURLValidator.Validate("https://example.com"); err != nil {
		t.Errorf("DefaultURLValidator should allow public URLs: %v", err)
	}
	if err := DefaultURLValidator.Validate("http://127.0.0.1"); err == nil {
		t.Error("DefaultURLValidator should block private IPs")
	}
}
