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
		AccessURLPrefix: "https://img.example.com/storage/uploads",
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

func TestRestoreLocalImage(t *testing.T) {
	baseDir := t.TempDir()
	savePath := filepath.Join(baseDir, "storage", "uploads")
	trashFile := filepath.Join(savePath, ".trash", "202607", "04", "test image.png")
	if err := os.MkdirAll(filepath.Dir(trashFile), os.ModePerm); err != nil {
		t.Fatalf("mkdir trash dir: %v", err)
	}
	if err := os.WriteFile(trashFile, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write trash file: %v", err)
	}

	result, err := restoreLocalImage(&TrashImageParams{
		ImageURL: "https://img.example.com/storage/uploads/202607/04/test%20image.png",
	}, &localTrashConfig{
		AccessURLPrefix: "https://img.example.com/storage/uploads",
		SavePath:        savePath,
	})
	if err != nil {
		t.Fatalf("restoreLocalImage err: %v", err)
	}

	expectedSource := filepath.Join(savePath, "202607", "04", "test image.png")
	if result.SourcePath != expectedSource {
		t.Fatalf("expected source %q, got %q", expectedSource, result.SourcePath)
	}
	if _, err := os.Stat(expectedSource); err != nil {
		t.Fatalf("expected restored source to exist: %v", err)
	}
}

func TestRestoreLocalImageFindsTimestampedTrashFile(t *testing.T) {
	baseDir := t.TempDir()
	savePath := filepath.Join(baseDir, "storage", "uploads")
	timestampedTrashFile := filepath.Join(savePath, ".trash", "202607", "04", "test image-1720000000.png")
	if err := os.MkdirAll(filepath.Dir(timestampedTrashFile), os.ModePerm); err != nil {
		t.Fatalf("mkdir trash dir: %v", err)
	}
	if err := os.WriteFile(timestampedTrashFile, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write trash file: %v", err)
	}

	result, err := restoreLocalImage(&TrashImageParams{
		ImageURL: "https://img.example.com/storage/uploads/202607/04/test%20image.png",
	}, &localTrashConfig{
		AccessURLPrefix: "https://img.example.com/storage/uploads",
		SavePath:        savePath,
	})
	if err != nil {
		t.Fatalf("restoreLocalImage err: %v", err)
	}

	if result.TrashPath != timestampedTrashFile {
		t.Fatalf("expected trash path %q, got %q", timestampedTrashFile, result.TrashPath)
	}
}

func TestRestoreLocalImageConflict(t *testing.T) {
	baseDir := t.TempDir()
	savePath := filepath.Join(baseDir, "storage", "uploads")
	sourceFile := filepath.Join(savePath, "202607", "04", "test image.png")
	trashFile := filepath.Join(savePath, ".trash", "202607", "04", "test image.png")
	if err := os.MkdirAll(filepath.Dir(sourceFile), os.ModePerm); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(trashFile), os.ModePerm); err != nil {
		t.Fatalf("mkdir trash dir: %v", err)
	}
	if err := os.WriteFile(sourceFile, []byte("source"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	if err := os.WriteFile(trashFile, []byte("trash"), 0o644); err != nil {
		t.Fatalf("write trash file: %v", err)
	}

	_, err := restoreLocalImage(&TrashImageParams{
		ImageURL: "https://img.example.com/storage/uploads/202607/04/test%20image.png",
	}, &localTrashConfig{
		AccessURLPrefix: "https://img.example.com/storage/uploads",
		SavePath:        savePath,
	})
	if err == nil || err != code.ErrorImageRestoreConflict {
		t.Fatalf("expected ErrorImageRestoreConflict, got %v", err)
	}
}

func TestRestoreLocalImageTrashNotFound(t *testing.T) {
	baseDir := t.TempDir()
	savePath := filepath.Join(baseDir, "storage", "uploads")

	_, err := restoreLocalImage(&TrashImageParams{
		ImageURL: "https://img.example.com/storage/uploads/202607/04/test%20image.png",
	}, &localTrashConfig{
		AccessURLPrefix: "https://img.example.com/storage/uploads",
		SavePath:        savePath,
	})
	if err == nil || err != code.ErrorImageTrashFileNotFound {
		t.Fatalf("expected ErrorImageTrashFileNotFound, got %v", err)
	}
}
