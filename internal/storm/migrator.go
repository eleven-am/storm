package storm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eleven-am/storm/internal/generator"
	"github.com/eleven-am/storm/internal/migrator"
	"github.com/eleven-am/storm/internal/parser"
	"github.com/eleven-am/storm/pkg/storm"
	"github.com/jmoiron/sqlx"
)

// MigratorImpl implements the storm.Migrator interface
type MigratorImpl struct {
	db     *sqlx.DB
	config *storm.Config
	logger storm.Logger
}

func NewMigrator(db *sqlx.DB, config *storm.Config, logger storm.Logger) *MigratorImpl {
	return &MigratorImpl{
		db:     db,
		config: config,
		logger: logger,
	}
}

func (m *MigratorImpl) Generate(ctx context.Context, opts storm.MigrateOptions) (*storm.Migration, error) {
	m.logger.Info("Generating migration...", "package", opts.PackagePath)

	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create migrations directory: %w", err)
	}

	var currentSchema *storm.Schema
	var err error

	if opts.CreateDBIfNotExists {
		currentSchema, err = m.getCurrentSchemaOrEmpty(ctx)
	} else {
		currentSchema, err = m.getCurrentSchema(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get current schema: %w", err)
	}

	desiredSchema, err := m.getDesiredSchema(opts.PackagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get desired schema: %w", err)
	}

	migration, err := m.generateMigration(currentSchema, desiredSchema, opts.CreateDBIfNotExists)
	if err != nil {
		return nil, fmt.Errorf("failed to generate migration: %w", err)
	}

	return migration, nil
}

func (m *MigratorImpl) Apply(ctx context.Context, migration *storm.Migration) error {
	m.logger.Info("Applying migration...", "name", migration.Name)

	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	applied, err := m.isMigrationApplied(ctx, migration.Name)
	if err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	if applied {
		m.logger.Info("Migration already applied", "name", migration.Name)
		return nil
	}

	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	var rollback = func() { tx.Rollback() }
	defer func() {
		if rollback != nil {
			rollback()
		}
	}()

	if err := m.executeMigration(ctx, tx, migration); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	if err := m.recordMigration(ctx, tx, migration); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}
	rollback = nil

	m.logger.Info("Migration applied successfully", "name", migration.Name)
	return nil
}

func (m *MigratorImpl) Rollback(ctx context.Context, migration *storm.Migration) error {
	m.logger.Info("Rolling back migration...", "name", migration.Name)

	applied, err := m.isMigrationApplied(ctx, migration.Name)
	if err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	if !applied {
		m.logger.Info("Migration not applied", "name", migration.Name)
		return nil
	}

	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	var rollback = func() { tx.Rollback() }
	defer func() {
		if rollback != nil {
			rollback()
		}
	}()

	if err := m.executeRollback(ctx, tx, migration); err != nil {
		return fmt.Errorf("failed to execute rollback: %w", err)
	}

	if err := m.removeMigrationRecord(ctx, tx, migration); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback: %w", err)
	}
	rollback = nil

	m.logger.Info("Migration rolled back successfully", "name", migration.Name)
	return nil
}

func (m *MigratorImpl) Status(ctx context.Context) (*storm.MigrationStatus, error) {
	if err := m.createMigrationsTable(ctx); err != nil {
		return nil, fmt.Errorf("failed to create migrations table: %w", err)
	}

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	pending, err := m.getPendingMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending migrations: %w", err)
	}

	return &storm.MigrationStatus{
		Applied:   len(applied),
		Pending:   len(pending),
		Available: len(applied) + len(pending),
		Current:   "",
	}, nil
}

