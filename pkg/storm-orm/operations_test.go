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

// TestFindByID tests the FindByID operation
func TestFindByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("FindByID with existing record", func(t *testing.T) {
		userID := 1
		now := time.Now()

		mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}).
				AddRow(userID, "John Doe", "john@example.com", true, now, now))

		user, err := repo.FindByID(context.Background(), userID)
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, "John Doe", user.Name)
		assert.Equal(t, "john@example.com", user.Email)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByID with non-existing record", func(t *testing.T) {
		userID := 999

		mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		user, err := repo.FindByID(context.Background(), userID)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "not found")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestDeleteRecord tests the DeleteRecord operation
func TestDeleteRecord(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("DeleteRecord with existing record", func(t *testing.T) {
		user := &TestUser{
			ID:       1,
			Name:     "John Doe",
			Email:    "john@example.com",
			IsActive: true,
		}

		mock.ExpectExec(`DELETE FROM users WHERE id = \$1`).
			WithArgs(user.ID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		_, err := repo.DeleteRecord(context.Background(), user)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DeleteRecord with non-existing record", func(t *testing.T) {
		user := &TestUser{
			ID:       999,
			Name:     "Non Existent",
			Email:    "none@example.com",
			IsActive: false,
		}

		mock.ExpectExec(`DELETE FROM users WHERE id = \$1`).
			WithArgs(user.ID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		_, err := repo.DeleteRecord(context.Background(), user)
		assert.Error(t, err)
		assert.Equal(t, ErrNotFound, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestCreateMany tests the CreateMany operation
func TestCreateMany(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("CreateMany with multiple records", func(t *testing.T) {
		users := []TestUser{
			{Name: "User1", Email: "user1@example.com", IsActive: true},
			{Name: "User2", Email: "user2@example.com", IsActive: false},
		}

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO users`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		err := repo.CreateMany(context.Background(), users)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CreateMany with empty slice", func(t *testing.T) {
		users := []TestUser{}

		err := repo.CreateMany(context.Background(), users)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CreateMany with transaction error", func(t *testing.T) {
		users := []TestUser{
			{Name: "User1", Email: "user1@example.com", IsActive: true},
		}

		mock.ExpectBegin().WillReturnError(assert.AnError)

		err := repo.CreateMany(context.Background(), users)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to begin transaction")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestUpsert tests the Upsert operation
func TestUpsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("Upsert with update on conflict", func(t *testing.T) {
		user := &TestUser{
			Name:     "John Doe",
			Email:    "john@example.com",
			IsActive: true,
		}

		opts := UpsertOptions{
			ConflictColumns: []string{"email"},
			UpdateColumns:   []string{"name", "is_active"},
		}

		mock.ExpectExec(`INSERT INTO users .* ON CONFLICT \(email\) DO UPDATE SET`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Upsert(context.Background(), user, opts)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Upsert with do nothing on conflict", func(t *testing.T) {
		user := &TestUser{
			Name:     "Jane Doe",
			Email:    "jane@example.com",
			IsActive: false,
		}

		opts := UpsertOptions{
			ConflictColumns: []string{"email"},
		}

		mock.ExpectExec(`INSERT INTO users .* ON CONFLICT \(email\) DO UPDATE SET`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Upsert(context.Background(), user, opts)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestUpsertMany tests the UpsertMany operation
func TestUpsertMany(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("UpsertMany with multiple records", func(t *testing.T) {
		users := []TestUser{
			{Name: "User1", Email: "user1@example.com", IsActive: true},
			{Name: "User2", Email: "user2@example.com", IsActive: false},
		}

		opts := UpsertOptions{
			ConflictColumns: []string{"email"},
			UpdateColumns:   []string{"name", "is_active"},
		}

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO users .* ON CONFLICT \(email\) DO UPDATE SET`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		err := repo.UpsertMany(context.Background(), users, opts)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("UpsertMany with empty slice", func(t *testing.T) {
		users := []TestUser{}

		opts := UpsertOptions{
			ConflictColumns: []string{"email"},
		}

		err := repo.UpsertMany(context.Background(), users, opts)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestUpdateFields tests the UpdateFields operation
func TestUpdateFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("UpdateFields with valid ID", func(t *testing.T) {
		userID := 1
		now := time.Now()
		updates := map[string]interface{}{
			"name":      "Updated Name",
			"is_active": false,
		}

		mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}).
				AddRow(userID, "Old Name", "old@example.com", true, now, now))

		mock.ExpectExec(`UPDATE users SET`).
			WithArgs("Updated Name", false, userID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}).
				AddRow(userID, "Updated Name", "old@example.com", false, now, now))

		user, err := repo.UpdateFields(context.Background(), userID, updates)
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, "Updated Name", user.Name)
		assert.Equal(t, false, user.IsActive)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("UpdateFields with non-existing record", func(t *testing.T) {
		userID := 999
		updates := map[string]interface{}{
			"name": "Updated Name",
		}

		mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		user, err := repo.UpdateFields(context.Background(), userID, updates)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "not found")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("UpdateFields with empty updates", func(t *testing.T) {
		userID := 1
		updates := map[string]interface{}{}

		user, err := repo.UpdateFields(context.Background(), userID, updates)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "no updates provided")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("UpdateFields with update failure", func(t *testing.T) {
		userID := 1
		now := time.Now()
		updates := map[string]interface{}{
			"name": "Updated Name",
		}

		mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}).
				AddRow(userID, "Old Name", "old@example.com", true, now, now))

		mock.ExpectExec(`UPDATE users SET`).
			WithArgs("Updated Name", userID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		user, err := repo.UpdateFields(context.Background(), userID, updates)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, ErrNotFound, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestQueryUpdate tests the Query.Update operation
func TestQueryUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("Query Update with WHERE condition", func(t *testing.T) {

		mock.ExpectExec(`UPDATE users SET`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 2))

		nameCol := StringColumn{Column: Column[string]{Name: "name", Table: "users"}}
		isActiveCol := Column[bool]{Name: "is_active", Table: "users"}
		condition := nameCol.Like("test%")
		rowsAffected, err := repo.Query(context.Background()).Where(condition).Update(
			nameCol.Set("Updated Name"),
			isActiveCol.Set(false),
		)
		require.NoError(t, err)
		assert.Equal(t, int64(2), rowsAffected)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Query Update with no actions", func(t *testing.T) {

		rowsAffected, err := repo.Query(context.Background()).Update()
		assert.Error(t, err)
		assert.Equal(t, int64(0), rowsAffected)
		assert.Contains(t, err.Error(), "no actions provided")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Query Update without WHERE clause", func(t *testing.T) {

		mock.ExpectExec(`UPDATE users SET is_active = \$1`).
			WithArgs(false).
			WillReturnResult(sqlmock.NewResult(0, 10))

		isActiveCol := Column[bool]{Name: "is_active", Table: "users"}
		rowsAffected, err := repo.Query(context.Background()).Update(
			isActiveCol.Set(false),
		)
		require.NoError(t, err)
		assert.Equal(t, int64(10), rowsAffected)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Query Update with multiple conditions", func(t *testing.T) {

		mock.ExpectExec(`UPDATE users SET is_active = \$1, name = \$2 WHERE .*`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 3))

		activeCol := Column[bool]{Name: "is_active", Table: "users"}
		nameCol := StringColumn{Column: Column[string]{Name: "name", Table: "users"}}

		rowsAffected, err := repo.Query(context.Background()).
			Where(activeCol.Eq(true).And(nameCol.Like("test%"))).
			Update(
				activeCol.Set(false),
				nameCol.Set("Deactivated"),
			)
		require.NoError(t, err)
		assert.Equal(t, int64(3), rowsAffected)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Query Update with no matching records", func(t *testing.T) {

		mock.ExpectExec(`UPDATE users SET name = \$1 WHERE .*`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 0))

		idCol := Column[int]{Name: "id", Table: "users"}
		nameCol := StringColumn{Column: Column[string]{Name: "name", Table: "users"}}
		rowsAffected, err := repo.Query(context.Background()).
			Where(idCol.Eq(999)).
			Update(
				nameCol.Set("Updated"),
			)
		require.NoError(t, err)
		assert.Equal(t, int64(0), rowsAffected)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}
