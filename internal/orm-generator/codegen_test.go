package orm_generator

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Test model for code generation
type TestUser struct {
	ID        int       `db:"id" dbdef:"primary_key;auto_increment"`
	Name      string    `db:"name" dbdef:"not_null"`
	Email     string    `db:"email" dbdef:"unique;not_null"`
	Age       int       `db:"age"`
	IsActive  bool      `db:"is_active" dbdef:"default:true"`
	CreatedAt time.Time `db:"created_at" dbdef:"default:now()"`
	UpdatedAt time.Time `db:"updated_at" dbdef:"default:now()"`

	// Relationships
	Posts   []TestPost   `db:"-" orm:"has_many:TestPost;foreign_key:user_id"`
	Profile *TestProfile `db:"-" orm:"has_one:TestProfile;foreign_key:user_id"`
}

type TestPost struct {
	ID      int    `db:"id" dbdef:"primary_key;auto_increment"`
	Title   string `db:"title" dbdef:"not_null"`
	Content string `db:"content"`
	UserID  int    `db:"user_id" dbdef:"not_null"`

	// Relationships
	User *TestUser `db:"-" orm:"belongs_to:TestUser;foreign_key:user_id"`
}

type TestProfile struct {
	ID     int    `db:"id" dbdef:"primary_key;auto_increment"`
	Bio    string `db:"bio"`
	UserID int    `db:"user_id" dbdef:"unique;not_null"`

	// Relationships
	User *TestUser `db:"-" orm:"belongs_to:TestUser;foreign_key:user_id"`
}