func (m *MigratorImpl) History(ctx context.Context) ([]*storm.MigrationRecord, error) {
	if err := m.createMigrationsTable(ctx); err != nil {
		return nil, fmt.Errorf("failed to create migrations table: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT name, applied_at, checksum
		FROM %s
		ORDER BY applied_at DESC
	`, m.config.MigrationsTable)

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query migration history: %w", err)
	}
	defer rows.Close()

	var records []*storm.MigrationRecord
	for rows.Next() {
		var record storm.MigrationRecord
		var name, checksum string
		if err := rows.Scan(&name, &record.AppliedAt, &checksum); err != nil {
			return nil, fmt.Errorf("failed to scan migration record: %w", err)
		}
		record.ID = name
		record.Version = name
		record.Success = true
		records = append(records, &record)
	}

	return records, nil
}

func (m *MigratorImpl) Pending(ctx context.Context) ([]*storm.Migration, error) {
	return m.getPendingMigrations(ctx)
}

func (m *MigratorImpl) AutoMigrate(ctx context.Context, opts storm.AutoMigrateOptions) error {
	m.logger.Info("Starting auto-migration...", "package", m.config.ModelsPackage)

	lockTimeout := opts.LockTimeout
	if lockTimeout == 0 {
		lockTimeout = 30 * time.Second
	}

	lockCtx, cancel := context.WithTimeout(ctx, lockTimeout)
	defer cancel()

	const lockID = 8675309
	if err := m.acquireAdvisoryLock(lockCtx, lockID); err != nil {
		return fmt.Errorf("failed to acquire migration lock: %w", err)
	}
	defer m.releaseAdvisoryLock(ctx, lockID)

	m.logger.Info("Acquired migration lock, proceeding with auto-migration")

	atlasMigrator := NewAtlasMigrator(m.config.DatabaseURL)

	migrationOpts := MigrationOptions{
		PackagePath:         m.config.ModelsPackage,
		OutputDir:           "",
		DryRun:              opts.DryRun,
		AllowDestructive:    opts.AllowDestructive,
		PushToDB:            true,
		CreateDBIfNotExists: opts.CreateDBIfNotExists,
	}

	result, err := atlasMigrator.GenerateMigration(ctx, m.db.DB, migrationOpts)
	if err != nil {
		return fmt.Errorf("auto-migration failed: %w", err)
	}

	if len(result.Changes) == 0 {
		m.logger.Info("No schema changes detected, database is up to date")
	} else {
		m.logger.Info("Auto-migration completed successfully", "changes", len(result.Changes))
	}

	return nil
}

func (m *MigratorImpl) acquireAdvisoryLock(ctx context.Context, lockID int64) error {
	_, err := m.db.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockID)
	return err
}

func (m *MigratorImpl) releaseAdvisoryLock(ctx context.Context, lockID int64) {
	_, _ = m.db.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", lockID)
}

func (m *MigratorImpl) createMigrationsTable(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			name VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			checksum VARCHAR(64) NOT NULL
		)
	`, m.config.MigrationsTable)

	_, err := m.db.ExecContext(ctx, query)
	return err
}

func (m *MigratorImpl) isMigrationApplied(ctx context.Context, name string) (bool, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s WHERE name = $1
	`, m.config.MigrationsTable)

	var count int
	err := m.db.GetContext(ctx, &count, query, name)
	return count > 0, err
}

func (m *MigratorImpl) getAppliedMigrations(ctx context.Context) ([]string, error) {
	query := fmt.Sprintf(`
		SELECT name FROM %s ORDER BY applied_at
	`, m.config.MigrationsTable)

	var names []string
	err := m.db.SelectContext(ctx, &names, query)
	return names, err
}

func (m *MigratorImpl) getPendingMigrations(ctx context.Context) ([]*storm.Migration, error) {

	if err := m.createMigrationsTable(ctx); err != nil {
		return nil, fmt.Errorf("failed to create migrations table: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(m.config.MigrationsDir, "*.up.sql"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob migration files: %w", err)
	}

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedMap := make(map[string]bool)
	for _, name := range applied {
		appliedMap[name] = true
	}

	var pending []*storm.Migration
	for _, file := range files {
		name := filepath.Base(file)
		name = strings.TrimSuffix(name, ".up.sql")

		if !appliedMap[name] {
			migration, err := m.loadMigration(file)
			if err != nil {
				return nil, fmt.Errorf("failed to load migration %s: %w", name, err)
			}
			pending = append(pending, migration)
		}
	}

	return pending, nil
}

func (m *MigratorImpl) loadMigration(upFile string) (*storm.Migration, error) {

	upContent, err := os.ReadFile(upFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read up migration file: %w", err)
	}

	name := filepath.Base(upFile)
	name = strings.TrimSuffix(name, ".up.sql")

	downFile := strings.TrimSuffix(upFile, ".up.sql") + ".down.sql"
	downContent := ""
	if downBytes, err := os.ReadFile(downFile); err == nil {
		downContent = string(downBytes)
	}

	return &storm.Migration{
		Name:      name,
		UpSQL:     string(upContent),
		DownSQL:   downContent,
		Checksum:  m.calculateChecksum(string(upContent)),
		CreatedAt: time.Now(),
	}, nil
}

func (m *MigratorImpl) executeMigration(ctx context.Context, tx *sqlx.Tx, migration *storm.Migration) error {
	if migration.UpSQL == "" {
		return nil
	}

	statements := m.splitSQLStatements(migration.UpSQL)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if strings.Contains(strings.ToUpper(stmt), "CREATE DATABASE") {
			m.logger.Info("Skipping CREATE DATABASE statement in migration apply")
			continue
		}

		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to execute statement: %s: %w", stmt, err)
		}
	}

	return nil
}

// splitSQLStatements properly splits PostgreSQL statements, handling dollar-quoted strings
func (m *MigratorImpl) splitSQLStatements(sql string) []string {
	var statements []string
	var current strings.Builder
	inDollarQuote := false

	runes := []rune(sql)
	i := 0

	for i < len(runes) {
		char := runes[i]

		if char == '$' && i+1 < len(runes) && runes[i+1] == '$' {
			if !inDollarQuote {

				inDollarQuote = true
				current.WriteRune(char)
				current.WriteRune(runes[i+1])
				i += 2
				continue
			} else {

				inDollarQuote = false
				current.WriteRune(char)
				current.WriteRune(runes[i+1])
				i += 2
				continue
			}
		}

		if !inDollarQuote && char == ';' {
			current.WriteRune(char)
			stmt := strings.TrimSpace(current.String())

			if stmt != "" && !isOnlyComments(stmt) {
				statements = append(statements, stmt)
			}
			current.Reset()
			i++
			continue
		}

		current.WriteRune(char)
		i++
	}

	if current.Len() > 0 {
		stmt := strings.TrimSpace(current.String())
		if stmt != "" && !isOnlyComments(stmt) {
			statements = append(statements, stmt)
		}
	}

	return statements
}

// isOnlyComments checks if a statement contains only comments and whitespace
func isOnlyComments(stmt string) bool {
	lines := strings.Split(stmt, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "--") {
			return false
		}
	}
	return true
}

func (m *MigratorImpl) executeRollback(ctx context.Context, tx *sqlx.Tx, migration *storm.Migration) error {
	if migration.DownSQL == "" {
		return fmt.Errorf("no rollback script available for migration %s", migration.Name)
	}

	statements := strings.Split(migration.DownSQL, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to execute rollback statement: %s: %w", stmt, err)
		}
	}

	return nil
}

func (m *MigratorImpl) recordMigration(ctx context.Context, tx *sqlx.Tx, migration *storm.Migration) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (name, applied_at, checksum)
		VALUES ($1, $2, $3)
	`, m.config.MigrationsTable)

	_, err := tx.ExecContext(ctx, query, migration.Name, time.Now(), migration.Checksum)
	return err
}

