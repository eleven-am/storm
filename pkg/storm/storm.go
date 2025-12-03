package storm

import (
	"context"
	"fmt"
	"sync"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// Storm is the main entry point for all database operations
type Storm struct {
	// Core components
	db     *sqlx.DB
	config *Config

	// Sub-systems
	migrator Migrator
	orm      *ORM
	schema   SchemaInspector

	// Internal state
	mu     sync.RWMutex
	closed bool
	logger Logger
}

// New creates a new Storm instance with the given database URL
func New(databaseURL string, opts ...Option) (*Storm, error) {
	config := NewConfig()
	config.DatabaseURL = databaseURL

	for _, opt := range opts {
		if err := opt(config); err != nil {
			return nil, NewConfigError("apply_option", err)
		}
	}

	return NewWithConfig(config)
}

// NewWithConfig creates a new Storm instance with explicit configuration
func NewWithConfig(config *Config) (*Storm, error) {
	if config == nil {
		return nil, NewConfigError("new_with_config", fmt.Errorf("config cannot be nil"))
	}

	if err := config.Validate(); err != nil {
		return nil, NewConfigError("validate", err)
	}

	db, err := sqlx.Open(config.Driver, config.DatabaseURL)
	if err != nil {
		return nil, NewConnectionError("open", err)
	}

	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)

	storm := &Storm{
		db:     db,
		config: config,
		logger: config.Logger,
	}

	if err := storm.initialize(); err != nil {
		db.Close()
		return nil, err
	}

	return storm, nil
}

// initialize sets up all sub-systems
func (s *Storm) initialize() error {
	if migrator, err := s.newMigrator(); err != nil {
		return NewMigrationError("initialize_migrator", err)
	} else {
		s.migrator = migrator
	}

	if orm, err := s.newORM(); err != nil {
		return NewORMError("initialize_orm", err)
	} else {
		s.orm = orm
	}

	if schema, err := s.newSchemaInspector(); err != nil {
		return NewSchemaError("initialize_schema", err)
	} else {
		s.schema = schema
	}

	return nil
}

// Factory functions for implementations
var (
	MigratorFactory        func(db *sqlx.DB, config *Config, logger Logger) Migrator
	ORMFactory             func(config *Config, logger Logger) ORMGenerator
	SchemaInspectorFactory func(db *sqlx.DB, config *Config, logger Logger) SchemaInspector
)

// newMigrator creates a new migrator instance
func (s *Storm) newMigrator() (Migrator, error) {
	if MigratorFactory != nil {
		return MigratorFactory(s.db, s.config, s.logger), nil
	}
	return &migrator{storm: s}, nil
}

// newORM creates a new ORM instance
func (s *Storm) newORM() (*ORM, error) {
	orm := &ORM{storm: s}
	if ORMFactory != nil {
		orm.impl = ORMFactory(s.config, s.logger)
	}
	return orm, nil
}

// newSchemaInspector creates a new schema inspector
func (s *Storm) newSchemaInspector() (SchemaInspector, error) {
	if SchemaInspectorFactory != nil {
		return SchemaInspectorFactory(s.db, s.config, s.logger), nil
	}
	return &schemaInspector{storm: s}, nil
}

// DB returns the underlying database connection
func (s *Storm) DB() *sqlx.DB {
	return s.db
}

// Config returns the current configuration
func (s *Storm) Config() *Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.Clone()
}

// Logger returns the current logger
func (s *Storm) Logger() Logger {
	return s.logger
}

// Migrator returns the migration interface
func (s *Storm) Migrator() Migrator {
	return s.migrator
}

// ORM returns the ORM interface
func (s *Storm) ORM() *ORM {
	return s.orm
}

// Schema returns the schema inspector
func (s *Storm) Schema() SchemaInspector {
	return s.schema
}

// Close closes all connections and cleans up resources
func (s *Storm) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	return s.db.Close()
}

// Ping verifies the database connection
func (s *Storm) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Migrate generates and optionally applies migrations
func (s *Storm) Migrate(ctx context.Context, opts ...MigrateOptions) error {
	var options MigrateOptions
	if len(opts) > 0 {
		options = opts[0]
	} else {
		options = MigrateOptions{
			PackagePath: s.config.ModelsPackage,
			OutputDir:   s.config.MigrationsDir,
			DryRun:      false,
		}
	}

	migration, err := s.migrator.Generate(ctx, options)
	if err != nil {
		return NewMigrationError("generate", err)
	}

	if options.DryRun {
		s.logger.Info("Dry run - migration would be:", "migration", migration.Name)
		return nil
	}

	return nil
}

// Generate creates ORM code from models
func (s *Storm) Generate(ctx context.Context, opts ...GenerateOptions) error {
	var options GenerateOptions
	if len(opts) > 0 {
		options = opts[0]
	} else {
		options = GenerateOptions{
			PackagePath:  s.config.ModelsPackage,
			IncludeHooks: s.config.GenerateHooks,
			IncludeTests: s.config.GenerateTests,
			IncludeMocks: s.config.GenerateMocks,
		}
	}

	return s.orm.Generate(ctx, options)
}

// Status returns the current migration status
func (s *Storm) Status(ctx context.Context) (*MigrationStatus, error) {
	return s.migrator.Status(ctx)
}

// Introspect analyzes the database schema
func (s *Storm) Introspect(ctx context.Context) (*Schema, error) {
	return s.schema.Inspect(ctx)
}

type migrator struct {
	storm *Storm
}

func (m *migrator) Generate(ctx context.Context, opts MigrateOptions) (*Migration, error) {
	return nil, ErrNotImplemented
}

func (m *migrator) Apply(ctx context.Context, migration *Migration) error {
	return ErrNotImplemented
}

func (m *migrator) Rollback(ctx context.Context, migration *Migration) error {
	return ErrNotImplemented
}

func (m *migrator) Status(ctx context.Context) (*MigrationStatus, error) {
	return nil, ErrNotImplemented
}

func (m *migrator) History(ctx context.Context) ([]*MigrationRecord, error) {
	return nil, ErrNotImplemented
}

func (m *migrator) Pending(ctx context.Context) ([]*Migration, error) {
	return nil, ErrNotImplemented
}

type ORM struct {
	storm *Storm
	impl  ORMGenerator
}

type ORMGenerator interface {
	Generate(ctx context.Context, opts GenerateOptions) error
}

func (o *ORM) Generate(ctx context.Context, opts GenerateOptions) error {
	if o.impl != nil {
		return o.impl.Generate(ctx, opts)
	}
	return ErrNotImplemented
}

type schemaInspector struct {
	storm *Storm
}

func (s *schemaInspector) Inspect(ctx context.Context) (*Schema, error) {
	return nil, ErrNotImplemented
}

func (s *schemaInspector) Compare(ctx context.Context, from, to *Schema) (*SchemaDiff, error) {
	return nil, ErrNotImplemented
}

func (s *schemaInspector) ExportSQL(ctx context.Context) (string, error) {
	return "", ErrNotImplemented
}

func (s *schemaInspector) ExportGo(ctx context.Context) (string, error) {
	return "", ErrNotImplemented
}
