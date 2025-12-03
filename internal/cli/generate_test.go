package cli

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunGenerate(t *testing.T) {

	tempDir, err := ioutil.TempDir("", "storm_generate_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	origGeneratePackage := generatePackage
	origGenerateOutput := generateOutput
	origDebug := debug
	defer func() {
		generatePackage = origGeneratePackage
		generateOutput = origGenerateOutput
		debug = origDebug
	}()

	t.Run("fails with non-existent package path", func(t *testing.T) {

		generatePackage = "/non/existent/path"
		generateOutput = filepath.Join(tempDir, "schema.sql")
		debug = false

		err := runGenerate(generateCmd, []string{})
		if err == nil {
			t.Error("expected error with non-existent package path")
		}

		if !strings.Contains(err.Error(), "failed to") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("fails with invalid output path", func(t *testing.T) {

		packageDir := filepath.Join(tempDir, "models")
		err := os.MkdirAll(packageDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		modelFile := filepath.Join(packageDir, "user.go")
		modelContent := `package models

type User struct {
	ID   int    ` + "`" + `db:"id"` + "`" + `
	Name string ` + "`" + `db:"name"` + "`" + `
}`
		err = ioutil.WriteFile(modelFile, []byte(modelContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		if os.Geteuid() != 0 {
			readOnlyDir := filepath.Join(tempDir, "readonly")
			err = os.MkdirAll(readOnlyDir, 0755)
			if err != nil {
				t.Fatal(err)
			}
			err = os.Chmod(readOnlyDir, 0444)
			if err != nil {
				t.Fatal(err)
			}
			defer os.Chmod(readOnlyDir, 0755)

			generatePackage = packageDir
			generateOutput = filepath.Join(readOnlyDir, "schema.sql")
			debug = false

			err = runGenerate(generateCmd, []string{})
			if err == nil {
				t.Error("expected error with invalid output path")
			}

			if !strings.Contains(err.Error(), "failed to") {
				t.Errorf("unexpected error message: %v", err)
			}
		} else {
			t.Skip("skipping permission test when running as root")
		}
	})

	t.Run("handles directory creation for output file", func(t *testing.T) {

		packageDir := filepath.Join(tempDir, "models2")
		err := os.MkdirAll(packageDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		modelFile := filepath.Join(packageDir, "user.go")
		modelContent := `package models

type User struct {
	ID   int    ` + "`" + `db:"id"` + "`" + `
	Name string ` + "`" + `db:"name"` + "`" + `
}`
		err = ioutil.WriteFile(modelFile, []byte(modelContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		generatePackage = packageDir
		generateOutput = filepath.Join(tempDir, "output", "schema.sql")
		debug = false

		err = runGenerate(generateCmd, []string{})

		if err != nil {

			if strings.Contains(err.Error(), "failed to resolve output path") {
				t.Errorf("should not fail on output path resolution: %v", err)
			}
		}
	})

	t.Run("handles empty package directory", func(t *testing.T) {

		packageDir := filepath.Join(tempDir, "empty")
		err := os.MkdirAll(packageDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		generatePackage = packageDir
		generateOutput = filepath.Join(tempDir, "empty_schema.sql")
		debug = false

		err = runGenerate(generateCmd, []string{})
		if err == nil {
			t.Error("expected error with empty package directory")
		}

		if !strings.Contains(err.Error(), "failed to") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestGenerateCommand(t *testing.T) {
	t.Run("command structure", func(t *testing.T) {
		if generateCmd.Use != "generate" {
			t.Errorf("expected Use to be 'generate', got %s", generateCmd.Use)
		}

		if generateCmd.Short != "Generate initial schema from Go structs" {
			t.Errorf("expected Short to be 'Generate initial schema from Go structs', got %s", generateCmd.Short)
		}

		if generateCmd.RunE == nil {
			t.Error("expected RunE to be set")
		}
	})

	t.Run("command flags", func(t *testing.T) {
		packageFlag := generateCmd.Flags().Lookup("package")
		if packageFlag == nil {
			t.Error("expected package flag to be defined")
		}
		if packageFlag.DefValue != "./models" {
			t.Errorf("expected package flag default to be './models', got %s", packageFlag.DefValue)
		}

		outputFlag := generateCmd.Flags().Lookup("output")
		if outputFlag == nil {
			t.Error("expected output flag to be defined")
		}
		if outputFlag.DefValue != "schema.sql" {
			t.Errorf("expected output flag default to be 'schema.sql', got %s", outputFlag.DefValue)
		}
	})
}