func (m *MigratorImpl) removeMigrationRecord(ctx context.Context, tx *sqlx.Tx, migration *storm.Migration) error {
	query := fmt.Sprintf(`
		DELETE FROM %s WHERE name = $1
	`, m.config.MigrationsTable)

	_, err := tx.ExecContext(ctx, query, migration.Name)
	return err
}

func (m *MigratorImpl) getCurrentSchema(ctx context.Context) (*storm.Schema, error) {
	schemaInspector := NewSchemaInspector(m.db, m.config, m.logger)
	return schemaInspector.Inspect(ctx)
}

func (m *MigratorImpl) getCurrentSchemaOrEmpty(ctx context.Context) (*storm.Schema, error) {

	currentSchema, err := m.getCurrentSchema(ctx)
	if err != nil {

		if strings.Contains(err.Error(), "does not exist") {
			m.logger.Info("Database does not exist, using empty schema for migration generation")
			return &storm.Schema{
				Tables: make(map[string]*storm.Table),
			}, nil
		}
		return nil, err
	}
	return currentSchema, nil
}

func (m *MigratorImpl) getDesiredSchema(packagePath string) (*storm.Schema, error) {
	structParser := NewStructParser()
	models, err := structParser.ParseDirectory(packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse structs: %w", err)
	}

	schemaGenerator := NewSchemaGenerator()
	schema, err := schemaGenerator.GenerateSchema(models)
	if err != nil {
		return nil, fmt.Errorf("failed to generate schema: %w", err)
	}

	return m.convertGeneratorSchemaToStorm(schema), nil
}

