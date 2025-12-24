package storm

import (
	"fmt"
	"time"
)

// Option configures Storm
type Option func(*Config) error

// WithDriver sets the database driver
func WithDriver(driver string) Option {
	return func(c *Config) error {
		if driver == "" {
			return fmt.Errorf("driver cannot be empty")
		}
		c.Driver = driver
		return nil
	}
}

// WithMaxConnections sets connection pool size
func WithMaxConnections(max int) Option {
	return func(c *Config) error {
		if max < 1 {
			return fmt.Errorf("max connections must be at least 1")
		}
		c.MaxOpenConns = max
		c.MaxIdleConns = max / 4
		return nil
	}
}

// WithMaxIdleConnections sets max idle connections
func WithMaxIdleConnections(max int) Option {
	return func(c *Config) error {
		if max < 0 {
			return fmt.Errorf("max idle connections cannot be negative")
		}
		c.MaxIdleConns = max
		return nil
	}
}

// WithConnMaxLifetime sets connection lifetime
func WithConnMaxLifetime(d time.Duration) Option {
	return func(c *Config) error {
		if d <= 0 {
			return fmt.Errorf("connection max lifetime must be positive")
		}
		c.ConnMaxLifetime = d
		return nil
	}
}

// WithModelsPackage sets the models package path
func WithModelsPackage(path string) Option {
	return func(c *Config) error {
		if path == "" {
			return fmt.Errorf("models package cannot be empty")
		}
		c.ModelsPackage = path
		return nil
	}
}

// WithMigrationsDir sets the migrations directory
func WithMigrationsDir(dir string) Option {
	return func(c *Config) error {
		if dir == "" {
			return fmt.Errorf("migrations directory cannot be empty")
		}
		c.MigrationsDir = dir
		return nil
	}
}

// WithMigrationsTable sets the migrations table name
func WithMigrationsTable(table string) Option {
	return func(c *Config) error {
		if table == "" {
			return fmt.Errorf("migrations table cannot be empty")
		}
		c.MigrationsTable = table
		return nil
	}
}

// WithAutoMigrate enables automatic migrations
func WithAutoMigrate(enabled bool) Option {
	return func(c *Config) error {
		c.AutoMigrate = enabled
		return nil
	}
}

// WithAutoMigrateOptions configures automatic migration behavior
func WithAutoMigrateOptions(opts AutoMigrateOptions) Option {
	return func(c *Config) error {
		c.AutoMigrateOpts = opts
		return nil
	}
}

// WithGenerateHooks enables hook generation
func WithGenerateHooks(enabled bool) Option {
	return func(c *Config) error {
		c.GenerateHooks = enabled
		return nil
	}
}

// WithGenerateTests enables test generation
func WithGenerateTests(enabled bool) Option {
	return func(c *Config) error {
		c.GenerateTests = enabled
		return nil
	}
}

// WithGenerateMocks enables mock generation
func WithGenerateMocks(enabled bool) Option {
	return func(c *Config) error {
		c.GenerateMocks = enabled
		return nil
	}
}

// WithStrictMode enables strict mode
func WithStrictMode(enabled bool) Option {
	return func(c *Config) error {
		c.StrictMode = enabled
		return nil
	}
}

// WithNamingConvention sets the naming convention
func WithNamingConvention(convention string) Option {
	return func(c *Config) error {
		if convention != "snake_case" && convention != "camelCase" {
			return fmt.Errorf("naming convention must be 'snake_case' or 'camelCase'")
		}
		c.NamingConvention = convention
		return nil
	}
}

// WithLogger sets a custom logger
func WithLogger(logger Logger) Option {
	return func(c *Config) error {
		if logger == nil {
			return fmt.Errorf("logger cannot be nil")
		}
		c.Logger = logger
		return nil
	}
}

// WithDebug enables debug mode
func WithDebug(enabled bool) Option {
	return func(c *Config) error {
		c.Debug = enabled
		return nil
	}
}

// WithConfigFile loads configuration from file
func WithConfigFile(path string) Option {
	return func(c *Config) error {
		if path == "" {
			return fmt.Errorf("config file path cannot be empty")
		}
		return c.LoadFile(path)
	}
}

// WithConfig merges another config
func WithConfig(other *Config) Option {
	return func(c *Config) error {
		if other == nil {
			return fmt.Errorf("config cannot be nil")
		}

		if other.Driver != "" {
			c.Driver = other.Driver
		}
		if other.DatabaseURL != "" {
			c.DatabaseURL = other.DatabaseURL
		}
		if other.MaxOpenConns > 0 {
			c.MaxOpenConns = other.MaxOpenConns
		}
		if other.MaxIdleConns > 0 {
			c.MaxIdleConns = other.MaxIdleConns
		}
		if other.ConnMaxLifetime > 0 {
			c.ConnMaxLifetime = other.ConnMaxLifetime
		}
		if other.ModelsPackage != "" {
			c.ModelsPackage = other.ModelsPackage
		}
		if other.MigrationsDir != "" {
			c.MigrationsDir = other.MigrationsDir
		}
		if other.MigrationsTable != "" {
			c.MigrationsTable = other.MigrationsTable
		}
		if other.NamingConvention != "" {
			c.NamingConvention = other.NamingConvention
		}
		if other.Logger != nil {
			c.Logger = other.Logger
		}

		c.AutoMigrate = other.AutoMigrate
		c.GenerateHooks = other.GenerateHooks
		c.GenerateTests = other.GenerateTests
		c.GenerateMocks = other.GenerateMocks
		c.StrictMode = other.StrictMode
		c.Debug = other.Debug

		return nil
	}
}
