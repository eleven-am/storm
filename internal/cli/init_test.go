package cli

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInit(t *testing.T) {

	tempDir, err := ioutil.TempDir("", "storm_init_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldCwd)
	os.Chdir(tempDir)

	origProject := initProject
	origDriver := initDriver
	origForce := initForce
	defer func() {
		initProject = origProject
		initDriver = origDriver
		initForce = origForce
	}()

	t.Run("creates storm.yaml with default values", func(t *testing.T) {

		initProject = ""
		initDriver = "postgres"
		initForce = false

		err := runInit(initCmd, []string{})
		if err != nil {
			t.Fatalf("runInit failed: %v", err)
		}

		if _, err := os.Stat("storm.yaml"); os.IsNotExist(err) {
			t.Error("storm.yaml was not created")
		}

		config, err := LoadStormConfig("storm.yaml")
		if err != nil {
			t.Fatalf("failed to load created config: %v", err)
		}

		if config.Version != "1" {
			t.Errorf("expected version 1, got %s", config.Version)
		}
		if config.Database.Driver != "postgres" {
			t.Errorf("expected database driver postgres, got %s", config.Database.Driver)
		}
		if config.Database.MaxConnections != 25 {
			t.Errorf("expected max connections 25, got %d", config.Database.MaxConnections)
		}
		if config.Models.Package != "./models" {
			t.Errorf("expected models package ./models, got %s", config.Models.Package)
		}
		if config.Migrations.Directory != "./migrations" {
			t.Errorf("expected migrations directory ./migrations, got %s", config.Migrations.Directory)
		}
		if config.Migrations.Table != "schema_migrations" {
			t.Errorf("expected migrations table schema_migrations, got %s", config.Migrations.Table)
		}
		if config.Migrations.AutoApply != false {
			t.Error("expected migrations auto_apply to be false")
		}
		if config.ORM.GenerateHooks != true {
			t.Error("expected ORM generate_hooks to be true")
		}
		if config.ORM.GenerateTests != false {
			t.Error("expected ORM generate_tests to be false")
		}
		if config.ORM.GenerateMocks != false {
			t.Error("expected ORM generate_mocks to be false")
		}
		if config.Schema.StrictMode != true {
			t.Error("expected schema strict_mode to be true")
		}
		if config.Schema.NamingConvention != "snake_case" {
			t.Errorf("expected naming convention snake_case, got %s", config.Schema.NamingConvention)
		}

	})

	t.Run("creates storm.yaml with custom project name", func(t *testing.T) {

		os.Remove("storm.yaml")

		initProject = "my-awesome-project"
		initDriver = "postgres"
		initForce = false

		err := runInit(initCmd, []string{})
		if err != nil {
			t.Fatalf("runInit failed: %v", err)
		}

		config, err := LoadStormConfig("storm.yaml")
		if err != nil {
			t.Fatalf("failed to load created config: %v", err)
		}

		if config.Project != "my-awesome-project" {
			t.Errorf("expected project my-awesome-project, got %s", config.Project)
		}
	})

	t.Run("creates storm.yaml with custom driver", func(t *testing.T) {

		os.Remove("storm.yaml")

		initProject = "test-project"
		initDriver = "mysql"
		initForce = false

		err := runInit(initCmd, []string{})
		if err != nil {
			t.Fatalf("runInit failed: %v", err)
		}

		config, err := LoadStormConfig("storm.yaml")
		if err != nil {
			t.Fatalf("failed to load created config: %v", err)
		}

		if config.Database.Driver != "mysql" {
			t.Errorf("expected database driver mysql, got %s", config.Database.Driver)
		}
		if !strings.Contains(config.Database.URL, "mysql://") {
			t.Errorf("expected database URL to contain mysql://, got %s", config.Database.URL)
		}
	})

	t.Run("fails when storm.yaml already exists", func(t *testing.T) {

		os.Remove("storm.yaml")

		initProject = "test-project"
		initDriver = "postgres"
		initForce = false

		err := runInit(initCmd, []string{})
		if err != nil {
			t.Fatalf("runInit failed: %v", err)
		}

		err = runInit(initCmd, []string{})
		if err == nil {
			t.Error("expected error when storm.yaml already exists")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("expected 'already exists' error, got: %v", err)
		}
	})

	t.Run("overwrites when force flag is set", func(t *testing.T) {

		initProject = "forced-project"
		initDriver = "postgres"
		initForce = true

		err := runInit(initCmd, []string{})
		if err != nil {
			t.Fatalf("runInit failed: %v", err)
		}

		config, err := LoadStormConfig("storm.yaml")
		if err != nil {
			t.Fatalf("failed to load created config: %v", err)
		}

		if config.Project != "forced-project" {
			t.Errorf("expected project forced-project, got %s", config.Project)
		}
	})

	t.Run("uses directory name as project when not specified", func(t *testing.T) {

		projectDir := filepath.Join(tempDir, "my-project-dir")
		err := os.MkdirAll(projectDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		os.Chdir(projectDir)
		defer os.Chdir(tempDir)

		initProject = ""
		initDriver = "postgres"
		initForce = false

		err = runInit(initCmd, []string{})
		if err != nil {
			t.Fatalf("runInit failed: %v", err)
		}

		config, err := LoadStormConfig("storm.yaml")
		if err != nil {
			t.Fatalf("failed to load created config: %v", err)
		}

		if config.Project != "my-project-dir" {
			t.Errorf("expected project my-project-dir, got %s", config.Project)
		}
	})
}

func TestInitCommand(t *testing.T) {
	t.Run("command structure", func(t *testing.T) {
		if initCmd.Use != "init" {
			t.Errorf("expected Use to be 'init', got %s", initCmd.Use)
		}

		if initCmd.Short != "Initialize a new Storm configuration file" {
			t.Errorf("expected Short to be 'Initialize a new Storm configuration file', got %s", initCmd.Short)
		}

		if initCmd.RunE == nil {
			t.Error("expected RunE to be set")
		}
	})

	t.Run("command flags", func(t *testing.T) {
		projectFlag := initCmd.Flags().Lookup("project")
		if projectFlag == nil {
			t.Error("expected project flag to be defined")
		}

		driverFlag := initCmd.Flags().Lookup("driver")
		if driverFlag == nil {
			t.Error("expected driver flag to be defined")
		}
		if driverFlag.DefValue != "postgres" {
			t.Errorf("expected driver flag default to be 'postgres', got %s", driverFlag.DefValue)
		}

		forceFlag := initCmd.Flags().Lookup("force")
		if forceFlag == nil {
			t.Error("expected force flag to be defined")
		}
		if forceFlag.DefValue != "false" {
			t.Errorf("expected force flag default to be 'false', got %s", forceFlag.DefValue)
		}
	})
}
