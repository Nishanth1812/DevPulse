package services

import (
	"strings"
	"testing"

	"github.com/Nishanth1812/devpulse/internal/config"
)

func loadManager(t *testing.T) {
	t.Helper()
	t.Setenv("DEVPULSE_CONFIG", t.TempDir()+"/config.toml")
	if _, err := config.Load(); err != nil {
		t.Fatalf("config.Load: %v", err)
	}
}

func TestNoteListMissingFileReturnsEmpty(t *testing.T) {
	loadManager(t)

	s := &NoteService{}
	// Repo with no notes file yet.
	content, err := s.List("no-notes-repo")
	if err != nil {
		t.Fatalf("List on missing notes file should not error, got %v", err)
	}
	if content != "" {
		t.Fatalf("expected empty content, got %q", content)
	}
}

func TestNoteAddThenListRoundTrip(t *testing.T) {
	loadManager(t)

	s := &NoteService{}
	if err := s.Add("some-repo", "first note"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Add("some-repo", "second note"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	content, err := s.List("some-repo")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if !strings.Contains(content, "first note") || !strings.Contains(content, "second note") {
		t.Fatalf("notes not both present: %q", content)
	}
}

func TestNoteSanitizeRejectsTraversal(t *testing.T) {
	s := &NoteService{}
	if err := s.Add("../../etc/passwd", "pwned"); err == nil {
		t.Fatal("Add with traversal name should error")
	}
}
