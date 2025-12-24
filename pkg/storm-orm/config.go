package orm

const defaultMigrationsTable = "schema_migrations"

type Config struct {
	DatabaseURL     string
	ModelsPackage   string
	MigrationsTable string
}

func (c *Config) validate() {
	if c.DatabaseURL == "" {
		panic("stormorm: DatabaseURL is required")
	}
	if c.ModelsPackage == "" {
		panic("stormorm: ModelsPackage is required")
	}
}

func (c *Config) applyDefaults() {
	if c.MigrationsTable == "" {
		c.MigrationsTable = defaultMigrationsTable
	}
}
