package cookies

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

type Cookier interface {
	LoadCookies() ([]byte, error)
	SaveCookies(data []byte) error
	DeleteCookies() error
}

type localCookie struct {
	path string
}

func NewLoadCookie(path string) Cookier {
	if path == "" {
		panic("path is required")
	}
	return &localCookie{path: path}
}

func (c *localCookie) LoadCookies() ([]byte, error) {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read cookies from file")
	}
	return data, nil
}

func (c *localCookie) SaveCookies(data []byte) error {
	return os.WriteFile(c.path, data, 0644)
}

func (c *localCookie) DeleteCookies() error {
	if _, err := os.Stat(c.path); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(c.path)
}

// GetCookiesFilePath returns the cookies file path.
// Priority: $COOKIES_PATH env > ~/.xhs-cli/cookies.json > /tmp/cookies.json (legacy)
func GetCookiesFilePath() string {
	if path := os.Getenv("COOKIES_PATH"); path != "" {
		return path
	}

	// Use ~/.xhs-cli/cookies.json as default
	home, err := os.UserHomeDir()
	if err == nil {
		dir := filepath.Join(home, ".xhs-cli")
		_ = os.MkdirAll(dir, 0755)
		return filepath.Join(dir, "cookies.json")
	}

	// Legacy fallback
	tmpDir := os.TempDir()
	oldPath := filepath.Join(tmpDir, "cookies.json")
	if _, err := os.Stat(oldPath); err == nil {
		return oldPath
	}

	return "cookies.json"
}