func TestCodeGeneration(t *testing.T) {
	tmpDir := os.TempDir()
	outputDir := filepath.Join(tmpDir, "orm_test_output")
	modelDir := filepath.Join(tmpDir, "test_models")
	defer os.RemoveAll(tmpDir)

	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatalf("Failed to create model directory: %v", err)
	}

	testModelCode := `package models

import "time"

type TestUser struct {
	_ struct{} ` + "`" + `dbdef:"table:test_users"` + "`" + `
	
	ID        int       ` + "`" + `db:"id" dbdef:"type:integer;primary_key"` + "`" + `
	Name      string    ` + "`" + `db:"name" dbdef:"type:varchar(100);not_null"` + "`" + `
	Email     string    ` + "`" + `db:"email" dbdef:"type:varchar(255);unique;not_null"` + "`" + `
	Age       int       ` + "`" + `db:"age" dbdef:"type:integer"` + "`" + `
	IsActive  bool      ` + "`" + `db:"is_active" dbdef:"type:boolean;default:true"` + "`" + `
	CreatedAt time.Time ` + "`" + `db:"created_at" dbdef:"type:timestamptz;default:now()"` + "`" + `
	UpdatedAt time.Time ` + "`" + `db:"updated_at" dbdef:"type:timestamptz;default:now()"` + "`" + `
	
	Posts []TestPost ` + "`" + `db:"-" orm:"has_many:TestPost,foreign_key:user_id"` + "`" + `
	Profile *TestProfile ` + "`" + `db:"-" orm:"has_one:TestProfile,foreign_key:user_id"` + "`" + `
}

type TestPost struct {
	_ struct{} ` + "`" + `dbdef:"table:test_posts"` + "`" + `
	
	ID       int    ` + "`" + `db:"id" dbdef:"type:integer;primary_key"` + "`" + `
	Title    string ` + "`" + `db:"title" dbdef:"type:varchar(255);not_null"` + "`" + `
	Content  string ` + "`" + `db:"content" dbdef:"type:text"` + "`" + `
	UserID   int    ` + "`" + `db:"user_id" dbdef:"type:integer;not_null"` + "`" + `
	
	User *TestUser ` + "`" + `db:"-" orm:"belongs_to:TestUser,foreign_key:user_id"` + "`" + `
}

type TestProfile struct {
	_ struct{} ` + "`" + `dbdef:"table:test_profiles"` + "`" + `
	
	ID     int    ` + "`" + `db:"id" dbdef:"type:integer;primary_key"` + "`" + `
	Bio    string ` + "`" + `db:"bio" dbdef:"type:text"` + "`" + `
	UserID int    ` + "`" + `db:"user_id" dbdef:"type:integer;unique;not_null"` + "`" + `
	
	User *TestUser ` + "`" + `db:"-" orm:"belongs_to:TestUser,foreign_key:user_id"` + "`" + `
}
`
	if err := os.WriteFile(filepath.Join(modelDir, "models.go"), []byte(testModelCode), 0644); err != nil {
		t.Fatalf("Failed to write test models: %v", err)
	}

	config := GenerationConfig{
		PackageName: "models",
		OutputDir:   outputDir,
	}

	generator := NewCodeGenerator(config)

	err := generator.DiscoverModels(modelDir)
	if err != nil {
		t.Fatalf("Failed to discover models: %v", err)
	}

	err = generator.ValidateModels()
	if err != nil {
		t.Fatalf("Model validation failed: %v", err)
	}

	err = generator.GenerateAll()
	if err != nil {
		t.Fatalf("Code generation failed: %v", err)
	}

	expectedFiles := []string{
		"columns.go",
		"test_user_repository.go",
		"test_post_repository.go",
		"test_profile_repository.go",
		"storm.go",
	}

	for _, filename := range expectedFiles {
		filepath := filepath.Join(outputDir, filename)
		if _, err := os.Stat(filepath); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", filename)
		}
	}

	columnsFile := filepath.Join(outputDir, "columns.go")
	content, err := os.ReadFile(columnsFile)
	if err != nil {
		t.Fatalf("Failed to read columns.go: %v", err)
	}

	expectedContent := []string{
		"TestUsers",
		"TestPosts",
		"TestProfiles",
		"storm.StringColumn",
		"storm.NumericColumn",
		"storm.BoolColumn",
		"storm.TimeColumn",
	}

	for _, expected := range expectedContent {
		if !containsString(string(content), expected) {
			t.Errorf("Generated columns.go missing expected content: %s", expected)
		}
	}

	userRepoFile := filepath.Join(outputDir, "test_user_repository.go")
	repoContent, err := os.ReadFile(userRepoFile)
	if err != nil {
		t.Fatalf("Failed to read test_user_repository.go: %v", err)
	}

	expectedRepoContent := []string{
		"func (r *TestUserRepository) Authorize(",
		"func(ctx context.Context, query *TestUserQuery) *TestUserQuery",
		"genericFn := func(ctx context.Context, query *storm.Query[TestUser]) *storm.Query[TestUser]",
		"testuserQuery := &TestUserQuery{",
		"baseRepo := r.Repository.Authorize(genericFn)",
		"return &TestUserRepository{",
	}

	for _, expected := range expectedRepoContent {
		if !containsString(string(repoContent), expected) {
			t.Errorf("Generated test_user_repository.go missing expected Authorize method content: %s", expected)
		}
	}

	expectedIncludeContent := []string{
		"func (q *TestUserQuery) IncludePosts() *TestUserQuery {",
		"q.Query = q.Query.Include(\"Posts\")",
		"func (q *TestUserQuery) IncludeProfile() *TestUserQuery {",
		"q.Query = q.Query.Include(\"Profile\")",
	}

	for _, expected := range expectedIncludeContent {
		if !containsString(string(repoContent), expected) {
			t.Errorf("Generated test_user_repository.go missing expected IncludeXXX method content: %s", expected)
		}
	}

	unexpectedWithContent := []string{
		"func (r *TestUserRepository) WithPosts(",
		"func (r *TestUserRepository) WithProfile(",
	}

	for _, unexpected := range unexpectedWithContent {
		if containsString(string(repoContent), unexpected) {
			t.Errorf("Generated test_user_repository.go contains unexpected WithXXX method that should be removed: %s", unexpected)
		}
	}

	t.Logf("Code generation test passed! Files created in: %s", outputDir)
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr[:len(substr)] ||
		len(s) > len(substr) && s[:len(substr)] == substr ||
		len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestModelDiscovery(t *testing.T) {
	tmpDir := os.TempDir()
	modelDir := filepath.Join(tmpDir, "test_models_discovery")
	defer os.RemoveAll(modelDir)

	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatalf("Failed to create model directory: %v", err)
	}

	testModelCode := `package test

import "time"

type TestUser struct {
	_ struct{} ` + "`" + `dbdef:"table:test_users"` + "`" + `
	
	ID        int       ` + "`" + `db:"id" dbdef:"type:integer;primary_key"` + "`" + `
	Name      string    ` + "`" + `db:"name" dbdef:"type:varchar(100);not_null"` + "`" + `
	Email     string    ` + "`" + `db:"email" dbdef:"type:varchar(255);unique;not_null"` + "`" + `
	CreatedAt time.Time ` + "`" + `db:"created_at" dbdef:"type:timestamptz;default:now()"` + "`" + `
}
`
	if err := os.WriteFile(filepath.Join(modelDir, "user.go"), []byte(testModelCode), 0644); err != nil {
		t.Fatalf("Failed to write test model: %v", err)
	}

	config := GenerationConfig{
		PackageName: "",
		OutputDir:   "/tmp/test",
	}

	generator := NewCodeGenerator(config)

	err := generator.DiscoverModels(modelDir)
	if err != nil {
		t.Fatalf("Failed to discover models: %v", err)
	}

	models := generator.GetModelNames()
	if len(models) != 1 || models[0] != "TestUser" {
		t.Errorf("Expected 1 model named 'TestUser', got: %v", models)
	}

	model, exists := generator.GetModel("TestUser")
	if !exists {
		t.Error("Expected TestUser model to exist")
	}

	if model.Name != "TestUser" {
		t.Errorf("Expected model name 'TestUser', got: %s", model.Name)
	}

	if model.TableName != "test_users" {
		t.Errorf("Expected table name 'test_users', got: %s", model.TableName)
	}

	if len(model.Columns) != 4 {
		t.Errorf("Expected model to have 4 columns, got %d", len(model.Columns))
	}

	if generator.packageName != "test" {
		t.Errorf("Expected auto-detected package name 'test', got: %s", generator.packageName)
	}
}

func TestIsAutoGeneratedDefault(t *testing.T) {
	tests := []struct {
		name         string
		defaultValue string
		expected     bool
	}{

		{"gen_random_uuid", "gen_random_uuid()", true},
		{"uuid_generate_v4", "uuid_generate_v4()", true},
		{"uppercase UUID", "GEN_RANDOM_UUID()", true},

		{"gen_cuid", "gen_cuid()", true},
		{"cuid", "cuid()", true},
		{"uppercase CUID", "GEN_CUID()", true},

		{"now", "now()", true},
		{"current_timestamp", "CURRENT_TIMESTAMP", true},
		{"current_timestamp lowercase", "current_timestamp", true},

		{"nextval", "nextval('users_id_seq')", true},
		{"NEXTVAL", "NEXTVAL('users_id_seq')", true},

		{"string literal", "'default_value'", false},
		{"number literal", "42", false},
		{"boolean literal", "true", false},
		{"false boolean", "false", false},
		{"empty string", "", false},
		{"random function", "random_function()", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAutoGeneratedDefault(tt.defaultValue)
			if result != tt.expected {
				t.Errorf("isAutoGeneratedDefault(%q) = %v, expected %v", tt.defaultValue, result, tt.expected)
			}
		})
	}
}
