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

// Test models for ScanToModel functionality
type ScanTestUser struct {
	ID      int64  `db:"id"`
	Name    string `db:"name"`
	Email   string `db:"email"`
	Profile *ScanTestProfile
	Posts   []ScanTestPost
}

type ScanTestProfile struct {
	ID     int64  `db:"id"`
	UserID int64  `db:"user_id"`
	Bio    string `db:"bio"`
	User   *ScanTestUser
}

type ScanTestPost struct {
	ID      int64  `db:"id"`
	UserID  int64  `db:"user_id"`
	Title   string `db:"title"`
	Content string `db:"content"`
	User    *ScanTestUser
}

func TestScanToModel_BelongsTo_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	ctx := context.Background()

	scanToModel := func(ctx context.Context, exec DBExecutor, query string, args []interface{}, model interface{}) error {
		var user ScanTestUser
		err := exec.GetContext(ctx, &user, query, args...)
		if err != nil {
			return err
		}
		model.(*ScanTestProfile).User = &user
		return nil
	}

	profile := &ScanTestProfile{
		ID:     1,
		UserID: 100,
		Bio:    "Software Engineer",
	}

	userRows := sqlmock.NewRows([]string{"id", "name", "email"}).
		AddRow(100, "John Doe", "john@example.com")

	mock.ExpectQuery("SELECT (.+) FROM users WHERE id = ?").
		WithArgs(100).
		WillReturnRows(userRows)

	err = scanToModel(ctx, sqlxDB, "SELECT id, name, email FROM users WHERE id = ?", []interface{}{100}, profile)
	require.NoError(t, err)

	require.NotNil(t, profile.User)
	assert.Equal(t, int64(100), profile.User.ID)
	assert.Equal(t, "John Doe", profile.User.Name)
	assert.Equal(t, "john@example.com", profile.User.Email)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScanToModel_HasOne_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	ctx := context.Background()

	scanToModel := func(ctx context.Context, exec DBExecutor, query string, args []interface{}, model interface{}) error {
		var profile ScanTestProfile
		err := exec.GetContext(ctx, &profile, query, args...)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}
		model.(*ScanTestUser).Profile = &profile
		return nil
	}

	user := &ScanTestUser{
		ID:    100,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	profileRows := sqlmock.NewRows([]string{"id", "user_id", "bio"}).
		AddRow(1, 100, "Software Engineer")

	mock.ExpectQuery("SELECT (.+) FROM profiles WHERE user_id = ?").
		WithArgs(100).
		WillReturnRows(profileRows)

	err = scanToModel(ctx, sqlxDB, "SELECT id, user_id, bio FROM profiles WHERE user_id = ?", []interface{}{100}, user)
	require.NoError(t, err)

	require.NotNil(t, user.Profile)
	assert.Equal(t, int64(1), user.Profile.ID)
	assert.Equal(t, int64(100), user.Profile.UserID)
	assert.Equal(t, "Software Engineer", user.Profile.Bio)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScanToModel_HasOne_NoRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	ctx := context.Background()

	scanToModel := func(ctx context.Context, exec DBExecutor, query string, args []interface{}, model interface{}) error {
		var profile ScanTestProfile
		err := exec.GetContext(ctx, &profile, query, args...)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}
		model.(*ScanTestUser).Profile = &profile
		return nil
	}

	user := &ScanTestUser{
		ID:    100,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	mock.ExpectQuery("SELECT (.+) FROM profiles WHERE user_id = ?").
		WithArgs(100).
		WillReturnError(sql.ErrNoRows)

	err = scanToModel(ctx, sqlxDB, "SELECT id, user_id, bio FROM profiles WHERE user_id = ?", []interface{}{100}, user)
	require.NoError(t, err)

	assert.Nil(t, user.Profile)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScanToModel_HasMany_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	ctx := context.Background()

	scanToModel := func(ctx context.Context, exec DBExecutor, query string, args []interface{}, model interface{}) error {
		var posts []ScanTestPost
		err := exec.SelectContext(ctx, &posts, query, args...)
		if err != nil {
			return err
		}
		model.(*ScanTestUser).Posts = posts
		return nil
	}

	user := &ScanTestUser{
		ID:    100,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	postRows := sqlmock.NewRows([]string{"id", "user_id", "title", "content"}).
		AddRow(1, 100, "First Post", "This is my first post").
		AddRow(2, 100, "Second Post", "This is my second post")

	mock.ExpectQuery("SELECT (.+) FROM posts WHERE user_id = ?").
		WithArgs(100).
		WillReturnRows(postRows)

	err = scanToModel(ctx, sqlxDB, "SELECT id, user_id, title, content FROM posts WHERE user_id = ?", []interface{}{100}, user)
	require.NoError(t, err)

	require.Len(t, user.Posts, 2)

	assert.Equal(t, int64(1), user.Posts[0].ID)
	assert.Equal(t, int64(100), user.Posts[0].UserID)
	assert.Equal(t, "First Post", user.Posts[0].Title)
	assert.Equal(t, "This is my first post", user.Posts[0].Content)

	assert.Equal(t, int64(2), user.Posts[1].ID)
	assert.Equal(t, int64(100), user.Posts[1].UserID)
	assert.Equal(t, "Second Post", user.Posts[1].Title)
	assert.Equal(t, "This is my second post", user.Posts[1].Content)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScanToModel_HasMany_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	ctx := context.Background()

	scanToModel := func(ctx context.Context, exec DBExecutor, query string, args []interface{}, model interface{}) error {
		var posts []ScanTestPost
		err := exec.SelectContext(ctx, &posts, query, args...)
		if err != nil {
			return err
		}
		model.(*ScanTestUser).Posts = posts
		return nil
	}

	user := &ScanTestUser{
		ID:    100,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	emptyRows := sqlmock.NewRows([]string{"id", "user_id", "title", "content"})
	mock.ExpectQuery("SELECT (.+) FROM posts WHERE user_id = ?").
		WithArgs(100).
		WillReturnRows(emptyRows)

	err = scanToModel(ctx, sqlxDB, "SELECT id, user_id, title, content FROM posts WHERE user_id = ?", []interface{}{100}, user)
	require.NoError(t, err)

	assert.Empty(t, user.Posts)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScanToModel_Error_Handling(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	ctx := context.Background()

	scanToModel := func(ctx context.Context, exec DBExecutor, query string, args []interface{}, model interface{}) error {
		var user ScanTestUser
		err := exec.GetContext(ctx, &user, query, args...)
		if err != nil {
			return err
		}
		model.(*ScanTestProfile).User = &user
		return nil
	}

	profile := &ScanTestProfile{
		ID:     1,
		UserID: 100,
		Bio:    "Software Engineer",
	}

	mock.ExpectQuery("SELECT (.+) FROM users WHERE id = ?").
		WithArgs(100).
		WillReturnError(sql.ErrConnDone)

	err = scanToModel(ctx, sqlxDB, "SELECT id, name, email FROM users WHERE id = ?", []interface{}{100}, profile)
	require.Error(t, err)
	assert.Equal(t, sql.ErrConnDone, err)

	assert.Nil(t, profile.User)

	assert.NoError(t, mock.ExpectationsWereMet())
}
