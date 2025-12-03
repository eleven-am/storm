package orm

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test model for authorization tests
type AuthTestUser struct {
	ID     string `db:"id"`
	Email  string `db:"email"`
	TeamID string `db:"team_id"`
	Role   string `db:"role"`
}

// Mock user context for testing
type mockUserContext struct {
	UserID string
	TeamID string
	Role   string
}

func createTestRepository(t testing.TB) *Repository[AuthTestUser] {

	metadata := &ModelMetadata{
		TableName:   "auth_test_users",
		PrimaryKeys: []string{"id"},
		Columns: map[string]*ColumnMetadata{
			"ID":     {DBName: "id", FieldName: "ID"},
			"Email":  {DBName: "email", FieldName: "Email"},
			"TeamID": {DBName: "team_id", FieldName: "TeamID"},
			"Role":   {DBName: "role", FieldName: "Role"},
		},
	}

	mockDB := &sqlx.DB{}
	repo, err := NewRepositoryWithExecutor[AuthTestUser](mockDB, metadata)
	require.NoError(t, err)
	return repo
}

func TestAuthorize_SingleFunction(t *testing.T) {
	baseRepo := createTestRepository(t)

	assert.Empty(t, baseRepo.authorizeFuncs)

	authFunc := func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
		return query
	}

	authRepo := baseRepo.Authorize(authFunc)

	assert.Len(t, authRepo.authorizeFuncs, 1)

	assert.Empty(t, baseRepo.authorizeFuncs)

	assert.NotSame(t, baseRepo, authRepo)
}

func TestAuthorize_MultipleFunction(t *testing.T) {
	baseRepo := createTestRepository(t)

	authRepo := baseRepo.
		Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
			return query
		}).
		Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
			return query
		}).
		Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
			return query
		})

	assert.Len(t, authRepo.authorizeFuncs, 3)

	assert.Empty(t, baseRepo.authorizeFuncs)

	assert.NotSame(t, baseRepo, authRepo)
}

func TestAuthorize_ImmutableChaining(t *testing.T) {
	baseRepo := createTestRepository(t)

	authRepo1 := baseRepo.Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
		return query
	})

	authRepo2 := authRepo1.Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
		return query
	})

	authRepo3 := baseRepo.Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
		return query
	})

	assert.Len(t, baseRepo.authorizeFuncs, 0)
	assert.Len(t, authRepo1.authorizeFuncs, 1)
	assert.Len(t, authRepo2.authorizeFuncs, 2)
	assert.Len(t, authRepo3.authorizeFuncs, 1)

	assert.NotSame(t, baseRepo, authRepo1)
	assert.NotSame(t, authRepo1, authRepo2)
	assert.NotSame(t, authRepo1, authRepo3)
	assert.NotSame(t, authRepo2, authRepo3)
}

func TestQuery_NoAuthorization(t *testing.T) {
	baseRepo := createTestRepository(t)
	ctx := context.Background()

	query := baseRepo.Query(ctx)

	assert.NotNil(t, query)
	assert.Equal(t, baseRepo, query.repo)
	assert.Equal(t, ctx, query.ctx)
}

func TestQuery_WithAuthorization(t *testing.T) {
	baseRepo := createTestRepository(t)
	ctx := context.Background()

	userCtx := mockUserContext{
		UserID: "user123",
		TeamID: "team456",
		Role:   "member",
	}
	ctx = context.WithValue(ctx, "user", userCtx)

	var authCallCount int
	var authContexts []context.Context

	authRepo := baseRepo.
		Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
			authCallCount++
			authContexts = append(authContexts, ctx)

			user, ok := ctx.Value("user").(mockUserContext)
			assert.True(t, ok)
			assert.Equal(t, "user123", user.UserID)
			assert.Equal(t, "team456", user.TeamID)

			return query
		}).
		Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
			authCallCount++
			authContexts = append(authContexts, ctx)
			return query
		})

	query := authRepo.Query(ctx)

	assert.NotNil(t, query)

	assert.Equal(t, 2, authCallCount)
	assert.Len(t, authContexts, 2)

	for _, authCtx := range authContexts {
		user, ok := authCtx.Value("user").(mockUserContext)
		assert.True(t, ok)
		assert.Equal(t, "user123", user.UserID)
	}
}

