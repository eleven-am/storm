package orm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoinBuilder(t *testing.T) {
	t.Run("newJoinBuilder", func(t *testing.T) {
		joinBuilder := newJoinBuilder()
		assert.NotNil(t, joinBuilder)
		assert.NotNil(t, joinBuilder.joins)
		assert.Empty(t, joinBuilder.joins)
	})

	t.Run("Inner Join", func(t *testing.T) {
		joinBuilder := newJoinBuilder()

		result := joinBuilder.Inner("posts", "posts.user_id = users.id")
		assert.Equal(t, joinBuilder, result)
		assert.Len(t, joinBuilder.joins, 1)
		assert.Equal(t, InnerJoin, joinBuilder.joins[0].Type)
		assert.Equal(t, "posts", joinBuilder.joins[0].Table)
		assert.Equal(t, "posts.user_id = users.id", joinBuilder.joins[0].Condition)
	})

	t.Run("Inner Join As", func(t *testing.T) {
		joinBuilder := newJoinBuilder()

		result := joinBuilder.InnerAs("posts", "p", "p.user_id = users.id")
		assert.Equal(t, joinBuilder, result)
		assert.Len(t, joinBuilder.joins, 1)
		assert.Equal(t, InnerJoin, joinBuilder.joins[0].Type)
		assert.Equal(t, "posts", joinBuilder.joins[0].Table)
		assert.Equal(t, "p", joinBuilder.joins[0].Alias)
		assert.Equal(t, "p.user_id = users.id", joinBuilder.joins[0].Condition)
	})

	t.Run("Left Join", func(t *testing.T) {
		joinBuilder := newJoinBuilder()

		result := joinBuilder.Left("posts", "posts.user_id = users.id")
		assert.Equal(t, joinBuilder, result)
		assert.Len(t, joinBuilder.joins, 1)
		assert.Equal(t, LeftJoin, joinBuilder.joins[0].Type)
	})

	t.Run("Left Join As", func(t *testing.T) {
		joinBuilder := newJoinBuilder()

		result := joinBuilder.LeftAs("posts", "p", "p.user_id = users.id")
		assert.Equal(t, joinBuilder, result)
		assert.Len(t, joinBuilder.joins, 1)
		assert.Equal(t, LeftJoin, joinBuilder.joins[0].Type)
		assert.Equal(t, "p", joinBuilder.joins[0].Alias)
	})

	t.Run("Right Join", func(t *testing.T) {
		joinBuilder := newJoinBuilder()

		result := joinBuilder.Right("posts", "posts.user_id = users.id")
		assert.Equal(t, joinBuilder, result)
		assert.Len(t, joinBuilder.joins, 1)
		assert.Equal(t, RightJoin, joinBuilder.joins[0].Type)
	})

	t.Run("Right Join As", func(t *testing.T) {
		joinBuilder := newJoinBuilder()

		result := joinBuilder.RightAs("posts", "p", "p.user_id = users.id")
		assert.Equal(t, joinBuilder, result)
		assert.Len(t, joinBuilder.joins, 1)
		assert.Equal(t, RightJoin, joinBuilder.joins[0].Type)
		assert.Equal(t, "p", joinBuilder.joins[0].Alias)
	})

	t.Run("Full Join", func(t *testing.T) {
		joinBuilder := newJoinBuilder()

		result := joinBuilder.Full("posts", "posts.user_id = users.id")
		assert.Equal(t, joinBuilder, result)
		assert.Len(t, joinBuilder.joins, 1)
		assert.Equal(t, FullJoin, joinBuilder.joins[0].Type)
	})

	t.Run("Full Join As", func(t *testing.T) {
		joinBuilder := newJoinBuilder()

		result := joinBuilder.FullAs("posts", "p", "p.user_id = users.id")
		assert.Equal(t, joinBuilder, result)
		assert.Len(t, joinBuilder.joins, 1)
		assert.Equal(t, FullJoin, joinBuilder.joins[0].Type)
		assert.Equal(t, "p", joinBuilder.joins[0].Alias)
	})

	t.Run("Build with no joins", func(t *testing.T) {
		joinBuilder := newJoinBuilder()

		result := joinBuilder.Build()
		assert.Empty(t, result)
	})

	t.Run("Build with multiple joins", func(t *testing.T) {
		joinBuilder := newJoinBuilder()

		joinBuilder.Inner("posts", "posts.user_id = users.id")
		joinBuilder.Left("comments", "comments.post_id = posts.id")

		result := joinBuilder.Build()
		assert.Len(t, result, 2)
		assert.Equal(t, InnerJoin, result[0].Type)
		assert.Equal(t, LeftJoin, result[1].Type)
	})
}
