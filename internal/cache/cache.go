package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// entry is the on-disk representation of a cached response.
type entry struct {
	StoredAt time.Time       `json:"stored_at"`
	Payload  json.RawMessage `json:"payload"`
}

// Cache is a small file-backed store for generated responses, keyed by
// repository name, HEAD SHA, provider, and command name. Entries expire
// after a caller-supplied max age.
type Cache struct {
	dir string
}

// New creates (or opens) a cache rooted at dir. The directory is created with
// restrictive permissions if it does not already exist.
func New(dir string) (*Cache, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("cache: create directory: %w", err)
	}
	return &Cache{dir: dir}, nil
}

func (c *Cache) fileName(repo, headSHA, provider, command string) string {
	sum := sha256.Sum256([]byte(repo + "\x00" + headSHA + "\x00" + provider + "\x00" + command))
	return fmt.Sprintf("%x.json", sum)
}

// GetRaw returns the cached raw JSON payload if one exists and is newer than maxAge.
// The boolean is false on miss, expired entry, or any read/decode error.
func (c *Cache) GetRaw(repo, headSHA, provider, command string, maxAge time.Duration) (json.RawMessage, bool) {
	path := filepath.Join(c.dir, c.fileName(repo, headSHA, provider, command))

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var e entry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, false
	}
	if time.Since(e.StoredAt) > maxAge {
		return nil, false
	}

	return e.Payload, true
}

// PutRaw stores a raw JSON payload, writing atomically via a temp file + rename.
func (c *Cache) PutRaw(repo, headSHA, provider, command string, payload json.RawMessage) error {
	e := entry{StoredAt: time.Now(), Payload: payload}

	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("cache: encode entry: %w", err)
	}

	path := filepath.Join(c.dir, c.fileName(repo, headSHA, provider, command))
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("cache: write entry: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("cache: commit entry: %w", err)
	}

	return nil
}
