package storm

import (
	"context"
	"time"
)

// Migrator handles database migrations
type Migrator interface {
	// Generate creates a new migration by comparing models with database
	Generate(ctx context.Context, opts MigrateOptions) (*Migration, error)

	// Apply executes a migration
	Apply(ctx context.Context, migration *Migration) error

	// Rollback reverses a migration
	Rollback(ctx context.Context, migration *Migration) error

	// Status returns the current migration status
	Status(ctx context.Context) (*MigrationStatus, error)

	// History returns all applied migrations
	History(ctx context.Context) ([]*MigrationRecord, error)

	// Pending returns all pending migrations
	Pending(ctx context.Context) ([]*Migration, error)

	// AutoMigrate reads Go structs and applies schema changes directly to the database
	AutoMigrate(ctx context.Context, opts AutoMigrateOptions) error
}

// SchemaInspector analyzes database schema
type SchemaInspector interface {
	// Inspect returns the current database schema
	Inspect(ctx context.Context) (*Schema, error)

	// Compare compares two schemas
	Compare(ctx context.Context, from, to *Schema) (*SchemaDiff, error)

	// ExportSQL exports schema as SQL
	ExportSQL(ctx context.Context) (string, error)

	// ExportGo exports schema as Go structs
	ExportGo(ctx context.Context) (string, error)
}

// Logger defines logging interface
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// Migration represents a database migration
type Migration struct {
	ID          string
	Name        string
	Version     string
	Description string
	UpSQL       string
	DownSQL     string
	Checksum    string
	CreatedAt   time.Time
}

// MigrationStatus represents current migration state
type MigrationStatus struct {
	Current   string
	Available int
	Pending   int
	Applied   int
}

// MigrationRecord represents an applied migration
type MigrationRecord struct {
	ID        string
	Version   string
	AppliedAt time.Time
	AppliedBy string
	Duration  time.Duration
	Success   bool
	Error     string
}

// Schema represents a database schema
type Schema struct {
	Tables      map[string]*Table
	Views       []*View
	Functions   []*Function
	Indexes     []*Index
	Constraints []*Constraint
	Enums       []*Enum
}

// SchemaDiff represents differences between schemas
type SchemaDiff struct {
	AddedTables    map[string]*Table
	DroppedTables  map[string]*Table
	ModifiedTables map[string]*TableDiff
}

// TableDiff represents differences between table schemas
type TableDiff struct {
	AddedColumns    map[string]*Column
	DroppedColumns  map[string]*Column
	ModifiedColumns map[string]*ColumnDiff
}

// ColumnDiff represents differences between column schemas
type ColumnDiff struct {
	TypeChanged     bool
	OldType         string
	NewType         string
	NullableChanged bool
	OldNullable     bool
	NewNullable     bool
	DefaultChanged  bool
	OldDefault      string
	NewDefault      string
}

// IsEmpty returns true if the table diff has no changes
func (td *TableDiff) IsEmpty() bool {
	return len(td.AddedColumns) == 0 && len(td.DroppedColumns) == 0 && len(td.ModifiedColumns) == 0
}

// IsEmpty returns true if the column diff has no changes
func (cd *ColumnDiff) IsEmpty() bool {
	return !cd.TypeChanged && !cd.NullableChanged && !cd.DefaultChanged
}

// Table represents a database table
type Table struct {
	Name        string
	Schema      string
	Columns     map[string]*Column
	PrimaryKey  *PrimaryKey
	ForeignKeys []*ForeignKey
	Indexes     []*Index
	Constraints []*Constraint
}

// Column represents a table column
type Column struct {
	Name         string
	Type         string
	Nullable     bool
	Default      string
	Length       int
	Precision    int
	Scale        int
	IsPrimaryKey bool
	IsForeignKey bool
}

// PrimaryKey represents a primary key constraint
type PrimaryKey struct {
	Name    string
	Columns []string
}

// ForeignKey represents a foreign key constraint
type ForeignKey struct {
	Name           string
	Columns        []string
	ForeignTable   string
	ForeignColumns []string
}

// View represents a database view
type View struct {
	Name       string
	Schema     string
	Definition string
}

// Function represents a database function
type Function struct {
	Name       string
	Schema     string
	Definition string
}

// Index represents a database index
type Index struct {
	Name    string
	Table   string
	Columns []string
	Unique  bool
}

// Constraint represents a database constraint
type Constraint struct {
	Name       string
	Table      string
	Type       string
	Definition string
}

// Enum represents a database enum type
type Enum struct {
	Name   string
	Values []string
}

// MigrateOptions configures migration generation
type MigrateOptions struct {
	Name                string
	PackagePath         string
	OutputDir           string
	DryRun              bool
	AllowDestructive    bool
	SkipPrompt          bool
	CreateDBIfNotExists bool
}

// AutoMigrateOptions configures automatic schema migration
type AutoMigrateOptions struct {
	AllowDestructive    bool
	DryRun              bool
	CreateDBIfNotExists bool
	LockTimeout         time.Duration
}

// GenerateOptions configures ORM code generation
type GenerateOptions struct {
	PackagePath  string
	OutputDir    string
	IncludeHooks bool
	IncludeTests bool
	IncludeMocks bool
}
