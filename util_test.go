package browse

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeBase64(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"valid", "aGVsbG8=", "hello", false},
		{"empty", "", "", false},
		{"padding", "YQ==", "a", false},
		{"invalid", "!!!invalid!!!", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeBase64(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("decodeBase64(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestWriteFile(t *testing.T) {
	t.Run("writes file successfully", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.txt")
		data := []byte("hello world")

		err := writeFile(path, data)
		if err != nil {
			t.Fatalf("writeFile() error: %v", err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile() error: %v", err)
		}
		if string(got) != "hello world" {
			t.Errorf("file content = %q, want %q", got, "hello world")
		}
	})

	t.Run("blocks path traversal", func(t *testing.T) {
		err := writeFile("../../../etc/passwd", []byte("bad"))
		if err == nil {
			t.Error("expected error for path traversal")
		}
		if err != nil && !strings.Contains(err.Error(), "path traversal") && !strings.Contains(err.Error(), "refusing") {
			t.Errorf("error should mention path traversal or refusing, got: %v", err)
		}
	})

	t.Run("blocks system path", func(t *testing.T) {
		if err := writeFile("/etc/cron.d/scout", []byte("bad")); err == nil {
			t.Fatal("expected error for /etc write")
		}
	})

	t.Run("empty data", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "empty.txt")

		err := writeFile(path, []byte{})
		if err != nil {
			t.Fatalf("writeFile() error: %v", err)
		}

		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Stat() error: %v", err)
		}
		if info.Size() != 0 {
			t.Errorf("file size = %d, want 0", info.Size())
		}
	})
}

func TestSanitizeCapturePath(t *testing.T) {
	got, err := SanitizeCapturePath("", "screenshot", "png")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, os.TempDir()) {
		t.Errorf("empty path should resolve under temp, got %q", got)
	}

	rel, err := SanitizeCapturePath("scout-rel/shot.png", "screenshot", "png")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rel, os.TempDir()) {
		t.Errorf("relative path should resolve under temp, got %q", rel)
	}

	if _, err := SanitizeCapturePath("../../../etc/cron.d/x", "screenshot", "png"); err == nil {
		t.Fatal("relative traversal must be rejected")
	}

	abs, err := SanitizeCapturePath(filepath.Join(t.TempDir(), "shot.png"), "screenshot", "png")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(abs) {
		t.Errorf("absolute path should stay absolute, got %q", abs)
	}
}

func TestJsonQuote(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "hello", `"hello"`},
		{"empty", "", `""`},
		{"quotes", `say "hi"`, `"say \"hi\""`},
		{"backslash", `a\b`, `"a\\b"`},
		{"newline", "line1\nline2", `"line1\nline2"`},
		{"tab", "a\tb", `"a\tb"`},
		{"unicode", "hello\u0000world", `"hello\u0000world"`},
		{"html special", "<script>", `"\u003cscript\u003e"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jsonQuote(tt.in)
			if got != tt.want {
				t.Errorf("jsonQuote(%q) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}
