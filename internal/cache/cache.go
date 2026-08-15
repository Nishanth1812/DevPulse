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
// restrictive permissions if it does not already exist, and any stale temp
// files left by interrupted writes are swept.
func New(dir string) (*Cache, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("cache: create directory: %w", err)
	}
	c := &Cache{dir: dir}
	if err := c.sweep(); err != nil {
		// Sweeping is best-effort; a failure must not block the cache itself.
		return c, nil
	}
	return c, nil
}

// sweep removes leftover .tmp files (interrupted writes) and entries that
// are already expired, keeping the directory from growing without bound.
func (c *Cache) sweep() error {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return err
	}
	for _, de := range entries {
		name := de.Name()
		if de.IsDir() {
			continue
		}
		path := filepath.Join(c.dir, name)
		if filepath.Ext(name) == ".tmp" {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
			continue
		}
		// Expired entries are removed; the max age is re-read from each file
		// since different commands may use different TTLs.
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var e entry
		if err := json.Unmarshal(data, &e); err != nil {
			// Corrupt entries cannot be trusted; remove them.
			_ = os.Remove(path)
			continue
		}
		// Only remove entries whose TTL has clearly passed (older than the
		// longest supported cache window). Individual commands still enforce
		// their own max age on read.
		if time.Since(e.StoredAt) > 30*24*time.Hour {
			_ = os.Remove(path)
		}
	}
	return nil
}

// Delete removes a cached entry, if present. Missing files are not an error.
func (c *Cache) Delete(repo, headSHA, provider, command string) error {
	path := filepath.Join(c.dir, c.fileName(repo, headSHA, provider, command))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cache: delete entry: %w", err)
	}
	return nil
}

func (c *Cache) fileName(repo, headSHA, provider, command string) string {
	sum := sha256.Sum256([]byte(repo + "\x00" + headSHA + "\x00" + provider + "\x00" + command))
	return fmt.Sprintf("%x.json", sum)
}

// Hash returns a hex SHA-256 of the given parts, used to build cache keys from
// multiple inputs (e.g. all repo HEADs, or HEAD plus plan-file hash) so cached
// entries invalidate when any contributing input changes.
func Hash(parts ...string) string {
	h := sha256.New()
	for i, part := range parts {
		if i > 0 {
			h.Write([]byte{0})
		}
		h.Write([]byte(part))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
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
	if maxAge <= 0 || time.Since(e.StoredAt) > maxAge {
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
		// Remove the temp file so a failed rename cannot leak it.
		_ = os.Remove(tmp)
		return fmt.Errorf("cache: commit entry: %w", err)
	}

	return nil
}
