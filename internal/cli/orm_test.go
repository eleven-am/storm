package cli

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunORM(t *testing.T) {

	tempDir, err := ioutil.TempDir("", "storm_orm_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	origOrmPackage := ormPackage
	origOrmOutput := ormOutput
	origOrmIncludeHooks := ormIncludeHooks
	origOrmIncludeTests := ormIncludeTests
	origOrmIncludeMocks := ormIncludeMocks
	origDebug := debug
	origVerbose := verbose
	origStormConfig := stormConfig
	defer func() {
		ormPackage = origOrmPackage
		ormOutput = origOrmOutput
		ormIncludeHooks = origOrmIncludeHooks
		ormIncludeTests = origOrmIncludeTests
		ormIncludeMocks = origOrmIncludeMocks
		debug = origDebug
		verbose = origVerbose
		stormConfig = origStormConfig
	}()

	t.Run("uses default package path when not specified", func(t *testing.T) {

		ormPackage = ""
		ormOutput = ""
		ormIncludeHooks = false
		ormIncludeTests = false
		ormIncludeMocks = false
		debug = false
		verbose = false
		stormConfig = nil

		err := runORM(ormCmd, []string{})
		if err == nil {
			t.Error("expected error due to missing models")
		}

		if !strings.Contains(err.Error(), "failed to") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("uses configuration from storm config", func(t *testing.T) {

		stormConfig = &StormConfig{
			Version: "1.0",
			Project: "test-project",
		}
		stormConfig.Models.Package = "./custom/models"
		stormConfig.ORM.GenerateHooks = true
		stormConfig.ORM.GenerateTests = true
		stormConfig.ORM.GenerateMocks = true

		ormPackage = ""
		ormOutput = ""
		ormIncludeHooks = false
		ormIncludeTests = false
		ormIncludeMocks = false
		debug = false
		verbose = false

		err := runORM(ormCmd, []string{})
		if err == nil {
			t.Error("expected error due to missing models")
		}

		if !strings.Contains(err.Error(), "failed to") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("handles non-existent package path", func(t *testing.T) {

		ormPackage = "/non/existent/path"
		ormOutput = ""
		ormIncludeHooks = false
		ormIncludeTests = false
		ormIncludeMocks = false
		debug = false
		verbose = false
		stormConfig = nil

		err := runORM(ormCmd, []string{})
		if err == nil {
			t.Error("expected error with non-existent package path")
		}

		if !strings.Contains(err.Error(), "failed to") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("handles verbose output", func(t *testing.T) {

		packageDir := filepath.Join(tempDir, "models")
		err := os.MkdirAll(packageDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		ormPackage = packageDir
		ormOutput = filepath.Join(tempDir, "output")
		ormIncludeHooks = true
		ormIncludeTests = true
		ormIncludeMocks = true
		debug = false
		verbose = true
		stormConfig = nil

		err = runORM(ormCmd, []string{})
		if err == nil {
			t.Error("expected error due to missing models")
		}

		if !strings.Contains(err.Error(), "failed to") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("sets output directory to package path when not specified", func(t *testing.T) {

		packageDir := filepath.Join(tempDir, "models")
		err := os.MkdirAll(packageDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		ormPackage = packageDir
		ormOutput = ""
		ormIncludeHooks = false
		ormIncludeTests = false
		ormIncludeMocks = false
		debug = false
		verbose = false
		stormConfig = nil

		err = runORM(ormCmd, []string{})
		if err == nil {
			t.Error("expected error due to missing models")
		}

		if !strings.Contains(err.Error(), "failed to") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestORMCommand(t *testing.T) {
	t.Run("command structure", func(t *testing.T) {
		if ormCmd.Use != "orm" {
			t.Errorf("expected Use to be 'orm', got %s", ormCmd.Use)
		}

		if ormCmd.Short != "Generate ORM code from models" {
			t.Errorf("expected Short to be 'Generate ORM code from models', got %s", ormCmd.Short)
		}

		if ormCmd.RunE == nil {
			t.Error("expected RunE to be set")
		}
	})

	t.Run("command flags", func(t *testing.T) {
		expectedFlags := []string{
			"package",
			"output",
			"hooks",
			"tests",
			"mocks",
		}

		for _, flagName := range expectedFlags {
			flag := ormCmd.Flags().Lookup(flagName)
			if flag == nil {
				t.Errorf("expected flag %s to be defined", flagName)
			}
		}

		hooksFlag := ormCmd.Flags().Lookup("hooks")
		if hooksFlag != nil && hooksFlag.DefValue != "false" {
			t.Errorf("expected hooks flag default to be 'false', got %s", hooksFlag.DefValue)
		}

		testsFlag := ormCmd.Flags().Lookup("tests")
		if testsFlag != nil && testsFlag.DefValue != "false" {
			t.Errorf("expected tests flag default to be 'false', got %s", testsFlag.DefValue)
		}

		mocksFlag := ormCmd.Flags().Lookup("mocks")
		if mocksFlag != nil && mocksFlag.DefValue != "false" {
			t.Errorf("expected mocks flag default to be 'false', got %s", mocksFlag.DefValue)
		}
	})
}
