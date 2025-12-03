package orm

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryGetMiddlewareManager(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("getMiddlewareManager creates new manager", func(t *testing.T) {

		mgr := repo.getMiddlewareManager()
		assert.NotNil(t, mgr)

		mgr2 := repo.getMiddlewareManager()
		assert.Equal(t, mgr, mgr2)
	})
}
