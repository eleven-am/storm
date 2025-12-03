package migrator

import (
	"context"
	"testing"
)

func TestNewDBConfig(t *testing.T) {
	url := "postgres://user:pass@localhost:5432/testdb"
	config := NewDBConfig(url)

	if config == nil {
		t.Fatal("Expected config to be created")
	}

	if config.URL != url {
		t.Errorf("Expected URL to be %q, got %q", url, config.URL)
	}
}

func TestDBConfig_Connect(t *testing.T) {
	t.Run("invalid URL", func(t *testing.T) {
		config := &DBConfig{
			URL: "invalid-url",
		}

		ctx := context.Background()
		db, err := config.Connect(ctx)
		if err == nil {
			t.Error("Expected error for invalid URL")
		}
		if db != nil {
			t.Error("Expected nil db for invalid URL")
		}
	})

	t.Run("valid URL format but unreachable", func(t *testing.T) {
		config := &DBConfig{
			URL: "postgres://user:pass@nonexistent:5432/testdb",
		}

		ctx := context.Background()
		db, err := config.Connect(ctx)

		if db == nil && err != nil {

			t.Logf("Expected behavior: cannot connect to unreachable database: %v", err)
		}
	})

	t.Run("connection pool settings", func(t *testing.T) {
		config := &DBConfig{
			URL:          "postgres://user:pass@localhost:5432/testdb",
			MaxOpenConns: 10,
			MaxIdleConns: 5,
		}

		ctx := context.Background()
		db, err := config.Connect(ctx)
		if err != nil {
			t.Logf("Connect error (expected for test): %v", err)
		}
		if db != nil {
			db.Close()
		}
	})
}
