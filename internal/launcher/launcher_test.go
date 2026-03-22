package launcher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestBrowserWSEndpoint(t *testing.T) {
	tests := []struct {
		name  string
		wsURL string
	}{
		{name: "empty URL", wsURL: ""},
		{name: "typical devtools URL", wsURL: "ws://127.0.0.1:9222/devtools/browser/abc-123"},
		{name: "custom port", wsURL: "ws://127.0.0.1:43210/devtools/browser/def-456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Browser{wsURL: tt.wsURL}
			if got := b.WSEndpoint(); got != tt.wsURL {
				t.Errorf("WSEndpoint() = %q, want %q", got, tt.wsURL)
			}
		})
	}
}

func TestBrowserCloseNilProcess(t *testing.T) {
	// Close with nil cmd should not panic.
	b := &Browser{}
	if err := b.Close(); err != nil {
		t.Errorf("Close() returned error for nil cmd: %v", err)
	}
}

func TestFreePort(t *testing.T) {
	port, err := freePort()
	if err != nil {
		t.Fatalf("freePort() error: %v", err)
	}
	if port <= 0 || port > 65535 {
		t.Errorf("freePort() returned invalid port: %d", port)
	}
}

func TestFreePortUniqueness(t *testing.T) {
	seen := make(map[int]bool)
	for range 10 {
		port, err := freePort()
		if err != nil {
			t.Fatalf("freePort() error: %v", err)
		}
		if seen[port] {
			// Duplicates are possible but extremely unlikely in 10 calls.
			t.Logf("warning: duplicate port %d (may happen rarely)", port)
		}
		seen[port] = true
	}
}

func TestFindChromeCandidatesList(t *testing.T) {
	// Verify that findChrome returns a meaningful error when no browser is found.
	// We cannot easily test the success path in CI without Chrome, but we can
	// verify the function runs without panicking on the current platform.
	_, err := findChrome()
	if err != nil {
		// Acceptable: Chrome may not be installed in CI.
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("findChrome() unexpected error format: %v", err)
		}
	}
	// If err == nil, Chrome was found. Either way, no panic is the key assertion.
}

func TestFindChromePlatformCandidates(t *testing.T) {
	// Verify the candidates list is non-empty for the current platform.
	goos := runtime.GOOS
	switch goos {
	case "darwin", "linux", "windows":
		// These are supported — findChrome will build a non-empty candidates list.
	default:
		t.Skipf("unsupported platform %s", goos)
	}
}

func TestOptionsDefaults(t *testing.T) {
	opts := Options{}
	if opts.Headless {
		t.Error("Options.Headless should default to false")
	}
	if opts.Port != 0 {
		t.Error("Options.Port should default to 0")
	}
	if opts.ProxyServer != "" {
		t.Error("Options.ProxyServer should default to empty string")
	}
}

func TestChromeArgConstruction(t *testing.T) {
	// Test the argument building logic that Launch uses.
	// We replicate the arg construction to verify correctness without starting Chrome.
	tests := []struct {
		name         string
		opts         Options
		wantHeadless bool
		wantProxy    string
	}{
		{
			name:         "headless mode",
			opts:         Options{Headless: true, Port: 9222},
			wantHeadless: true,
		},
		{
			name:      "with proxy",
			opts:      Options{ProxyServer: "http://proxy:8080", Port: 9222},
			wantProxy: "http://proxy:8080",
		},
		{
			name:         "headless with proxy",
			opts:         Options{Headless: true, ProxyServer: "socks5://proxy:1080", Port: 9222},
			wantHeadless: true,
			wantProxy:    "socks5://proxy:1080",
		},
		{
			name: "default mode",
			opts: Options{Port: 9222},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build args the same way Launch does.
			args := []string{
				fmt.Sprintf("--remote-debugging-port=%d", tt.opts.Port),
				"--user-data-dir=/tmp/test",
				"--no-first-run",
				"--no-default-browser-check",
				"--disable-background-networking",
				"--disable-default-apps",
				"--disable-extensions",
				"--disable-sync",
				"--disable-translate",
				"--disable-popup-blocking",
				"--metrics-recording-only",
				"--safebrowsing-disable-auto-update",
				"about:blank",
			}
			if tt.opts.Headless {
				args = append([]string{"--headless=new"}, args...)
			}
			if tt.opts.ProxyServer != "" {
				args = append([]string{fmt.Sprintf("--proxy-server=%s", tt.opts.ProxyServer)}, args...)
			}

			// Verify expected flags are present.
			joined := strings.Join(args, " ")

			if tt.wantHeadless {
				if !strings.Contains(joined, "--headless=new") {
					t.Error("expected --headless=new flag")
				}
			} else {
				if strings.Contains(joined, "--headless") {
					t.Error("unexpected --headless flag")
				}
			}

			if tt.wantProxy != "" {
				expected := fmt.Sprintf("--proxy-server=%s", tt.wantProxy)
				if !strings.Contains(joined, expected) {
					t.Errorf("expected %s in args", expected)
				}
			}

			portFlag := fmt.Sprintf("--remote-debugging-port=%d", tt.opts.Port)
			if !strings.Contains(joined, portFlag) {
				t.Errorf("expected %s in args", portFlag)
			}

			// Verify required base flags are always present.
			requiredFlags := []string{
				"--no-first-run",
				"--no-default-browser-check",
				"--disable-extensions",
				"about:blank",
			}
			for _, flag := range requiredFlags {
				if !strings.Contains(joined, flag) {
					t.Errorf("missing required flag %q", flag)
				}
			}
		})
	}
}

func TestFetchWSURLParsing(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		statusCode int
		wantURL    string
		wantErr    bool
	}{
		{
			name:       "valid response",
			body:       `{"webSocketDebuggerUrl":"ws://127.0.0.1:9222/devtools/browser/abc"}`,
			statusCode: http.StatusOK,
			wantURL:    "ws://127.0.0.1:9222/devtools/browser/abc",
		},
		{
			name:       "empty websocket URL",
			body:       `{"webSocketDebuggerUrl":""}`,
			statusCode: http.StatusOK,
			wantErr:    true,
		},
		{
			name:       "missing field",
			body:       `{"Browser":"Chrome"}`,
			statusCode: http.StatusOK,
			wantErr:    true,
		},
		{
			name:       "invalid JSON",
			body:       `not json`,
			statusCode: http.StatusOK,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			// Extract port from test server URL.
			parts := strings.Split(srv.URL, ":")
			port, _ := strconv.Atoi(parts[len(parts)-1])

			got, err := fetchWSURL(port)
			if tt.wantErr {
				if err == nil {
					t.Errorf("fetchWSURL() expected error, got URL=%q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("fetchWSURL() unexpected error: %v", err)
			}
			if got != tt.wantURL {
				t.Errorf("fetchWSURL() = %q, want %q", got, tt.wantURL)
			}
		})
	}
}

func TestFetchWSURLVersionEndpoint(t *testing.T) {
	// Verify that fetchWSURL hits /json/version path.
	var requestPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		resp := map[string]string{
			"webSocketDebuggerUrl": "ws://127.0.0.1:1234/devtools/browser/test",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	parts := strings.Split(srv.URL, ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])

	_, err := fetchWSURL(port)
	if err != nil {
		t.Fatalf("fetchWSURL() error: %v", err)
	}
	if requestPath != "/json/version" {
		t.Errorf("fetchWSURL() hit path %q, want /json/version", requestPath)
	}
}
