package orm

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryIncludeWhere(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("IncludeWhere with condition", func(t *testing.T) {
		query := repo.Query(context.Background())
		nameCol := Column[string]{Name: "name", Table: "posts"}
		result := query.IncludeWhere("posts", nameCol.Eq("Test Post"))
		assert.NotNil(t, result)
		assert.Equal(t, query, result)

	})

	t.Run("IncludeWhere without condition", func(t *testing.T) {
		query := repo.Query(context.Background())
		result := query.IncludeWhere("posts")
		assert.NotNil(t, result)
		assert.Equal(t, query, result)

	})
}
