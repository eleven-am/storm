package orm

import (
	"context"
	"fmt"
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMiddlewareIntegration tests that middleware can modify queries
func TestMiddlewareIntegration(t *testing.T) {

	t.Run("Middleware modifies SELECT query", func(t *testing.T) {
		called := false
		var capturedBuilder interface{}

		middleware := func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
			return func(ctx *MiddlewareContext) error {
				called = true
				capturedBuilder = ctx.QueryBuilder

				if sb, ok := ctx.QueryBuilder.(squirrel.SelectBuilder); ok {
					ctx.QueryBuilder = sb.Where(squirrel.Eq{"tenant_id": 123})
				}

				return next(ctx)
			}
		}

		finalFunc := func(ctx *MiddlewareContext) error {

			sb, ok := ctx.QueryBuilder.(squirrel.SelectBuilder)
			require.True(t, ok)

			sql, args, err := sb.ToSql()
			require.NoError(t, err)

			assert.Contains(t, sql, "tenant_id")
			assert.Contains(t, args, 123)

			return nil
		}

		mm := newMiddlewareManager()
		mm.AddMiddleware(middleware)

		query := squirrel.Select("*").From("users").Where(squirrel.Eq{"active": true})

		ctx := &MiddlewareContext{
			Operation:    OpQuery,
			TableName:    "users",
			QueryBuilder: query,
			Context:      context.Background(),
			Metadata:     make(map[string]interface{}),
		}

		err := mm.ExecuteMiddleware(ctx, finalFunc)
		require.NoError(t, err)

		assert.True(t, called)
		assert.NotNil(t, capturedBuilder)
	})

	t.Run("Middleware can block operations", func(t *testing.T) {
		middleware := func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
			return func(ctx *MiddlewareContext) error {
				if ctx.Operation == OpDelete {
					return fmt.Errorf("delete operations are not allowed")
				}
				return next(ctx)
			}
		}

		mm := newMiddlewareManager()
		mm.AddMiddleware(middleware)

		query := squirrel.Delete("users").Where(squirrel.Eq{"id": 1})

		ctx := &MiddlewareContext{
			Operation:    OpDelete,
			TableName:    "users",
			QueryBuilder: query,
			Context:      context.Background(),
			Metadata:     make(map[string]interface{}),
		}

		err := mm.ExecuteMiddleware(ctx, func(ctx *MiddlewareContext) error {
			t.Fatal("Final function should not be called")
			return nil
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "delete operations are not allowed")
	})

	t.Run("Multiple middleware execute in correct order", func(t *testing.T) {
		var order []int

		middleware1 := func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
			return func(ctx *MiddlewareContext) error {
				order = append(order, 1)
				err := next(ctx)
				order = append(order, 3)
				return err
			}
		}

		middleware2 := func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
			return func(ctx *MiddlewareContext) error {
				order = append(order, 2)
				return next(ctx)
			}
		}

		mm := newMiddlewareManager()
		mm.AddMiddleware(middleware1)
		mm.AddMiddleware(middleware2)

		ctx := &MiddlewareContext{
			Operation:    OpQuery,
			TableName:    "users",
			QueryBuilder: squirrel.Select("*").From("users"),
			Context:      context.Background(),
			Metadata:     make(map[string]interface{}),
		}

		err := mm.ExecuteMiddleware(ctx, func(ctx *MiddlewareContext) error {
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, order)
	})

	t.Run("Middleware can access and modify metadata", func(t *testing.T) {
		middleware := func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
			return func(ctx *MiddlewareContext) error {

				ctx.Metadata["user_id"] = "123"
				ctx.Metadata["request_id"] = "abc-123"

				return next(ctx)
			}
		}

		mm := newMiddlewareManager()
		mm.AddMiddleware(middleware)

		ctx := &MiddlewareContext{
			Operation:    OpQuery,
			TableName:    "users",
			QueryBuilder: squirrel.Select("*").From("users"),
			Context:      context.Background(),
			Metadata:     make(map[string]interface{}),
		}

		err := mm.ExecuteMiddleware(ctx, func(ctx *MiddlewareContext) error {

			assert.Equal(t, "123", ctx.Metadata["user_id"])
			assert.Equal(t, "abc-123", ctx.Metadata["request_id"])
			return nil
		})

		require.NoError(t, err)
	})

	t.Run("Middleware can modify different query types", func(t *testing.T) {
		testCases := []struct {
			name      string
			operation OperationType
			builder   interface{}
			modifier  func(interface{}) interface{}
		}{
			{
				name:      "SELECT query",
				operation: OpQuery,
				builder:   squirrel.Select("*").From("users"),
				modifier: func(b interface{}) interface{} {
					sb := b.(squirrel.SelectBuilder)
					return sb.Where(squirrel.Eq{"active": true})
				},
			},
			{
				name:      "INSERT query",
				operation: OpCreate,
				builder:   squirrel.Insert("users").Columns("name", "email"),
				modifier: func(b interface{}) interface{} {
					ib := b.(squirrel.InsertBuilder)
					return ib.Suffix("RETURNING id")
				},
			},
			{
				name:      "UPDATE query",
				operation: OpUpdate,
				builder:   squirrel.Update("users").Set("active", true),
				modifier: func(b interface{}) interface{} {
					ub := b.(squirrel.UpdateBuilder)
					return ub.Where(squirrel.Eq{"tenant_id": 123})
				},
			},
			{
				name:      "DELETE query",
				operation: OpDelete,
				builder:   squirrel.Delete("users"),
				modifier: func(b interface{}) interface{} {
					db := b.(squirrel.DeleteBuilder)
					return db.Where(squirrel.Eq{"tenant_id": 123})
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				middleware := func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
					return func(ctx *MiddlewareContext) error {
						ctx.QueryBuilder = tc.modifier(ctx.QueryBuilder)
						return next(ctx)
					}
				}

				mm := newMiddlewareManager()
				mm.AddMiddleware(middleware)

				ctx := &MiddlewareContext{
					Operation:    tc.operation,
					TableName:    "users",
					QueryBuilder: tc.builder,
					Context:      context.Background(),
					Metadata:     make(map[string]interface{}),
				}

				var modifiedBuilder interface{}
				err := mm.ExecuteMiddleware(ctx, func(ctx *MiddlewareContext) error {
					modifiedBuilder = ctx.QueryBuilder
					return nil
				})

				require.NoError(t, err)
				assert.NotNil(t, modifiedBuilder)

				assert.NotEqual(t, tc.builder, modifiedBuilder)
			})
		}
	})
}

