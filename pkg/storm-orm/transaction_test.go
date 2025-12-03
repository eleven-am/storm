package orm

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTransactionMethods tests transaction-related methods
func TestTransactionMethods(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("IsTransaction on regular repository", func(t *testing.T) {
		isTransaction := repo.IsTransaction()
		assert.False(t, isTransaction)
	})

	t.Run("GetTransactionManager", func(t *testing.T) {
		txManager, err := repo.GetTransactionManager()
		require.NoError(t, err)
		assert.NotNil(t, txManager)
		assert.Equal(t, sqlxDB, txManager.db)
	})

	t.Run("WithinTransaction success", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectCommit()

		executed := false
		err := repo.WithinTransaction(context.Background(), func(tx *sqlx.Tx) error {
			executed = true
			assert.NotNil(t, tx)
			return nil
		})

		require.NoError(t, err)
		assert.True(t, executed)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("WithinTransaction with error", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectRollback()

		err := repo.WithinTransaction(context.Background(), func(tx *sqlx.Tx) error {
			return assert.AnError
		})

		assert.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("WithinTransaction with panic", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectRollback()

		defer func() {
			if r := recover(); r != nil {

				assert.Equal(t, "test panic", r)
			}
		}()

		repo.WithinTransaction(context.Background(), func(tx *sqlx.Tx) error {
			panic("test panic")
		})

		t.Fatal("Expected panic but none occurred")
	})
}

// TestNewRepositoryWithTx tests creating repository with transaction
func TestNewRepositoryWithTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	t.Run("Valid transaction", func(t *testing.T) {
		mock.ExpectBegin()
		tx, err := sqlxDB.Beginx()
		require.NoError(t, err)

		repo, err := NewRepositoryWithTx[TestUser](tx, metadata)
		require.NoError(t, err)
		assert.NotNil(t, repo)

		isTransaction := repo.IsTransaction()
		assert.True(t, isTransaction)

		mock.ExpectRollback()
		tx.Rollback()
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestTransactionManager tests the TransactionManager
func TestTransactionManager(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")

	t.Run("NewTransactionManager", func(t *testing.T) {
		tm := NewTransactionManager(sqlxDB)
		assert.NotNil(t, tm)
		assert.Equal(t, sqlxDB, tm.db)
	})

	t.Run("WithTransaction success", func(t *testing.T) {
		tm := NewTransactionManager(sqlxDB)

		mock.ExpectBegin()
		mock.ExpectCommit()

		executed := false
		err := tm.WithTransaction(context.Background(), func(tx *sqlx.Tx) error {
			executed = true
			return nil
		})

		require.NoError(t, err)
		assert.True(t, executed)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("WithTransaction error rolls back", func(t *testing.T) {
		tm := NewTransactionManager(sqlxDB)

		mock.ExpectBegin()
		mock.ExpectRollback()

		err := tm.WithTransaction(context.Background(), func(tx *sqlx.Tx) error {
			return assert.AnError
		})

		assert.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("WithTransactionOptions", func(t *testing.T) {
		tm := NewTransactionManager(sqlxDB)

		opts := &TransactionOptions{
			Isolation: sql.LevelReadCommitted,
		}

		mock.ExpectBegin()
		mock.ExpectCommit()

		executed := false
		err := tm.WithTransactionOptions(context.Background(), opts, func(tx *sqlx.Tx) error {
			executed = true
			return nil
		})

		require.NoError(t, err)
		assert.True(t, executed)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
