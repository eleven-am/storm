package cli

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCreate(t *testing.T) {

	tempDir, err := ioutil.TempDir("", "storm_create_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	origOutputDir := outputDir
	defer func() { outputDir = origOutputDir }()

	t.Run("creates migration files", func(t *testing.T) {

		outputDir = tempDir

		cmd := createCmd
		args := []string{"add_users_table"}

		err := runCreate(cmd, args)
		if err != nil {
			t.Fatalf("runCreate failed: %v", err)
		}

		files, err := ioutil.ReadDir(tempDir)
		if err != nil {
			t.Fatal(err)
		}

		var upFile, downFile string
		for _, file := range files {
			if strings.Contains(file.Name(), "add_users_table.up.sql") {
				upFile = file.Name()
			}
			if strings.Contains(file.Name(), "add_users_table.down.sql") {
				downFile = file.Name()
			}
		}

		if upFile == "" {
			t.Error("UP migration file was not created")
		}
		if downFile == "" {
			t.Error("DOWN migration file was not created")
		}

		if upFile != "" {
			upContent, err := ioutil.ReadFile(filepath.Join(tempDir, upFile))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(upContent), "Migration: add_users_table") {
				t.Error("UP file does not contain expected migration name")
			}
		}

		if downFile != "" {
			downContent, err := ioutil.ReadFile(filepath.Join(tempDir, downFile))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(downContent), "Migration: add_users_table") {
				t.Error("DOWN file does not contain expected migration name")
			}
		}
	})

	t.Run("creates output directory if it doesn't exist", func(t *testing.T) {

		nestedDir := filepath.Join(tempDir, "nested", "migrations")
		outputDir = nestedDir

		cmd := createCmd
		args := []string{"create_posts_table"}

		err := runCreate(cmd, args)
		if err != nil {
			t.Fatalf("runCreate failed: %v", err)
		}

		if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
			t.Error("output directory was not created")
		}

		files, err := ioutil.ReadDir(nestedDir)
		if err != nil {
			t.Fatal(err)
		}

		if len(files) != 2 {
			t.Errorf("expected 2 files, got %d", len(files))
		}
	})

	t.Run("handles permission errors gracefully", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("skipping permission test when running as root")
		}

		readOnlyDir := filepath.Join(tempDir, "readonly")
		if err := os.MkdirAll(readOnlyDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(readOnlyDir, 0444); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(readOnlyDir, 0755)

		restrictedDir := filepath.Join(readOnlyDir, "restricted")
		outputDir = restrictedDir

		cmd := createCmd
		args := []string{"permission_test"}

		err := runCreate(cmd, args)
		if err == nil {
			t.Error("expected error for permission denied")
		}
		if !strings.Contains(err.Error(), "failed to create output directory") {
			t.Errorf("expected directory creation error, got: %v", err)
		}
	})
}

func TestCreateCommand(t *testing.T) {
	t.Run("command structure", func(t *testing.T) {
		if createCmd.Use != "create [name]" {
			t.Errorf("expected Use to be 'create [name]', got %s", createCmd.Use)
		}

		if createCmd.Short != "Create empty migration files" {
			t.Errorf("expected Short to be 'Create empty migration files', got %s", createCmd.Short)
		}

		if createCmd.RunE == nil {
			t.Error("expected RunE to be set")
		}
	})

	t.Run("command flags", func(t *testing.T) {
		outputFlag := createCmd.Flags().Lookup("output")
		if outputFlag == nil {
			t.Error("expected output flag to be defined")
		}

		if outputFlag.DefValue != "./migrations" {
			t.Errorf("expected output flag default to be './migrations', got %s", outputFlag.DefValue)
		}
	})
}
