package dao

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/haierkeys/custom-image-gateway/global"
)

func TestCreateCloudConfigPreservesDisabledState(t *testing.T) {
	testDir, err := os.MkdirTemp(".", "dao-cloud-config-test-")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(testDir)
	})

	db, err := NewDBEngineTest(global.Database{
		Type:         "sqlite",
		Path:         filepath.Join(testDir, "cloud-config.db"),
		MaxIdleConns: 1,
		MaxOpenConns: 1,
	})
	if err != nil {
		t.Fatalf("create test db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	d := New(db, context.Background())
	id, err := d.Create(&CloudConfigSet{
		Type:            "localfs",
		AccessURLPrefix: "https://img.example.com",
		IsEnabled:       0,
	}, 42)
	if err != nil {
		t.Fatalf("create cloud config: %v", err)
	}

	got, err := d.GetById(id, 42)
	if err != nil {
		t.Fatalf("load cloud config: %v", err)
	}
	if got.IsEnabled != 0 {
		t.Fatalf("expected created config to remain disabled, got isEnabled=%d", got.IsEnabled)
	}
}
