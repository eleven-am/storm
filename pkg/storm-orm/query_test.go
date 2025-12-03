package orm

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestQueryContext tests QueryContext method
func TestQueryContext(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("QueryContext with custom context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "test-key", "test-value")

		mock.ExpectQuery(`SELECT .* FROM users`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}))

		query := repo.Query(ctx)
		assert.NotNil(t, query)
		assert.Equal(t, ctx, query.ctx)

		_, err := query.Find()
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestQueryWithTx tests query with transaction
func TestQueryWithTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("Query with transaction", func(t *testing.T) {
		mock.ExpectBegin()
		tx, err := sqlxDB.Beginx()
		require.NoError(t, err)

		mock.ExpectQuery(`SELECT .* FROM users`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}))

		query := repo.Query(context.Background()).WithTx(tx)
		assert.NotNil(t, query.tx)

		_, err = query.Find()
		require.NoError(t, err)

		mock.ExpectRollback()
		tx.Rollback()

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestQueryOrderBy tests OrderBy functionality
func TestQueryOrderBy(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("OrderBy single column", func(t *testing.T) {

		mock.ExpectQuery(`SELECT .* FROM users ORDER BY users.name ASC`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}))

		nameCol := Column[string]{Name: "name", Table: "users"}
		_, err := repo.Query(context.Background()).OrderBy(nameCol.Asc()).Find()
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("OrderBy multiple columns", func(t *testing.T) {

		mock.ExpectQuery(`SELECT .* FROM users ORDER BY users.is_active DESC, users.name ASC`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}))

		activeCol := Column[bool]{Name: "is_active", Table: "users"}
		nameCol := Column[string]{Name: "name", Table: "users"}
		_, err := repo.Query(context.Background()).OrderBy(activeCol.Desc(), nameCol.Asc()).Find()
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestQueryLimit tests Limit and Offset functionality
func TestQueryLimitOffset(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("Query with Limit", func(t *testing.T) {

		mock.ExpectQuery(`SELECT .* FROM users LIMIT 10`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}))

		_, err := repo.Query(context.Background()).Limit(10).Find()
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Query with Limit and Offset", func(t *testing.T) {

		mock.ExpectQuery(`SELECT .* FROM users LIMIT 10 OFFSET 20`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}))

		_, err := repo.Query(context.Background()).Limit(10).Offset(20).Find()
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestQueryFirst tests First method
func TestQueryFirst(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("First with result", func(t *testing.T) {
		now := time.Now()

		mock.ExpectQuery(`SELECT .* FROM users LIMIT 1`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}).
				AddRow(1, "John Doe", "john@example.com", true, now, now))

		user, err := repo.Query(context.Background()).First()
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, "John Doe", user.Name)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("First with no result", func(t *testing.T) {

		mock.ExpectQuery(`SELECT .* FROM users LIMIT 1`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}))

		user, err := repo.Query(context.Background()).First()
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "not found")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestQueryExists tests Exists method
func TestQueryExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("Exists returns true", func(t *testing.T) {

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		exists, err := repo.Query(context.Background()).Exists()
		require.NoError(t, err)
		assert.True(t, exists)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Exists returns false", func(t *testing.T) {

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		exists, err := repo.Query(context.Background()).Exists()
		require.NoError(t, err)
		assert.False(t, exists)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestQueryDelete tests Delete method on query
func TestQueryDelete(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("Delete with conditions", func(t *testing.T) {

		mock.ExpectExec(`DELETE FROM users WHERE`).
			WithArgs(false).
			WillReturnResult(sqlmock.NewResult(0, 5))

		activeCol := Column[bool]{Name: "is_active", Table: "users"}
		rowsAffected, err := repo.Query(context.Background()).Where(activeCol.Eq(false)).Delete()
		require.NoError(t, err)
		assert.Equal(t, int64(5), rowsAffected)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Delete with no matches", func(t *testing.T) {

		mock.ExpectExec(`DELETE FROM users WHERE`).
			WithArgs("nonexistent@example.com").
			WillReturnResult(sqlmock.NewResult(0, 0))

		emailCol := Column[string]{Name: "email", Table: "users"}
		rowsAffected, err := repo.Query(context.Background()).Where(emailCol.Eq("nonexistent@example.com")).Delete()
		require.NoError(t, err)
		assert.Equal(t, int64(0), rowsAffected)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestQueryJoins tests join methods
func TestQueryJoins(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("InnerJoin", func(t *testing.T) {

		mock.ExpectQuery(`SELECT .* FROM users INNER JOIN posts ON posts.user_id = users.id`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}))

		_, err := repo.Query(context.Background()).InnerJoin("posts", "posts.user_id = users.id").Find()
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("LeftJoin", func(t *testing.T) {

		mock.ExpectQuery(`SELECT .* FROM users LEFT JOIN posts ON posts.user_id = users.id`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}))

		_, err := repo.Query(context.Background()).LeftJoin("posts", "posts.user_id = users.id").Find()
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("RightJoin", func(t *testing.T) {

		mock.ExpectQuery(`SELECT .* FROM users RIGHT JOIN posts ON posts.user_id = users.id`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}))

		_, err := repo.Query(context.Background()).RightJoin("posts", "posts.user_id = users.id").Find()
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FullJoin", func(t *testing.T) {

		mock.ExpectQuery(`SELECT .* FROM users JOIN FULL OUTER JOIN posts ON posts.user_id = users.id`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}))

		_, err := repo.Query(context.Background()).FullJoin("posts", "posts.user_id = users.id").Find()
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestQueryInclude tests Include functionality
func TestQueryInclude(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("Include single relationship", func(t *testing.T) {

		query := repo.Query(context.Background()).Include("posts")
		assert.Len(t, query.includes, 1)
		assert.Equal(t, "posts", query.includes[0].name)
	})

	t.Run("Include multiple relationships", func(t *testing.T) {

		query := repo.Query(context.Background()).Include("posts", "comments", "tags")
		assert.Len(t, query.includes, 3)
		assert.Equal(t, "posts", query.includes[0].name)
		assert.Equal(t, "comments", query.includes[1].name)
		assert.Equal(t, "tags", query.includes[2].name)
	})
}

// TestQueryExecuteRaw tests ExecuteRaw method
func TestQueryExecuteRaw(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("ExecuteRaw with custom SQL", func(t *testing.T) {
		now := time.Now()

		mock.ExpectQuery(`SELECT \* FROM users WHERE created_at > \$1`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}).
				AddRow(1, "John Doe", "john@example.com", true, now, now))

		query := repo.Query(context.Background())
		users, err := query.ExecuteRaw("SELECT * FROM users WHERE created_at > $1", now.Add(-24*time.Hour))
		require.NoError(t, err)
		assert.Len(t, users, 1)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ExecuteRaw with error", func(t *testing.T) {

		mock.ExpectQuery(`SELECT \* FROM invalid_table`).
			WillReturnError(sql.ErrNoRows)

		query := repo.Query(context.Background())
		users, err := query.ExecuteRaw("SELECT * FROM invalid_table")
		assert.Error(t, err)
		assert.Nil(t, users)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}
