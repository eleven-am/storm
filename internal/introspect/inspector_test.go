package introspect

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNewInspector(t *testing.T) {
	var db *sql.DB

	inspector := NewInspector(db, "postgres")

	if inspector == nil {
		t.Fatal("Expected inspector to be created")
	}

	if inspector.driver != "postgres" {
		t.Errorf("Expected driver to be 'postgres', got %s", inspector.driver)
	}
}

func TestInspector_UnsupportedDriver(t *testing.T) {
	var db *sql.DB
	inspector := NewInspector(db, "mysql")

	ctx := context.Background()

	_, err := inspector.GetSchema(ctx)
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}
	if err.Error() != "unsupported database driver: mysql" {
		t.Errorf("Unexpected error message: %v", err)
	}

	_, err = inspector.GetTable(ctx, "public", "users")
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}

	_, err = inspector.GetTables(ctx)
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}

	_, err = inspector.GetDatabaseMetadata(ctx)
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}

	_, err = inspector.GetEnums(ctx)
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}

	_, err = inspector.GetFunctions(ctx)
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}

	_, err = inspector.GetSequences(ctx)
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}

	_, err = inspector.GetViews(ctx)
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}

	_, err = inspector.GetTableStatistics(ctx, "public", "users")
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}
}

func TestInspector_PostgresDriver(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	inspector := NewInspector(db, "postgres")
	ctx := context.Background()

	t.Run("GetSchema", func(t *testing.T) {

		mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"schema_name"}).AddRow("public"))

		_, err := inspector.GetSchema(ctx)

		_ = err
	})

	t.Run("GetTable", func(t *testing.T) {

		mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"table_name"}).AddRow("users"))

		_, err := inspector.GetTable(ctx, "public", "users")

		_ = err
	})

	t.Run("GetTables", func(t *testing.T) {

		mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"table_name"}).AddRow("users"))

		_, err := inspector.GetTables(ctx)

		_ = err
	})

	t.Run("GetDatabaseMetadata", func(t *testing.T) {

		mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("14.0"))

		_, err := inspector.GetDatabaseMetadata(ctx)

		_ = err
	})

	t.Run("GetEnums", func(t *testing.T) {

		mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"enum_name"}).AddRow("status"))

		_, err := inspector.GetEnums(ctx)

		_ = err
	})

	t.Run("GetFunctions", func(t *testing.T) {

		mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"function_name"}).AddRow("test_func"))

		_, err := inspector.GetFunctions(ctx)

		_ = err
	})

	t.Run("GetSequences", func(t *testing.T) {

		mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"sequence_name"}).AddRow("users_id_seq"))

		_, err := inspector.GetSequences(ctx)

		_ = err
	})

	t.Run("GetViews", func(t *testing.T) {

		mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"view_name"}).AddRow("user_view"))

		_, err := inspector.GetViews(ctx)

		_ = err
	})

	t.Run("GetTableStatistics", func(t *testing.T) {

		mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"reltuples"}).AddRow(100))

		_, err := inspector.GetTableStatistics(ctx, "public", "users")

		_ = err
	})
}
