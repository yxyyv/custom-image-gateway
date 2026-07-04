package service

import (
	"errors"
	"testing"

	"github.com/haierkeys/custom-image-gateway/global"
	"github.com/haierkeys/custom-image-gateway/internal/dao"
	"github.com/haierkeys/custom-image-gateway/pkg/code"
	"github.com/haierkeys/custom-image-gateway/pkg/storage"
	"gorm.io/gorm"
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

func TestResolveUserUploadConfigByID(t *testing.T) {
	svc := &Service{
		dao: &dao.Dao{},
	}
	params := &ClientUploadParams{ID: 12}
	got, err := resolveUserUploadConfig(
		svc,
		99,
		params,
		func(id int64, uid int64) (*dao.CloudConfig, error) {
			if id != 12 || uid != 99 {
				t.Fatalf("unexpected id/uid: %d/%d", id, uid)
			}
			return &dao.CloudConfig{ID: 12, UID: 99, Type: storage.LOCAL}, nil
		},
		func(uid int64) (*dao.CloudConfig, error) {
			t.Fatal("fallback lookup should not be called when id is provided")
			return nil, nil
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got == nil || got.ID != 12 {
		t.Fatalf("expected config id 12, got %#v", got)
	}
}

func TestResolveUserUploadConfigByIDNotFound(t *testing.T) {
	svc := &Service{
		dao: &dao.Dao{},
	}
	_, err := resolveUserUploadConfig(
		svc,
		99,
		&ClientUploadParams{ID: 12},
		func(id int64, uid int64) (*dao.CloudConfig, error) {
			return nil, gorm.ErrRecordNotFound
		},
		func(uid int64) (*dao.CloudConfig, error) {
			t.Fatal("fallback lookup should not be called when id is provided")
			return nil, nil
		},
	)
	if !errors.Is(err, code.ErrorUserCloudConfigIDNotFound) {
		t.Fatalf("expected ErrorUserCloudConfigIDNotFound, got %v", err)
	}
}

func TestResolveUserUploadConfigFallbackToEnabled(t *testing.T) {
	svc := &Service{
		dao: &dao.Dao{},
	}
	got, err := resolveUserUploadConfig(
		svc,
		88,
		&ClientUploadParams{},
		func(id int64, uid int64) (*dao.CloudConfig, error) {
			t.Fatal("id lookup should not be called when id is empty")
			return nil, nil
		},
		func(uid int64) (*dao.CloudConfig, error) {
			if uid != 88 {
				t.Fatalf("unexpected uid: %d", uid)
			}
			return &dao.CloudConfig{ID: 3, UID: 88, Type: storage.MinIO}, nil
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got == nil || got.ID != 3 {
		t.Fatalf("expected config id 3, got %#v", got)
	}
}

func TestResolveUserUploadConfigFallbackNotFound(t *testing.T) {
	svc := &Service{
		dao: &dao.Dao{},
	}
	_, err := resolveUserUploadConfig(
		svc,
		88,
		&ClientUploadParams{},
		func(id int64, uid int64) (*dao.CloudConfig, error) {
			t.Fatal("id lookup should not be called when id is empty")
			return nil, nil
		},
		func(uid int64) (*dao.CloudConfig, error) {
			return nil, gorm.ErrRecordNotFound
		},
	)
	if !errors.Is(err, code.ErrorUserCloudConfigNotFound) {
		t.Fatalf("expected ErrorUserCloudConfigNotFound, got %v", err)
	}
}