func TestQuery_AuthorizationOrder(t *testing.T) {
	baseRepo := createTestRepository(t)
	ctx := context.Background()

	var callOrder []string

	authRepo := baseRepo.
		Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
			callOrder = append(callOrder, "first")
			return query
		}).
		Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
			callOrder = append(callOrder, "second")
			return query
		}).
		Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
			callOrder = append(callOrder, "third")
			return query
		})

	query := authRepo.Query(ctx)

	assert.NotNil(t, query)

	assert.Equal(t, []string{"first", "second", "third"}, callOrder)
}

func TestQuery_AuthorizationModifiesQuery(t *testing.T) {
	baseRepo := createTestRepository(t)
	ctx := context.Background()

	userCtx := mockUserContext{
		UserID: "user123",
		TeamID: "team456",
		Role:   "member",
	}
	ctx = context.WithValue(ctx, "user", userCtx)

	var queryModified bool

	authRepo := baseRepo.Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {

		queryModified = true

		assert.NotNil(t, query)
		assert.Equal(t, "auth_test_users", query.repo.metadata.TableName)

		return query
	})

	query := authRepo.Query(ctx)

	assert.True(t, queryModified)
	assert.NotNil(t, query)
}

func TestQuery_AuthorizationWithRoleBasedLogic(t *testing.T) {
	baseRepo := createTestRepository(t)
	ctx := context.Background()

	testCases := []struct {
		name     string
		role     string
		expected string
	}{
		{
			name:     "Admin role",
			role:     "admin",
			expected: "admin_filter",
		},
		{
			name:     "Member role",
			role:     "member",
			expected: "member_filter",
		},
		{
			name:     "Guest role",
			role:     "guest",
			expected: "guest_filter",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			userCtx := mockUserContext{
				UserID: "user123",
				TeamID: "team456",
				Role:   tc.role,
			}
			testCtx := context.WithValue(ctx, "user", userCtx)

			var appliedFilter string

			authRepo := baseRepo.Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
				user, ok := ctx.Value("user").(mockUserContext)
				assert.True(t, ok)

				switch user.Role {
				case "admin":
					appliedFilter = "admin_filter"
				case "member":
					appliedFilter = "member_filter"
				case "guest":
					appliedFilter = "guest_filter"
				default:
					appliedFilter = "unknown_filter"
				}

				return query
			})

			query := authRepo.Query(testCtx)

			assert.NotNil(t, query)
			assert.Equal(t, tc.expected, appliedFilter)
		})
	}
}

func TestAuthorize_NilFunction(t *testing.T) {
	baseRepo := createTestRepository(t)

	authRepo := baseRepo.Authorize(nil)

	assert.Len(t, authRepo.authorizeFuncs, 1)
	assert.Nil(t, authRepo.authorizeFuncs[0])
}

func TestQuery_WithNilAuthorizationFunction(t *testing.T) {
	baseRepo := createTestRepository(t)
	ctx := context.Background()

	authRepo := baseRepo.Authorize(nil)

	assert.Panics(t, func() {
		authRepo.Query(ctx)
	})
}

// Benchmark authorization overhead
func BenchmarkQuery_NoAuthorization(b *testing.B) {
	baseRepo := createTestRepository(b)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := baseRepo.Query(ctx)
		_ = query
	}
}

func BenchmarkQuery_SingleAuthorization(b *testing.B) {
	baseRepo := createTestRepository(b)
	ctx := context.Background()

	authRepo := baseRepo.Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
		return query
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := authRepo.Query(ctx)
		_ = query
	}
}

func BenchmarkQuery_MultipleAuthorization(b *testing.B) {
	baseRepo := createTestRepository(b)
	ctx := context.Background()

	authRepo := baseRepo.
		Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
			return query
		}).
		Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
			return query
		}).
		Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
			return query
		})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := authRepo.Query(ctx)
		_ = query
	}
}

func BenchmarkAuthorize_ChainCreation(b *testing.B) {
	baseRepo := createTestRepository(b)

	authFunc := func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
		return query
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		authRepo := baseRepo.Authorize(authFunc)
		_ = authRepo
	}
}
