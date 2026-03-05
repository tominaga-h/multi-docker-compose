package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenEditor(t *testing.T) {
	t.Run("succeeds with true command as EDITOR", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "test.yml")
		if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		t.Setenv("EDITOR", "true")
		if err := openEditor(tmpFile); err != nil {
			t.Errorf("openEditor() error: %v", err)
		}
	})

	t.Run("returns error when editor fails", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "test.yml")
		if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		t.Setenv("EDITOR", "false")
		err := openEditor(tmpFile)
		if err == nil {
			t.Fatal("openEditor() expected error, got nil")
		}
	})

	t.Run("returns error when editor not found", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "test.yml")
		if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		t.Setenv("EDITOR", "nonexistent-editor-binary-xyz")
		err := openEditor(tmpFile)
		if err == nil {
			t.Fatal("openEditor() expected error, got nil")
		}
	})
}
