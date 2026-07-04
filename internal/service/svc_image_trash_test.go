package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/haierkeys/custom-image-gateway/pkg/code"
)

func TestResolveLocalTrashPathsAbsoluteCustomPath(t *testing.T) {
	baseDir := t.TempDir()
	customDir := filepath.Join(baseDir, "vault-assets")
	sourceFile := filepath.Join(customDir, "202607", "04", "test image.png")
	if err := os.MkdirAll(filepath.Dir(sourceFile), os.ModePerm); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	if err := os.WriteFile(sourceFile, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	sourcePath, trashPath, err := resolveLocalTrashPaths("https://img.example.com/files/202607/04/test%20image.png", &localTrashConfig{
		AccessURLPrefix: "https://img.example.com/files",
		SavePath:        filepath.Join(baseDir, "storage", "uploads"),
		CustomPath:      customDir,
	})
	if err != nil {
		t.Fatalf("resolveLocalTrashPaths err: %v", err)
	}

	if sourcePath != sourceFile {
		t.Fatalf("expected source %q, got %q", sourceFile, sourcePath)
	}

	expectedTrash := filepath.Join(customDir, ".trash", "202607", "04", "test image.png")
	if trashPath != expectedTrash {
		t.Fatalf("expected trash %q, got %q", expectedTrash, trashPath)
	}
}

func TestResolveLocalTrashPathsEmptyCustomPath(t *testing.T) {
	baseDir := t.TempDir()
	savePath := filepath.Join(baseDir, "storage", "uploads")
	sourceFile := filepath.Join(savePath, "202607", "04", "test image.png")
	if err := os.MkdirAll(filepath.Dir(sourceFile), os.ModePerm); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	if err := os.WriteFile(sourceFile, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	sourcePath, trashPath, err := resolveLocalTrashPaths("https://img.example.com/storage/uploads/202607/04/test%20image.png", &localTrashConfig{
		AccessURLPrefix: "https://img.example.com",
		SavePath:        savePath,
	})
	if err != nil {
		t.Fatalf("resolveLocalTrashPaths err: %v", err)
	}

	if sourcePath != sourceFile {
		t.Fatalf("expected source %q, got %q", sourceFile, sourcePath)
	}

	expectedTrash := filepath.Join(savePath, ".trash", "202607", "04", "test image.png")
	if trashPath != expectedTrash {
		t.Fatalf("expected trash %q, got %q", expectedTrash, trashPath)
	}
}

func TestResolveLocalTrashPathsPrefixMismatch(t *testing.T) {
	_, _, err := resolveLocalTrashPaths("https://cdn.example.com/storage/uploads/202607/04/test%20image.png", &localTrashConfig{
		AccessURLPrefix: "https://img.example.com",
		SavePath:        "storage/uploads",
	})
	if err == nil || err != code.ErrorImageURLPrefixMismatch {
		t.Fatalf("expected ErrorImageURLPrefixMismatch, got %v", err)
	}
}
