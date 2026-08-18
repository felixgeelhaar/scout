package browse

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func writeFile(path string, data []byte) error {
	cleaned, err := SanitizeWritePath(path)
	if err != nil {
		return err
	}
	return os.WriteFile(cleaned, data, 0o600)
}

func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// SanitizeWritePath resolves path to an absolute location that is safe to
// write. It rejects leftover ".." segments, well-known credential paths, and
// system directories. Relative paths resolve against the process working
// directory (library/CLI callers).
func SanitizeWritePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("browse: empty write path")
	}
	cleaned := filepath.Clean(path)
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("browse: path traversal detected in %q", path)
	}
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("browse: %w", err)
	}
	if strings.Contains(path, "..") && !underDir(abs, mustWd()) && !underDir(abs, os.TempDir()) {
		return "", fmt.Errorf("browse: path traversal detected in %q", path)
	}
	if reason, blocked := SensitivePathReason(abs); blocked {
		return "", fmt.Errorf("browse: refusing to write %q (%s)", path, reason)
	}
	return abs, nil
}

// SanitizeCapturePath resolves an agent/MCP-supplied capture destination.
// Empty yields a generated name under the OS temp dir. Relative paths join
// under the temp dir and must not escape it. Absolute paths go through
// SanitizeWritePath.
func SanitizeCapturePath(outputPath, kind, ext string) (string, error) {
	if ext == "" {
		ext = "bin"
	}
	if strings.HasPrefix(ext, ".") {
		ext = strings.TrimPrefix(ext, ".")
	}
	if strings.TrimSpace(outputPath) == "" {
		return filepath.Join(os.TempDir(),
			fmt.Sprintf("scout-%s-%d.%s", kind, time.Now().UnixNano(), ext)), nil
	}
	if !filepath.IsAbs(outputPath) {
		joined := filepath.Clean(filepath.Join(os.TempDir(), outputPath))
		if !underDir(joined, os.TempDir()) {
			return "", fmt.Errorf("browse: path traversal: relative path %q escapes temp dir", outputPath)
		}
		if reason, blocked := SensitivePathReason(joined); blocked {
			return "", fmt.Errorf("browse: refusing to write %q (%s)", outputPath, reason)
		}
		return joined, nil
	}
	return SanitizeWritePath(outputPath)
}

// SanitizeOutputDir resolves an agent-supplied directory for recordings/traces.
// Empty yields the OS temp dir. Relative paths join under the temp dir and
// must not escape it.
func SanitizeOutputDir(dir string) (string, error) {
	if strings.TrimSpace(dir) == "" {
		return os.TempDir(), nil
	}
	if !filepath.IsAbs(dir) {
		joined := filepath.Clean(filepath.Join(os.TempDir(), dir))
		if !underDir(joined, os.TempDir()) {
			return "", fmt.Errorf("browse: path traversal: relative path %q escapes temp dir", dir)
		}
		if reason, blocked := SensitivePathReason(joined); blocked {
			return "", fmt.Errorf("browse: refusing to write %q (%s)", dir, reason)
		}
		return joined, nil
	}
	return SanitizeWritePath(dir)
}

// WriteSecureFile writes data to path with 0600 permissions after creating
// missing parent directories (0750). path should already have been passed
// through SanitizeWritePath or SanitizeCapturePath.
func WriteSecureFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("browse: create output dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("browse: write %s: %w", path, err)
	}
	return nil
}

// SensitivePathReason reports whether absPath names credential material or a
// system directory that must not be read or written by agent-driven file ops.
func SensitivePathReason(absPath string) (string, bool) {
	lower := strings.ToLower(filepath.ToSlash(absPath))
	base := filepath.Base(lower)
	segments := strings.Split(lower, "/")

	sensitiveDirs := map[string]struct{}{
		".ssh": {}, ".gnupg": {}, ".aws": {}, ".azure": {},
		".kube": {}, ".docker": {}, "gcloud": {},
	}
	for _, seg := range segments {
		if _, ok := sensitiveDirs[seg]; ok {
			return "credential directory", true
		}
	}

	sensitiveBases := map[string]struct{}{
		"id_rsa": {}, "id_dsa": {}, "id_ecdsa": {}, "id_ed25519": {},
		".netrc": {}, ".pgpass": {},
		".env": {}, ".env.local": {}, ".env.production": {},
		"credentials.json": {}, ".git-credentials": {},
	}
	if _, ok := sensitiveBases[base]; ok {
		return "credential file", true
	}

	switch lower {
	case "/etc/shadow", "/etc/gshadow", "/etc/sudoers", "/etc/master.passwd":
		return "system secret file", true
	}

	systemPrefixes := []string{
		"/etc/", "/usr/", "/bin/", "/sbin/", "/proc/",
		"/sys/", "/dev/", "/root/", "/boot/",
	}
	for _, p := range systemPrefixes {
		if lower == strings.TrimSuffix(p, "/") || strings.HasPrefix(lower, p) {
			return "system directory", true
		}
	}
	return "", false
}

func underDir(path, dir string) bool {
	absPath, err1 := filepath.Abs(path)
	absDir, err2 := filepath.Abs(dir)
	if err1 != nil || err2 != nil {
		return false
	}
	rel, err := filepath.Rel(absDir, absPath)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func mustWd() string {
	wd, err := os.Getwd()
	if err != nil {
		return os.TempDir()
	}
	return wd
}
