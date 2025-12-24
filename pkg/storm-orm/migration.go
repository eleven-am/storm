package orm

import (
	"context"
	"strings"
	"time"

	"ariga.io/atlas/sql/schema"
	"github.com/eleven-am/storm/internal/migrator"
)

type ColumnChange struct {
	Table  string
	Column string
	Type   string
}

type MigrationResult struct {
	TablesCreated  []string
	TablesDropped  []string
	ColumnsAdded   []ColumnChange
	ColumnsDropped []ColumnChange
	IndexesCreated []string
	IndexesDropped []string
	SQL            []string
	Duration       time.Duration
}

type migrationOptions struct {
	allowDestructive bool
	dryRun           bool
}

func (s *Storm) AutoMigrate(ctx context.Context, config Config) (MigrationResult, error) {
	return s.runMigration(ctx, config, migrationOptions{
		allowDestructive: false,
		dryRun:           false,
	})
}

func (s *Storm) AutoMigrateDryRun(ctx context.Context, config Config) (MigrationResult, error) {
	return s.runMigration(ctx, config, migrationOptions{
		allowDestructive: false,
		dryRun:           true,
	})
}

func (s *Storm) AutoMigrateDestructive(ctx context.Context, config Config) (MigrationResult, error) {
	return s.runMigration(ctx, config, migrationOptions{
		allowDestructive: true,
		dryRun:           false,
	})
}

func (s *Storm) runMigration(ctx context.Context, config Config, opts migrationOptions) (MigrationResult, error) {
	config.validate()
	config.applyDefaults()

	start := time.Now()

	db := s.GetDB()
	if db == nil {
		panic("stormorm: database connection is nil")
	}

	atlasMigrator := migrator.NewAtlasMigrator(migrator.NewDBConfig(config.DatabaseURL))

	migratorOpts := migrator.MigrationOptions{
		PackagePath:         config.ModelsPackage,
		DryRun:              opts.dryRun,
		AllowDestructive:    opts.allowDestructive,
		PushToDB:            !opts.dryRun,
		CreateDBIfNotExists: false,
	}

	result, err := atlasMigrator.GenerateMigration(ctx, db.DB, migratorOpts)
	if err != nil {
		return MigrationResult{}, err
	}

	migrationResult := buildMigrationResult(result, start)
	return migrationResult, nil
}

func buildMigrationResult(result *migrator.MigrationResult, start time.Time) MigrationResult {
	mr := MigrationResult{
		Duration: time.Since(start),
	}

	if result == nil || len(result.Changes) == 0 {
		return mr
	}

	mr.SQL = extractSQLStatements(result.UpSQL)

	for _, change := range result.Changes {
		processChange(change, &mr)
	}

	return mr
}

func extractSQLStatements(upSQL string) []string {
	if upSQL == "" {
		return nil
	}

	var statements []string
	lines := strings.Split(upSQL, "\n")
	var current strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		current.WriteString(line)
		current.WriteString("\n")
		if strings.HasSuffix(trimmed, ";") {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
		}
	}

	if current.Len() > 0 {
		stmt := strings.TrimSpace(current.String())
		if stmt != "" {
			statements = append(statements, stmt)
		}
	}

	return statements
}

func processChange(change schema.Change, mr *MigrationResult) {
	switch c := change.(type) {
	case *schema.AddTable:
		mr.TablesCreated = append(mr.TablesCreated, c.T.Name)
	case *schema.DropTable:
		mr.TablesDropped = append(mr.TablesDropped, c.T.Name)
	case *schema.ModifyTable:
		processTableChanges(c, mr)
	case *schema.AddColumn:
		mr.ColumnsAdded = append(mr.ColumnsAdded, ColumnChange{
			Column: c.C.Name,
			Type:   c.C.Type.Raw,
		})
	case *schema.DropColumn:
		mr.ColumnsDropped = append(mr.ColumnsDropped, ColumnChange{
			Column: c.C.Name,
		})
	case *schema.AddIndex:
		mr.IndexesCreated = append(mr.IndexesCreated, c.I.Name)
	case *schema.DropIndex:
		mr.IndexesDropped = append(mr.IndexesDropped, c.I.Name)
	}
}

func processTableChanges(modifyTable *schema.ModifyTable, mr *MigrationResult) {
	tableName := modifyTable.T.Name

	for _, change := range modifyTable.Changes {
		switch c := change.(type) {
		case *schema.AddColumn:
			mr.ColumnsAdded = append(mr.ColumnsAdded, ColumnChange{
				Table:  tableName,
				Column: c.C.Name,
				Type:   c.C.Type.Raw,
			})
		case *schema.DropColumn:
			mr.ColumnsDropped = append(mr.ColumnsDropped, ColumnChange{
				Table:  tableName,
				Column: c.C.Name,
			})
		case *schema.AddIndex:
			mr.IndexesCreated = append(mr.IndexesCreated, c.I.Name)
		case *schema.DropIndex:
			mr.IndexesDropped = append(mr.IndexesDropped, c.I.Name)
		}
	}
}
