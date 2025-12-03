package orm

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryJoinRelationship(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("JoinRelationship", func(t *testing.T) {
		query := repo.Query(context.Background())
		result := query.JoinRelationship("posts", InnerJoin)
		assert.NotNil(t, result)
		assert.Equal(t, query, result)
	})

	t.Run("RawJoin", func(t *testing.T) {
		query := repo.Query(context.Background())
		result := query.RawJoin("CROSS JOIN posts")
		assert.NotNil(t, result)
		assert.Equal(t, query, result)
	})
}
