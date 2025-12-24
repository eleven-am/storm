package storm

import (
	"fmt"
	"testing"
	"time"
)

func TestNewConfig(t *testing.T) {
	config := NewConfig()

	if config.Driver != "postgres" {
		t.Errorf("Expected driver to be 'postgres', got %s", config.Driver)
	}
	if config.MaxOpenConns != 25 {
		t.Errorf("Expected MaxOpenConns to be 25, got %d", config.MaxOpenConns)
	}
	if config.MaxIdleConns != 5 {
		t.Errorf("Expected MaxIdleConns to be 5, got %d", config.MaxIdleConns)
	}
	if config.ConnMaxLifetime != time.Hour {
		t.Errorf("Expected ConnMaxLifetime to be 1 hour, got %v", config.ConnMaxLifetime)
	}
	if config.ModelsPackage != "./models" {
		t.Errorf("Expected ModelsPackage to be './models', got %s", config.ModelsPackage)
	}
	if config.MigrationsDir != "./migrations" {
		t.Errorf("Expected MigrationsDir to be './migrations', got %s", config.MigrationsDir)
	}
	if config.MigrationsTable != "schema_migrations" {
		t.Errorf("Expected MigrationsTable to be 'schema_migrations', got %s", config.MigrationsTable)
	}
	if config.AutoMigrate != false {
		t.Errorf("Expected AutoMigrate to be false, got %v", config.AutoMigrate)
	}
	if config.GenerateHooks != true {
		t.Errorf("Expected GenerateHooks to be true, got %v", config.GenerateHooks)
	}
	if config.GenerateTests != false {
		t.Errorf("Expected GenerateTests to be false, got %v", config.GenerateTests)
	}
	if config.GenerateMocks != false {
		t.Errorf("Expected GenerateMocks to be false, got %v", config.GenerateMocks)
	}
	if config.StrictMode != true {
		t.Errorf("Expected StrictMode to be true, got %v", config.StrictMode)
	}
	if config.NamingConvention != "snake_case" {
		t.Errorf("Expected NamingConvention to be 'snake_case', got %s", config.NamingConvention)
	}
	if config.Debug != false {
		t.Errorf("Expected Debug to be false, got %v", config.Debug)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name:        "valid config",
			config:      NewConfig(),
			expectError: false,
		},
		{
			name: "missing database URL",
			config: &Config{
				Driver: "postgres",
			},
			expectError: true,
		},
		{
			name: "missing driver",
			config: &Config{
				DatabaseURL: "postgres://localhost/test",
			},
			expectError: true,
		},
		{
			name: "invalid max open conns",
			config: &Config{
				DatabaseURL:  "postgres://localhost/test",
				Driver:       "postgres",
				MaxOpenConns: 0,
			},
			expectError: true,
		},
		{
			name: "invalid max idle conns",
			config: &Config{
				DatabaseURL:  "postgres://localhost/test",
				Driver:       "postgres",
				MaxOpenConns: 10,
				MaxIdleConns: -1,
			},
			expectError: true,
		},
		{
			name: "max idle > max open",
			config: &Config{
				DatabaseURL:  "postgres://localhost/test",
				Driver:       "postgres",
				MaxOpenConns: 5,
				MaxIdleConns: 10,
			},
			expectError: true,
		},
		{
			name: "invalid naming convention",
			config: &Config{
				DatabaseURL:      "postgres://localhost/test",
				Driver:           "postgres",
				MaxOpenConns:     10,
				MaxIdleConns:     5,
				ModelsPackage:    "./models",
				MigrationsDir:    "./migrations",
				MigrationsTable:  "schema_migrations",
				NamingConvention: "invalid",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.DatabaseURL == "" && !tt.expectError {
				tt.config.DatabaseURL = "postgres://localhost/test"
			}
			if tt.config.Driver == "" && !tt.expectError {
				tt.config.Driver = "postgres"
			}
			if tt.config.MaxOpenConns == 0 && !tt.expectError {
				tt.config.MaxOpenConns = 10
			}
			if tt.config.MaxIdleConns == 0 && !tt.expectError {
				tt.config.MaxIdleConns = 5
			}
			if tt.config.ModelsPackage == "" && !tt.expectError {
				tt.config.ModelsPackage = "./models"
			}
			if tt.config.MigrationsDir == "" && !tt.expectError {
				tt.config.MigrationsDir = "./migrations"
			}
			if tt.config.MigrationsTable == "" && !tt.expectError {
				tt.config.MigrationsTable = "schema_migrations"
			}
			if tt.config.NamingConvention == "" && !tt.expectError {
				tt.config.NamingConvention = "snake_case"
			}

			err := tt.config.Validate()
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestOptions(t *testing.T) {
	config := NewConfig()

	err := WithDriver("mysql")(config)
	if err != nil {
		t.Errorf("WithDriver failed: %v", err)
	}
	if config.Driver != "mysql" {
		t.Errorf("Expected driver to be 'mysql', got %s", config.Driver)
	}

	err = WithMaxConnections(50)(config)
	if err != nil {
		t.Errorf("WithMaxConnections failed: %v", err)
	}
	if config.MaxOpenConns != 50 {
		t.Errorf("Expected MaxOpenConns to be 50, got %d", config.MaxOpenConns)
	}
	if config.MaxIdleConns != 12 {
		t.Errorf("Expected MaxIdleConns to be 12, got %d", config.MaxIdleConns)
	}

	err = WithModelsPackage("./internal/models")(config)
	if err != nil {
		t.Errorf("WithModelsPackage failed: %v", err)
	}
	if config.ModelsPackage != "./internal/models" {
		t.Errorf("Expected ModelsPackage to be './internal/models', got %s", config.ModelsPackage)
	}

	err = WithAutoMigrate(true)(config)
	if err != nil {
		t.Errorf("WithAutoMigrate failed: %v", err)
	}
	if config.AutoMigrate != true {
		t.Errorf("Expected AutoMigrate to be true, got %v", config.AutoMigrate)
	}

	err = WithDebug(true)(config)
	if err != nil {
		t.Errorf("WithDebug failed: %v", err)
	}
	if config.Debug != true {
		t.Errorf("Expected Debug to be true, got %v", config.Debug)
	}
}

func TestOptionValidation(t *testing.T) {
	config := NewConfig()

	err := WithDriver("")(config)
	if err == nil {
		t.Error("Expected error for empty driver")
	}

	err = WithMaxConnections(0)(config)
	if err == nil {
		t.Error("Expected error for zero max connections")
	}

	err = WithModelsPackage("")(config)
	if err == nil {
		t.Error("Expected error for empty models package")
	}

	err = WithNamingConvention("invalid")(config)
	if err == nil {
		t.Error("Expected error for invalid naming convention")
	}

	err = WithLogger(nil)(config)
	if err == nil {
		t.Error("Expected error for nil logger")
	}
}

func TestErrorTypes(t *testing.T) {
	err := NewError(ErrorTypeConnection, "test", fmt.Errorf("connection failed"))
	if err.Type != ErrorTypeConnection {
		t.Errorf("Expected error type to be %s, got %s", ErrorTypeConnection, err.Type)
	}
	if err.Op != "test" {
		t.Errorf("Expected operation to be 'test', got %s", err.Op)
	}

	err = err.WithDetails("host", "localhost").WithDetails("port", 5432)
	if err.Details["host"] != "localhost" {
		t.Errorf("Expected host detail to be 'localhost', got %v", err.Details["host"])
	}
	if err.Details["port"] != 5432 {
		t.Errorf("Expected port detail to be 5432, got %v", err.Details["port"])
	}

	expected := "storm connection: test: connection failed"
	if err.Error() != expected {
		t.Errorf("Expected error string to be '%s', got '%s'", expected, err.Error())
	}

	err2 := NewError(ErrorTypeConnection, "other", fmt.Errorf("other error"))
	if !err.Is(err2) {
		t.Error("Expected errors to be the same type")
	}

	err3 := NewError(ErrorTypeConfig, "test", fmt.Errorf("config error"))
	if err.Is(err3) {
		t.Error("Expected errors to be different types")
	}
}

func TestVersionInfo(t *testing.T) {
	info := VersionInfo()
	expected := "Storm 1.0.0-alpha (API v1)"
	if info != expected {
		t.Errorf("Expected version info to be '%s', got '%s'", expected, info)
	}

	fullInfo := FullVersionInfo()
	if !contains(fullInfo, "Storm 1.0.0-alpha") {
		t.Error("Expected full version info to contain version")
	}
	if !contains(fullInfo, "API Version: v1") {
		t.Error("Expected full version info to contain API version")
	}
	if !contains(fullInfo, "Go Version:") {
		t.Error("Expected full version info to contain Go version")
	}

	SetBuildInfo("abc123", "2023-01-01", "go1.21.0")
	fullInfo = FullVersionInfo()
	if !contains(fullInfo, "Git Commit: abc123") {
		t.Error("Expected full version info to contain git commit")
	}
	if !contains(fullInfo, "Build Date: 2023-01-01") {
		t.Error("Expected full version info to contain build date")
	}
}

func TestStormCreation(t *testing.T) {
	_, err := New("invalid://url")
	if err != nil {
		t.Logf("Got expected error for invalid URL: %v", err)
	}

	_, err = NewWithConfig(nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}

	invalidConfig := &Config{
		DatabaseURL: "postgres://user:pass@localhost/test",
		Driver:      "",
	}
	_, err = NewWithConfig(invalidConfig)
	if err == nil {
		t.Error("Expected error for invalid config")
	}
}

func TestWithAutoMigrateOptions(t *testing.T) {
	config := NewConfig()

	opts := AutoMigrateOptions{
		AllowDestructive:    true,
		DryRun:              true,
		CreateDBIfNotExists: true,
		LockTimeout:         30 * time.Second,
	}

	err := WithAutoMigrateOptions(opts)(config)
	if err != nil {
		t.Errorf("WithAutoMigrateOptions failed: %v", err)
	}

	if config.AutoMigrateOpts.AllowDestructive != true {
		t.Errorf("Expected AllowDestructive to be true, got %v", config.AutoMigrateOpts.AllowDestructive)
	}
	if config.AutoMigrateOpts.DryRun != true {
		t.Errorf("Expected DryRun to be true, got %v", config.AutoMigrateOpts.DryRun)
	}
	if config.AutoMigrateOpts.CreateDBIfNotExists != true {
		t.Errorf("Expected CreateDBIfNotExists to be true, got %v", config.AutoMigrateOpts.CreateDBIfNotExists)
	}
	if config.AutoMigrateOpts.LockTimeout != 30*time.Second {
		t.Errorf("Expected LockTimeout to be 30s, got %v", config.AutoMigrateOpts.LockTimeout)
	}
}

func TestAutoMigrateOptionsDefaults(t *testing.T) {
	opts := AutoMigrateOptions{}

	if opts.AllowDestructive != false {
		t.Error("Expected AllowDestructive default to be false")
	}
	if opts.DryRun != false {
		t.Error("Expected DryRun default to be false")
	}
	if opts.CreateDBIfNotExists != false {
		t.Error("Expected CreateDBIfNotExists default to be false")
	}
	if opts.LockTimeout != 0 {
		t.Error("Expected LockTimeout default to be 0")
	}
}

func TestStubMigratorAutoMigrate(t *testing.T) {
	m := &migrator{}
	err := m.AutoMigrate(nil, AutoMigrateOptions{})
	if err != ErrNotImplemented {
		t.Errorf("Expected ErrNotImplemented, got %v", err)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && s[:len(substr)] == substr ||
		(len(s) > len(substr) && findInString(s, substr))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
