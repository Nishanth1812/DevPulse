package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestCache(t *testing.T) *Cache {
	t.Helper()
	c, err := New(filepath.Join(t.TempDir(), "cache"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestPutGetRaw(t *testing.T) {
	c := newTestCache(t)
	payload := json.RawMessage(`{"a":1}`)
	if err := c.PutRaw("repo", "sha", "groq", "brief", payload); err != nil {
		t.Fatalf("PutRaw: %v", err)
	}
	got, ok := c.GetRaw("repo", "sha", "groq", "brief", time.Hour)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(got) != string(payload) {
		t.Fatalf("payload mismatch: %s != %s", got, payload)
	}
}

func TestGetRawMiss(t *testing.T) {
	c := newTestCache(t)
	if _, ok := c.GetRaw("repo", "sha", "groq", "brief", time.Hour); ok {
		t.Fatal("expected cache miss")
	}
}

func TestGetRawExpired(t *testing.T) {
	c := newTestCache(t)
	if err := c.PutRaw("repo", "sha", "groq", "brief", json.RawMessage(`{}`)); err != nil {
		t.Fatalf("PutRaw: %v", err)
	}
	if _, ok := c.GetRaw("repo", "sha", "groq", "brief", 0); ok {
		t.Fatal("expected expiry miss for zero max age")
	}
}

func TestDelete(t *testing.T) {
	c := newTestCache(t)
	if err := c.PutRaw("repo", "sha", "groq", "brief", json.RawMessage(`{}`)); err != nil {
		t.Fatalf("PutRaw: %v", err)
	}
	if err := c.Delete("repo", "sha", "groq", "brief"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok := c.GetRaw("repo", "sha", "groq", "brief", time.Hour); ok {
		t.Fatal("expected miss after delete")
	}
	if err := c.Delete("repo", "sha", "groq", "brief"); err != nil {
		t.Fatalf("Delete missing entry should not error: %v", err)
	}
}

func TestNewSweepsTmpFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cache")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "stale.json.tmp"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "stale.json.tmp")); !os.IsNotExist(err) {
		t.Fatalf("expected .tmp file swept, err=%v", err)
	}
	// Directory still usable.
	if err := c.PutRaw("r", "s", "p", "c", json.RawMessage(`{}`)); err != nil {
		t.Fatalf("PutRaw after sweep: %v", err)
	}
}

func TestPutRawCleansTmpOnRenameFailure(t *testing.T) {
	// Force rename failure by making the final path a directory.
	dir := filepath.Join(t.TempDir(), "cache")
	cc, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	finalPath := filepath.Join(dir, cc.fileName("r", "s", "p", "c"))
	if err := os.Mkdir(finalPath, 0o700); err != nil {
		t.Fatal(err)
	}
	err = cc.PutRaw("r", "s", "p", "c", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected rename failure")
	}
	if _, statErr := os.Stat(finalPath + ".tmp"); !os.IsNotExist(statErr) {
		t.Fatalf("expected .tmp cleaned up after rename failure, err=%v", statErr)
	}
}
