package storm

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all Storm configuration
type Config struct {
	// Database settings
	Driver          string        `yaml:"driver" env:"STORM_DRIVER"`
	DatabaseURL     string        `yaml:"database_url" env:"STORM_DATABASE_URL"`
	MaxOpenConns    int           `yaml:"max_open_conns" env:"STORM_MAX_OPEN_CONNS"`
	MaxIdleConns    int           `yaml:"max_idle_conns" env:"STORM_MAX_IDLE_CONNS"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" env:"STORM_CONN_MAX_LIFETIME"`

	// Models settings
	ModelsPackage string `yaml:"models_package" env:"STORM_MODELS_PACKAGE"`

	// Migration settings
	MigrationsDir   string `yaml:"migrations_dir" env:"STORM_MIGRATIONS_DIR"`
	MigrationsTable string `yaml:"migrations_table" env:"STORM_MIGRATIONS_TABLE"`
	AutoMigrate     bool   `yaml:"auto_migrate" env:"STORM_AUTO_MIGRATE"`
	AutoMigrateOpts AutoMigrateOptions `yaml:"-"`

	// ORM settings
	GenerateHooks bool `yaml:"generate_hooks" env:"STORM_GENERATE_HOOKS"`
	GenerateTests bool `yaml:"generate_tests" env:"STORM_GENERATE_TESTS"`
	GenerateMocks bool `yaml:"generate_mocks" env:"STORM_GENERATE_MOCKS"`

	// Schema settings
	StrictMode       bool   `yaml:"strict_mode" env:"STORM_STRICT_MODE"`
	NamingConvention string `yaml:"naming_convention" env:"STORM_NAMING_CONVENTION"`

	// Runtime settings
	Logger Logger `yaml:"-"`
	Debug  bool   `yaml:"debug" env:"STORM_DEBUG"`
}

// NewConfig creates a config with sensible defaults
func NewConfig() *Config {
	return &Config{
		Driver:           "postgres",
		MaxOpenConns:     25,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Hour,
		ModelsPackage:    "./models",
		MigrationsDir:    "./migrations",
		MigrationsTable:  "schema_migrations",
		AutoMigrate:      false,
		GenerateHooks:    true,
		GenerateTests:    false,
		GenerateMocks:    false,
		StrictMode:       true,
		NamingConvention: "snake_case",
		Logger:           NewDefaultLogger(),
		Debug:            false,
	}
}

// LoadConfig loads configuration from file and environment
func LoadConfig(path string) (*Config, error) {
	config := NewConfig()

	if path != "" {
		if err := config.LoadFile(path); err != nil {
			return nil, NewConfigError("load_file", err)
		}
	} else {
		locations := []string{"storm.yaml", "storm.yml", ".storm.yaml", ".storm.yml"}
		for _, loc := range locations {
			if _, err := os.Stat(loc); err == nil {
				if err := config.LoadFile(loc); err != nil {
					return nil, NewConfigError("load_file", err)
				}
				break
			}
		}
	}

	config.LoadEnv()

	if err := config.Validate(); err != nil {
		return nil, NewConfigError("validate", err)
	}

	return config, nil
}

// LoadFile loads configuration from a YAML file
func (c *Config) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	return yaml.Unmarshal(data, c)
}

// LoadEnv loads configuration from environment variables
func (c *Config) LoadEnv() {
	if url := os.Getenv("STORM_DATABASE_URL"); url != "" {
		c.DatabaseURL = url
	}
	if driver := os.Getenv("STORM_DRIVER"); driver != "" {
		c.Driver = driver
	}
	if maxOpen := os.Getenv("STORM_MAX_OPEN_CONNS"); maxOpen != "" {
		if val, err := strconv.Atoi(maxOpen); err == nil {
			c.MaxOpenConns = val
		}
	}
	if maxIdle := os.Getenv("STORM_MAX_IDLE_CONNS"); maxIdle != "" {
		if val, err := strconv.Atoi(maxIdle); err == nil {
			c.MaxIdleConns = val
		}
	}
	if lifetime := os.Getenv("STORM_CONN_MAX_LIFETIME"); lifetime != "" {
		if val, err := time.ParseDuration(lifetime); err == nil {
			c.ConnMaxLifetime = val
		}
	}
	if pkg := os.Getenv("STORM_MODELS_PACKAGE"); pkg != "" {
		c.ModelsPackage = pkg
	}
	if dir := os.Getenv("STORM_MIGRATIONS_DIR"); dir != "" {
		c.MigrationsDir = dir
	}
	if table := os.Getenv("STORM_MIGRATIONS_TABLE"); table != "" {
		c.MigrationsTable = table
	}
	if auto := os.Getenv("STORM_AUTO_MIGRATE"); auto != "" {
		c.AutoMigrate = auto == "true"
	}
	if hooks := os.Getenv("STORM_GENERATE_HOOKS"); hooks != "" {
		c.GenerateHooks = hooks == "true"
	}
	if tests := os.Getenv("STORM_GENERATE_TESTS"); tests != "" {
		c.GenerateTests = tests == "true"
	}
	if mocks := os.Getenv("STORM_GENERATE_MOCKS"); mocks != "" {
		c.GenerateMocks = mocks == "true"
	}
	if strict := os.Getenv("STORM_STRICT_MODE"); strict != "" {
		c.StrictMode = strict == "true"
	}
	if naming := os.Getenv("STORM_NAMING_CONVENTION"); naming != "" {
		c.NamingConvention = naming
	}
	if debug := os.Getenv("STORM_DEBUG"); debug != "" {
		c.Debug = debug == "true"
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("database URL is required")
	}

	if c.Driver == "" {
		return fmt.Errorf("database driver is required")
	}

	if c.MaxOpenConns < 1 {
		return fmt.Errorf("max open connections must be at least 1")
	}

	if c.MaxIdleConns < 0 {
		return fmt.Errorf("max idle connections cannot be negative")
	}

	if c.MaxIdleConns > c.MaxOpenConns {
		return fmt.Errorf("max idle connections cannot exceed max open connections")
	}

	if c.ModelsPackage == "" {
		return fmt.Errorf("models package is required")
	}

	if c.MigrationsDir == "" {
		return fmt.Errorf("migrations directory is required")
	}

	if c.MigrationsTable == "" {
		return fmt.Errorf("migrations table is required")
	}

	if c.NamingConvention != "snake_case" && c.NamingConvention != "camelCase" {
		return fmt.Errorf("naming convention must be 'snake_case' or 'camelCase'")
	}

	return nil
}

// Clone returns a deep copy of the configuration
func (c *Config) Clone() *Config {
	clone := *c
	return &clone
}

// SaveFile saves the configuration to a YAML file
func (c *Config) SaveFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// DefaultLogger creates a simple default logger
func NewDefaultLogger() Logger {
	return &defaultLogger{}
}

// defaultLogger is a simple logger implementation
type defaultLogger struct{}

func (l *defaultLogger) Debug(msg string, fields ...interface{}) {
	fmt.Printf("[DEBUG] %s %v\n", msg, fields)
}

func (l *defaultLogger) Info(msg string, fields ...interface{}) {
	fmt.Printf("[INFO] %s %v\n", msg, fields)
}

func (l *defaultLogger) Warn(msg string, fields ...interface{}) {
	fmt.Printf("[WARN] %s %v\n", msg, fields)
}

func (l *defaultLogger) Error(msg string, fields ...interface{}) {
	fmt.Printf("[ERROR] %s %v\n", msg, fields)
}
