package browse

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func writeFile(path string, data []byte) error {
	cleaned := filepath.Clean(path)
	if strings.Contains(cleaned, "..") {
		return fmt.Errorf("browse: path traversal detected in %q", path)
	}
	return os.WriteFile(cleaned, data, 0o600)
}

func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
