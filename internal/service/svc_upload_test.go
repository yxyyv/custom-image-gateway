package service

import (
	"testing"

	"github.com/haierkeys/custom-image-gateway/global"
	"github.com/haierkeys/custom-image-gateway/internal/dao"
	"github.com/haierkeys/custom-image-gateway/pkg/storage"
)

func loadTestConfig(t *testing.T) {
	t.Helper()
	if global.Config != nil {
		return
	}
	if _, err := global.ConfigLoad("../../config/config.yaml"); err != nil {
		t.Fatalf("load config: %v", err)
	}
}

func TestGetUserLocalSavePath(t *testing.T) {
	loadTestConfig(t)
	global.Config.LocalFS.SavePath = "storage/uploads"

	cfg := &dao.CloudConfig{
		Type:       storage.LOCAL,
		CustomPath: "G:/data/obsidian-images",
	}

	if got := getUserLocalSavePath(cfg); got != "G:/data/obsidian-images" {
		t.Fatalf("expected custom save path, got %q", got)
	}

	cfg.CustomPath = ""
	if got := getUserLocalSavePath(cfg); got != "storage/uploads" {
		t.Fatalf("expected default save path, got %q", got)
	}
}

func TestBuildUserAccessURLForLocalFS(t *testing.T) {
	loadTestConfig(t)
	global.Config.LocalFS.SavePath = "storage/uploads"

	tests := []struct {
		name     string
		cfg      *dao.CloudConfig
		dstFile  string
		fileKey  string
		expected string
	}{
		{
			name: "relative custom path is reflected in URL",
			cfg: &dao.CloudConfig{
				Type:            storage.LOCAL,
				AccessURLPrefix: "https://img.example.com",
				CustomPath:      "vault-assets",
			},
			dstFile:  "ignored",
			fileKey:  "202606/24/test image.png",
			expected: "https://img.example.com/vault-assets/202606/24/test%20image.png",
		},
		{
			name: "absolute custom path does not leak into URL",
			cfg: &dao.CloudConfig{
				Type:            storage.LOCAL,
				AccessURLPrefix: "https://img.example.com/files",
				CustomPath:      "G:\\assets\\uploads",
			},
			dstFile:  "ignored",
			fileKey:  "202606/24/test image.png",
			expected: "https://img.example.com/files/202606/24/test%20image.png",
		},
		{
			name: "empty custom path falls back to default local route",
			cfg: &dao.CloudConfig{
				Type:            storage.LOCAL,
				AccessURLPrefix: "https://img.example.com",
			},
			dstFile:  "ignored",
			fileKey:  "202606/24/test image.png",
			expected: "https://img.example.com/storage/uploads/202606/24/test%20image.png",
		},
		{
			name: "non local storage keeps returned object key",
			cfg: &dao.CloudConfig{
				Type:            storage.MinIO,
				AccessURLPrefix: "https://cdn.example.com",
			},
			dstFile:  "bucket/path/test image.png",
			fileKey:  "202606/24/test image.png",
			expected: "https://cdn.example.com/bucket/path/test%20image.png",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildUserAccessURL(tc.cfg, tc.dstFile, tc.fileKey); got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}