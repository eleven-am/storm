package migrator

import (
	"strings"
	"testing"
)

func TestMigrationReverser_CreateTableWithConstraints(t *testing.T) {
	reverser := NewMigrationReverser()

	t.Run("CREATE TABLE with constraints", func(t *testing.T) {
		sql := `CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)`

		got, err := reverser.ReverseSQL(sql)
		if err != nil {
			t.Errorf("ReverseSQL() error = %v", err)
		}

		if !strings.Contains(got, "DROP TABLE") {
			t.Errorf("Expected DROP TABLE in result, got: %s", got)
		}
	})
}

func TestMigrationReverser_CreateIndexWithError(t *testing.T) {
	reverser := NewMigrationReverser()

	t.Run("CREATE INDEX with invalid regex", func(t *testing.T) {
		sql := `CREATE INDEX ON users`

		got, err := reverser.ReverseSQL(sql)
		if err != nil {
			t.Errorf("ReverseSQL() error = %v", err)
		}

		if !strings.Contains(got, "DROP INDEX") {
			t.Errorf("Expected DROP INDEX in result, got: %s", got)
		}
	})
}

func TestMigrationReverser_CreateSequenceWithError(t *testing.T) {
	reverser := NewMigrationReverser()

	t.Run("CREATE SEQUENCE with invalid regex", func(t *testing.T) {
		sql := `CREATE SEQUENCE`

		got, err := reverser.ReverseSQL(sql)
		if err == nil {
			t.Error("Expected error for invalid CREATE SEQUENCE")
		}

		if got != "" {
			t.Errorf("Expected empty result for error case, got: %s", got)
		}
	})
}

func TestMigrationReverser_CreateTypeWithError(t *testing.T) {
	reverser := NewMigrationReverser()

	t.Run("CREATE TYPE with invalid regex", func(t *testing.T) {
		sql := `CREATE TYPE`

		got, err := reverser.ReverseSQL(sql)
		if err == nil {
			t.Error("Expected error for invalid CREATE TYPE")
		}

		if got != "" {
			t.Errorf("Expected empty result for error case, got: %s", got)
		}
	})
}

func TestMigrationReverser_CreateFunctionWithError(t *testing.T) {
	reverser := NewMigrationReverser()

	t.Run("CREATE FUNCTION with invalid regex", func(t *testing.T) {
		sql := `CREATE FUNCTION`

		got, err := reverser.ReverseSQL(sql)
		if err == nil {
			t.Error("Expected error for invalid CREATE FUNCTION")
		}

		if got != "" {
			t.Errorf("Expected empty result for error case, got: %s", got)
		}
	})
}

func TestMigrationReverser_CreateTriggerWithError(t *testing.T) {
	reverser := NewMigrationReverser()

	t.Run("CREATE TRIGGER with invalid regex", func(t *testing.T) {
		sql := `CREATE TRIGGER`

		got, err := reverser.ReverseSQL(sql)
		if err == nil {
			t.Error("Expected error for invalid CREATE TRIGGER")
		}

		if got != "" {
			t.Errorf("Expected empty result for error case, got: %s", got)
		}
	})
}

func TestMigrationReverser_AlterTableWithError(t *testing.T) {
	reverser := NewMigrationReverser()

	t.Run("ALTER TABLE with invalid regex", func(t *testing.T) {
		sql := `ALTER TABLE users RENAME COLUMN`

		got, err := reverser.ReverseSQL(sql)
		if err == nil {
			t.Error("Expected error for invalid RENAME COLUMN")
		}

		if got != "" {
			t.Errorf("Expected empty result for error case, got: %s", got)
		}
	})
}

func TestMigrationReverser_RenameTableWithError(t *testing.T) {
	reverser := NewMigrationReverser()

	t.Run("RENAME TABLE with invalid regex", func(t *testing.T) {
		sql := `ALTER TABLE users RENAME TO`

		got, err := reverser.ReverseSQL(sql)
		if err == nil {
			t.Error("Expected error for invalid RENAME TO")
		}

		if got != "" {
			t.Errorf("Expected empty result for error case, got: %s", got)
		}
	})
}
