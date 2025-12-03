package cli

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigStructure(t *testing.T) {
	cfg := &StormConfig{
		Version: "1.0",
		Project: "test",
	}
	cfg.Database.URL = "postgres://localhost:5432/test"
	cfg.Database.MaxConnections = 10

	if cfg.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", cfg.Version)
	}
	if cfg.Project != "test" {
		t.Errorf("expected project test, got %s", cfg.Project)
	}
	if cfg.Database.URL != "postgres://localhost:5432/test" {
		t.Errorf("expected database URL postgres://localhost:5432/test, got %s", cfg.Database.URL)
	}
}

func TestDatabaseConfig(t *testing.T) {
	tests := []struct {
		name   string
		config StormConfig
		valid  bool
	}{
		{
			name: "valid config",
			config: StormConfig{
				Version: "1.0",
				Project: "test",
			},
			valid: true,
		},
		{
			name: "config with max connections",
			config: StormConfig{
				Version: "1.0",
				Project: "test",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Project == "" && tt.valid {
				t.Error("expected valid config to have Project")
			}
		})
	}
}

func TestLoadStormConfig(t *testing.T) {

	tempDir, err := ioutil.TempDir("", "storm_config_test")
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

	t.Run("load valid config", func(t *testing.T) {
		configContent := `version: "1.0"
project: "test-project"
database:
  driver: "mysql"
  url: "mysql://localhost:3306/test"
  max_connections: 50
models:
  package: "./custom/models"
migrations:
  directory: "./custom/migrations"
  table: "custom_migrations"
  auto_apply: true
orm:
  generate_hooks: true
  generate_tests: true
  generate_mocks: true
schema:
  strict_mode: true
  naming_convention: "camel_case"
`
		configFile := filepath.Join(tempDir, "storm.yaml")
		err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		config, err := LoadStormConfig(configFile)
		if err != nil {
			t.Fatalf("LoadStormConfig failed: %v", err)
		}

		if config.Version != "1.0" {
			t.Errorf("expected version 1.0, got %s", config.Version)
		}
		if config.Project != "test-project" {
			t.Errorf("expected project test-project, got %s", config.Project)
		}
		if config.Database.Driver != "mysql" {
			t.Errorf("expected database driver mysql, got %s", config.Database.Driver)
		}
		if config.Database.MaxConnections != 50 {
			t.Errorf("expected max connections 50, got %d", config.Database.MaxConnections)
		}
		if config.Models.Package != "./custom/models" {
			t.Errorf("expected models package ./custom/models, got %s", config.Models.Package)
		}
		if config.Migrations.Directory != "./custom/migrations" {
			t.Errorf("expected migrations directory ./custom/migrations, got %s", config.Migrations.Directory)
		}
		if config.Migrations.Table != "custom_migrations" {
			t.Errorf("expected migrations table custom_migrations, got %s", config.Migrations.Table)
		}
		if !config.Migrations.AutoApply {
			t.Error("expected migrations auto_apply to be true")
		}
		if !config.ORM.GenerateHooks {
			t.Error("expected ORM generate_hooks to be true")
		}
		if !config.Schema.StrictMode {
			t.Error("expected schema strict_mode to be true")
		}
		if config.Schema.NamingConvention != "camel_case" {
			t.Errorf("expected naming convention camel_case, got %s", config.Schema.NamingConvention)
		}
	})

	t.Run("load config with defaults", func(t *testing.T) {
		configContent := `version: "1.0"
project: "test-project"
`
		configFile := filepath.Join(tempDir, "storm_defaults.yaml")
		err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		config, err := LoadStormConfig(configFile)
		if err != nil {
			t.Fatalf("LoadStormConfig failed: %v", err)
		}

		if config.Database.Driver != "postgres" {
			t.Errorf("expected default database driver postgres, got %s", config.Database.Driver)
		}
		if config.Database.MaxConnections != 25 {
			t.Errorf("expected default max connections 25, got %d", config.Database.MaxConnections)
		}
		if config.Models.Package != "./models" {
			t.Errorf("expected default models package ./models, got %s", config.Models.Package)
		}
		if config.Migrations.Directory != "./migrations" {
			t.Errorf("expected default migrations directory ./migrations, got %s", config.Migrations.Directory)
		}
		if config.Migrations.Table != "schema_migrations" {
			t.Errorf("expected default migrations table schema_migrations, got %s", config.Migrations.Table)
		}
		if config.Schema.NamingConvention != "snake_case" {
			t.Errorf("expected default naming convention snake_case, got %s", config.Schema.NamingConvention)
		}
	})

	t.Run("load config with empty path", func(t *testing.T) {

		configContent := `version: "1.0"
project: "auto-detect"
`
		err := ioutil.WriteFile("storm.yaml", []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		config, err := LoadStormConfig("")
		if err != nil {
			t.Fatalf("LoadStormConfig failed: %v", err)
		}

		if config.Project != "auto-detect" {
			t.Errorf("expected project auto-detect, got %s", config.Project)
		}
	})

	t.Run("load config with invalid yaml", func(t *testing.T) {
		configContent := `version: "1.0"
project: "test"
invalid: yaml: content:
  - bad
    - format
`
		configFile := filepath.Join(tempDir, "invalid.yaml")
		err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		_, err = LoadStormConfig(configFile)
		if err == nil {
			t.Error("expected error for invalid yaml")
		}
		if !strings.Contains(err.Error(), "failed to parse config file") {
			t.Errorf("expected parse error, got %v", err)
		}
	})

	t.Run("load config with non-existent file", func(t *testing.T) {
		_, err := LoadStormConfig("/non/existent/file.yaml")
		if err == nil {
			t.Error("expected error for non-existent file")
		}
		if !strings.Contains(err.Error(), "failed to read config file") {
			t.Errorf("expected read error, got %v", err)
		}
	})

	t.Run("load config returns nil when no file found", func(t *testing.T) {

		os.Remove("storm.yaml")
		os.Remove("storm.yml")
		os.Remove(".storm.yaml")
		os.Remove(".storm.yml")

		config, err := LoadStormConfig("")
		if err != nil {
			t.Fatalf("LoadStormConfig failed: %v", err)
		}
		if config != nil {
			t.Error("expected nil config when no file found")
		}
	})
}

func TestGetConfigPath(t *testing.T) {

	tempDir, err := ioutil.TempDir("", "storm_config_path_test")
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

	t.Run("get config path with env var", func(t *testing.T) {
		envPath := "/custom/storm.yaml"
		os.Setenv("STORM_CONFIG", envPath)
		defer os.Unsetenv("STORM_CONFIG")

		path := GetConfigPath()
		if path != envPath {
			t.Errorf("expected path %s, got %s", envPath, path)
		}
	})

	t.Run("get config path from current directory", func(t *testing.T) {
		os.Unsetenv("STORM_CONFIG")

		configContent := `version: "1.0"`
		err := ioutil.WriteFile("storm.yaml", []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		path := GetConfigPath()
		if path != "storm.yaml" {
			t.Errorf("expected path storm.yaml, got %s", path)
		}
	})

	t.Run("get config path returns empty when no file found", func(t *testing.T) {
		os.Unsetenv("STORM_CONFIG")

		os.Remove("storm.yaml")
		os.Remove("storm.yml")
		os.Remove(".storm.yaml")
		os.Remove(".storm.yml")

		path := GetConfigPath()
		if path != "" {
			t.Errorf("expected empty path, got %s", path)
		}
	})
}

func TestSaveStormConfig(t *testing.T) {

	tempDir, err := ioutil.TempDir("", "storm_save_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("save config to file", func(t *testing.T) {
		config := &StormConfig{
			Version: "1.0",
			Project: "test-project",
		}
		config.Database.URL = "postgres://localhost:5432/test"
		config.Database.MaxConnections = 20

		configFile := filepath.Join(tempDir, "output.yaml")
		err := SaveStormConfig(config, configFile)
		if err != nil {
			t.Fatalf("SaveStormConfig failed: %v", err)
		}

		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			t.Error("config file was not created")
		}

		loadedConfig, err := LoadStormConfig(configFile)
		if err != nil {
			t.Fatalf("failed to load saved config: %v", err)
		}

		if loadedConfig.Version != config.Version {
			t.Errorf("expected version %s, got %s", config.Version, loadedConfig.Version)
		}
		if loadedConfig.Project != config.Project {
			t.Errorf("expected project %s, got %s", config.Project, loadedConfig.Project)
		}
		if loadedConfig.Database.URL != config.Database.URL {
			t.Errorf("expected database URL %s, got %s", config.Database.URL, loadedConfig.Database.URL)
		}
	})

	t.Run("save config with default filename", func(t *testing.T) {

		oldCwd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		defer os.Chdir(oldCwd)
		os.Chdir(tempDir)

		config := &StormConfig{
			Version: "1.0",
			Project: "default-test",
		}

		err = SaveStormConfig(config, "")
		if err != nil {
			t.Fatalf("SaveStormConfig failed: %v", err)
		}

		if _, err := os.Stat("storm.yaml"); os.IsNotExist(err) {
			t.Error("default config file was not created")
		}
	})

	t.Run("save config creates directory", func(t *testing.T) {
		config := &StormConfig{
			Version: "1.0",
			Project: "nested-test",
		}

		nestedPath := filepath.Join(tempDir, "nested", "dir", "config.yaml")
		err := SaveStormConfig(config, nestedPath)
		if err != nil {
			t.Fatalf("SaveStormConfig failed: %v", err)
		}

		if _, err := os.Stat(nestedPath); os.IsNotExist(err) {
			t.Error("nested config file was not created")
		}
	})

	t.Run("save config handles permission errors", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("skipping permission test when running as root")
		}

		config := &StormConfig{
			Version: "1.0",
			Project: "permission-test",
		}

		readOnlyDir := filepath.Join(tempDir, "readonly")
		if err := os.MkdirAll(readOnlyDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(readOnlyDir, 0444); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(readOnlyDir, 0755)

		restrictedPath := filepath.Join(readOnlyDir, "restricted", "config.yaml")
		err := SaveStormConfig(config, restrictedPath)
		if err == nil {
			t.Error("expected error for permission denied")
		}
		if !strings.Contains(err.Error(), "failed to create directory") {
			t.Errorf("expected directory creation error, got: %v", err)
		}
	})
}