func (m *MigratorImpl) generateMigration(current, desired *storm.Schema, createDBIfNotExists bool) (*storm.Migration, error) {
	atlasMigrator := NewAtlasMigrator(m.config.DatabaseURL)

	opts := MigrationOptions{
		PackagePath:         m.config.ModelsPackage,
		OutputDir:           m.config.MigrationsDir,
		DryRun:              false,
		AllowDestructive:    false,
		PushToDB:            false,
		CreateDBIfNotExists: createDBIfNotExists,
	}

	ctx := context.Background()
	result, err := atlasMigrator.GenerateMigration(ctx, m.db.DB, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate migration: %w", err)
	}

	timestamp := time.Now().Format("20060102150405")
	name := fmt.Sprintf("%s_auto_migration", timestamp)

	return &storm.Migration{
		Name:      name,
		UpSQL:     result.UpSQL,
		DownSQL:   result.DownSQL,
		Checksum:  m.calculateChecksum(result.UpSQL),
		CreatedAt: time.Now(),
	}, nil
}

// saveMigration removed - migration files are saved by AtlasMigrator

func (m *MigratorImpl) calculateChecksum(content string) string {
	return fmt.Sprintf("%x", len(content))
}

func NewStructParser() *parser.StructParser {
	return parser.NewStructParser()
}

func NewSchemaGenerator() *generator.SchemaGenerator {
	return generator.NewSchemaGenerator()
}

func NewAtlasMigrator(databaseURL string) *migrator.AtlasMigrator {
	config := migrator.NewDBConfig(databaseURL)
	return migrator.NewAtlasMigrator(config)
}

type MigrationOptions = migrator.MigrationOptions

func (m *MigratorImpl) convertGeneratorSchemaToStorm(genSchema *generator.DatabaseSchema) *storm.Schema {
	stormSchema := &storm.Schema{
		Tables: make(map[string]*storm.Table),
	}

	for tableName, table := range genSchema.Tables {
		stormTable := &storm.Table{
			Name:    table.Name,
			Columns: make(map[string]*storm.Column),
		}

		for _, col := range table.Columns {
			stormCol := &storm.Column{
				Name:     col.Name,
				Type:     col.Type,
				Nullable: col.IsNullable,
			}

			if col.DefaultValue != nil {
				stormCol.Default = *col.DefaultValue
			}

			stormTable.Columns[col.Name] = stormCol
		}

		stormSchema.Tables[tableName] = stormTable
	}

	return stormSchema
}
