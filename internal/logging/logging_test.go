package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRotatingWriter_RotatesAndKeepsLimit(t *testing.T) {
	dir := t.TempDir()
	w, err := NewRotatingWriter(dir, 200, 3)
	if err != nil {
		t.Fatalf("new writer: %v", err)
	}
	defer w.Close()

	chunk := strings.Repeat("a", 100) + "\n"
	for i := 0; i < 8; i++ {
		if _, err := w.Write([]byte(chunk)); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 log files, got %d", len(entries))
	}
	for _, name := range []string{"moments.log", "moments.1.log", "moments.2.log"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("expected %s: %v", name, err)
		}
	}
}

func TestRotatingWriter_AppendsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	w, err := NewRotatingWriter(dir, 1<<20, 7)
	if err != nil {
		t.Fatalf("new writer: %v", err)
	}
	if _, err := w.Write([]byte("first\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	w2, err := NewRotatingWriter(dir, 1<<20, 7)
	if err != nil {
		t.Fatalf("reopen writer: %v", err)
	}
	defer w2.Close()
	if _, err := w2.Write([]byte("second\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if string(body) != "first\nsecond\n" {
		t.Fatalf("unexpected log content: %q", body)
	}
}
