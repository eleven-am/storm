package orm

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStormInitializeRepositories(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	storm := NewStorm(sqlxDB)

	t.Run("initializeRepositories", func(t *testing.T) {

		userMeta := &ModelMetadata{
			TableName:   "users",
			StructName:  "TestUser",
			PrimaryKeys: []string{"id"},
			Columns: map[string]*ColumnMetadata{
				"ID": &ColumnMetadata{
					FieldName:    "ID",
					DBName:       "id",
					DBType:       "bigint",
					GoType:       "int64",
					IsPrimaryKey: true,
				},
			},
		}

		postMeta := &ModelMetadata{
			TableName:   "posts",
			StructName:  "TestPost",
			PrimaryKeys: []string{"id"},
			Columns: map[string]*ColumnMetadata{
				"ID": &ColumnMetadata{
					FieldName:    "ID",
					DBName:       "id",
					DBType:       "bigint",
					GoType:       "int64",
					IsPrimaryKey: true,
				},
			},
		}

		userRepo, err := NewRepository[TestUser](sqlxDB, userMeta)
		require.NoError(t, err)

		postRepo, err := NewRepository[TestPost](sqlxDB, postMeta)
		require.NoError(t, err)

		storm.repositories["users"] = userRepo
		storm.repositories["posts"] = postRepo

		storm.initializeRepositories()

		assert.Len(t, storm.repositories, 2)
		assert.Contains(t, storm.repositories, "users")
		assert.Contains(t, storm.repositories, "posts")
	})
}

type TestPost struct {
	ID      int64  `storm:"id,primarykey"`
	Title   string `storm:"title"`
	Content string `storm:"content"`
}