// TestAuthorizationMiddleware demonstrates a real-world authorization middleware
func TestAuthorizationMiddleware(t *testing.T) {

	createAuthMiddleware := func(userID int) QueryMiddleware {
		return func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
			return func(ctx *MiddlewareContext) error {
				switch ctx.Operation {
				case OpQuery:
					if sb, ok := ctx.QueryBuilder.(squirrel.SelectBuilder); ok {

						ctx.QueryBuilder = sb.Where(squirrel.Or{
							squirrel.Eq{"user_id": userID},
							squirrel.Eq{"is_public": true},
						})
					}
				case OpUpdate, OpUpdateMany:
					if ub, ok := ctx.QueryBuilder.(squirrel.UpdateBuilder); ok {

						ctx.QueryBuilder = ub.Where(squirrel.Eq{"user_id": userID})
					}
				case OpDelete:
					if db, ok := ctx.QueryBuilder.(squirrel.DeleteBuilder); ok {

						ctx.QueryBuilder = db.Where(squirrel.Eq{"user_id": userID})
					}
				case OpCreate:

				}

				return next(ctx)
			}
		}
	}

	t.Run("Auth middleware filters SELECT queries", func(t *testing.T) {
		mm := newMiddlewareManager()
		mm.AddMiddleware(createAuthMiddleware(123))

		query := squirrel.Select("*").From("posts").Where(squirrel.Eq{"status": "published"})

		ctx := &MiddlewareContext{
			Operation:    OpQuery,
			TableName:    "posts",
			QueryBuilder: query,
			Context:      context.Background(),
			Metadata:     make(map[string]interface{}),
		}

		err := mm.ExecuteMiddleware(ctx, func(ctx *MiddlewareContext) error {
			sb := ctx.QueryBuilder.(squirrel.SelectBuilder)
			sql, args, err := sb.ToSql()
			require.NoError(t, err)

			assert.Contains(t, sql, "user_id")
			assert.Contains(t, sql, "is_public")
			assert.Contains(t, args, 123)
			assert.Contains(t, args, true)

			return nil
		})

		require.NoError(t, err)
	})

	t.Run("Auth middleware restricts UPDATE queries", func(t *testing.T) {
		mm := newMiddlewareManager()
		mm.AddMiddleware(createAuthMiddleware(456))

		query := squirrel.Update("posts").Set("title", "New Title").Where(squirrel.Eq{"id": 1})

		ctx := &MiddlewareContext{
			Operation:    OpUpdate,
			TableName:    "posts",
			QueryBuilder: query,
			Context:      context.Background(),
			Metadata:     make(map[string]interface{}),
		}

		err := mm.ExecuteMiddleware(ctx, func(ctx *MiddlewareContext) error {
			ub := ctx.QueryBuilder.(squirrel.UpdateBuilder)
			sql, args, err := ub.ToSql()
			require.NoError(t, err)

			assert.Contains(t, sql, "user_id")
			assert.Contains(t, args, 456)

			return nil
		})

		require.NoError(t, err)
	})
}

// TestMultiTenancyMiddleware demonstrates multi-tenancy implementation
func TestMultiTenancyMiddleware(t *testing.T) {
	createTenantMiddleware := func(tenantID int) QueryMiddleware {
		return func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
			return func(ctx *MiddlewareContext) error {

				switch v := ctx.QueryBuilder.(type) {
				case squirrel.SelectBuilder:
					ctx.QueryBuilder = v.Where(squirrel.Eq{"tenant_id": tenantID})
				case squirrel.UpdateBuilder:
					ctx.QueryBuilder = v.Where(squirrel.Eq{"tenant_id": tenantID})
				case squirrel.DeleteBuilder:
					ctx.QueryBuilder = v.Where(squirrel.Eq{"tenant_id": tenantID})
				case squirrel.InsertBuilder:

				}

				return next(ctx)
			}
		}
	}

	mm := newMiddlewareManager()
	mm.AddMiddleware(createTenantMiddleware(999))

	query := squirrel.Select("*").From("products").Where(squirrel.Eq{"active": true})

	ctx := &MiddlewareContext{
		Operation:    OpQuery,
		TableName:    "products",
		QueryBuilder: query,
		Context:      context.Background(),
		Metadata:     make(map[string]interface{}),
	}

	err := mm.ExecuteMiddleware(ctx, func(ctx *MiddlewareContext) error {
		sb := ctx.QueryBuilder.(squirrel.SelectBuilder)
		sql, _, err := sb.ToSql()
		require.NoError(t, err)

		assert.Contains(t, sql, "tenant_id")

		return nil
	})

	require.NoError(t, err)
}
