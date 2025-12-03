package introspect

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestExportJSON(t *testing.T) {
	schema := createTestSchema()
	inspector := &Inspector{}

	output, err := inspector.ExportSchema(schema, ExportFormatJSON)
	if err != nil {
		t.Fatalf("Failed to export JSON: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	if result["Name"] != "test_db" {
		t.Errorf("Expected database name to be 'test_db', got %v", result["Name"])
	}

	tables, ok := result["Tables"].(map[string]interface{})
	if !ok || len(tables) != 1 {
		t.Errorf("Expected 1 table in schema")
	}
}

func TestExportYAML(t *testing.T) {
	schema := createTestSchema()
	inspector := &Inspector{}

	output, err := inspector.ExportSchema(schema, ExportFormatYAML)
	if err != nil {
		t.Fatalf("Failed to export YAML: %v", err)
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal(output, &result); err != nil {
		t.Fatalf("Invalid YAML output: %v", err)
	}

	if result["name"] != "test_db" {
		t.Errorf("Expected database name to be 'test_db', got %v", result["name"])
	}
}

func TestExportMarkdown(t *testing.T) {
	schema := createTestSchema()
	inspector := &Inspector{}

	output, err := inspector.ExportSchema(schema, ExportFormatMarkdown)
	if err != nil {
		t.Fatalf("Failed to export Markdown: %v", err)
	}

	outputStr := string(output)

	expectedContents := []string{
		"# Database Schema: test_db",
		"## Database Information",
		"## Tables",
		"### users",
		"#### Columns",
		"| Name | Type | Nullable | Default | Description |",
		"| id | uuid | NO |",
		"| email | varchar(255) | NO |",
		"#### Primary Key",
		"- **Name**: users_pkey",
		"- **Columns**: id",
		"#### Foreign Keys",
		"#### Indexes",
		"- **idx_users_email** (UNIQUE): email",
		"## Enum Types",
		"### user_role",
		"- admin",
		"- user",
		"- guest",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Expected markdown to contain %q, but it didn't", expected)
		}
	}
}

func TestExportSQL(t *testing.T) {
	schema := createTestSchema()
	inspector := &Inspector{}

	output, err := inspector.ExportSchema(schema, ExportFormatSQL)
	if err != nil {
		t.Fatalf("Failed to export SQL: %v", err)
	}

	outputStr := string(output)

	expectedContents := []string{
		"-- Database: test_db",
		"-- Enum Types",
		"CREATE TYPE public.user_role AS ENUM",
		"'admin'",
		"'user'",
		"'guest'",
		"CREATE TABLE users",
		"id uuid NOT NULL DEFAULT gen_random_uuid()",
		"email varchar(255) NOT NULL",
		"CONSTRAINT users_pkey PRIMARY KEY (id)",
		"ALTER TABLE users ADD CONSTRAINT fk_users_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE",
		"CREATE UNIQUE INDEX idx_users_email ON users (email)",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Expected SQL to contain %q, but it didn't.\nSQL:\n%s", expected, outputStr)
		}
	}
}

func TestExportDOT(t *testing.T) {
	schema := createTestSchema()
	inspector := &Inspector{}

	output, err := inspector.ExportSchema(schema, ExportFormatDOT)
	if err != nil {
		t.Fatalf("Failed to export DOT: %v", err)
	}

	outputStr := string(output)

	expectedContents := []string{
		"digraph DatabaseSchema {",
		"rankdir=LR;",
		"node [shape=box];",
		`users [label="users|{id: uuid (PK)\lemail: varchar(255)\lteam_id: uuid\lcreated_at: timestamptz}"`,
		"users -> teams",
		"fk_users_team",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Expected DOT to contain %q, but it didn't.\nDOT:\n%s", expected, outputStr)
		}
	}
}

func TestExportUnsupportedFormat(t *testing.T) {
	schema := createTestSchema()
	inspector := &Inspector{}

	_, err := inspector.ExportSchema(schema, "invalid")
	if err == nil {
		t.Error("Expected error for unsupported format")
	}

	if !strings.Contains(err.Error(), "unsupported export format") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestSortedTables(t *testing.T) {
	tables := map[string]*TableSchema{
		"zebra":    {Name: "zebra"},
		"apple":    {Name: "apple"},
		"mongoose": {Name: "mongoose"},
		"banana":   {Name: "banana"},
	}

	sorted := sortedTables(tables)

	expectedOrder := []string{"apple", "banana", "mongoose", "zebra"}

	if len(sorted) != len(expectedOrder) {
		t.Fatalf("Expected %d tables, got %d", len(expectedOrder), len(sorted))
	}

	for i, expected := range expectedOrder {
		if sorted[i].Name != expected {
			t.Errorf("Expected table at position %d to be %s, got %s", i, expected, sorted[i].Name)
		}
	}
}

func TestExportSQL_WithViews(t *testing.T) {
	schema := createTestSchema()

	schema.Views["user_view"] = &ViewSchema{
		Name:       "user_view",
		Schema:     "public",
		Definition: "SELECT id, email FROM users WHERE active = true",
	}

	inspector := &Inspector{}
	output, err := inspector.ExportSchema(schema, ExportFormatSQL)
	if err != nil {
		t.Fatalf("Failed to export SQL with views: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "CREATE VIEW user_view") {
		t.Error("Expected SQL to contain view creation")
	}
}

func TestExportSQL_WithSequences(t *testing.T) {
	schema := createTestSchema()

	schema.Sequences["users_id_seq"] = &SequenceSchema{
		Name:        "users_id_seq",
		Schema:      "public",
		DataType:    "bigint",
		StartValue:  1,
		MinValue:    1,
		MaxValue:    9223372036854775807,
		Increment:   1,
		CycleOption: false,
		OwnedBy:     "users.id",
	}

	inspector := &Inspector{}
	output, err := inspector.ExportSchema(schema, ExportFormatSQL)
	if err != nil {
		t.Fatalf("Failed to export SQL with sequences: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "CREATE SEQUENCE users_id_seq") {
		t.Error("Expected SQL to contain sequence creation")
	}
}

func TestExportSQL_WithFunctions(t *testing.T) {
	schema := createTestSchema()

	schema.Functions["test_func"] = &FunctionSchema{
		Name:       "test_func",
		Schema:     "public",
		ReturnType: "integer",
		Arguments: []FunctionArgument{
			{
				Name:     "input_val",
				DataType: "integer",
				Mode:     "IN",
			},
		},
		Language:   "plpgsql",
		Definition: "BEGIN RETURN input_val * 2; END;",
		IsVolatile: false,
	}

	inspector := &Inspector{}
	output, err := inspector.ExportSchema(schema, ExportFormatSQL)
	if err != nil {
		t.Fatalf("Failed to export SQL with functions: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "CREATE FUNCTION test_func") {
		t.Error("Expected SQL to contain function creation")
	}
}

func TestExportMarkdown_EmptySchema(t *testing.T) {
	schema := &DatabaseSchema{
		Name:      "empty_db",
		Tables:    map[string]*TableSchema{},
		Enums:     map[string]*EnumSchema{},
		Views:     map[string]*ViewSchema{},
		Functions: map[string]*FunctionSchema{},
		Sequences: map[string]*SequenceSchema{},
		Metadata:  DatabaseMetadata{},
	}

	inspector := &Inspector{}
	output, err := inspector.ExportSchema(schema, ExportFormatMarkdown)
	if err != nil {
		t.Fatalf("Failed to export empty schema: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "# Database Schema: empty_db") {
		t.Error("Expected markdown to contain database header")
	}
	if !strings.Contains(outputStr, "No tables found") {
		t.Error("Expected markdown to indicate no tables")
	}
}

// Helper function to create test schema
func createTestSchema() *DatabaseSchema {
	return &DatabaseSchema{
		Name: "test_db",
		Tables: map[string]*TableSchema{
			"users": {
				Name:   "users",
				Schema: "public",
				Columns: []*ColumnSchema{
					{
						Name:         "id",
						DataType:     "uuid",
						IsNullable:   false,
						DefaultValue: stringPtr("gen_random_uuid()"),
					},
					{
						Name:          "email",
						DataType:      "varchar(255)",
						CharMaxLength: intPtr(255),
						IsNullable:    false,
					},
					{
						Name:       "team_id",
						DataType:   "uuid",
						IsNullable: false,
					},
					{
						Name:         "created_at",
						DataType:     "timestamptz",
						IsNullable:   false,
						DefaultValue: stringPtr("now()"),
					},
				},
				PrimaryKey: &PrimaryKeySchema{
					Name:    "users_pkey",
					Columns: []string{"id"},
				},
				ForeignKeys: []*ForeignKeySchema{
					{
						Name:              "fk_users_team",
						Columns:           []string{"team_id"},
						ReferencedTable:   "teams",
						ReferencedColumns: []string{"id"},
						OnDelete:          "CASCADE",
						OnUpdate:          "NO ACTION",
					},
				},
				Indexes: []*IndexSchema{
					{
						Name:     "idx_users_email",
						Columns:  []IndexColumn{{Name: "email"}},
						IsUnique: true,
					},
				},
			},
		},
		Enums: map[string]*EnumSchema{
			"public.user_role": {
				Name:   "user_role",
				Schema: "public",
				Values: []string{"admin", "user", "guest"},
			},
		},
		Views:     map[string]*ViewSchema{},
		Functions: map[string]*FunctionSchema{},
		Sequences: map[string]*SequenceSchema{},
		Metadata: DatabaseMetadata{
			Version:         "PostgreSQL 15.1",
			Encoding:        "UTF8",
			Collation:       "en_US.UTF-8",
			Size:            1024 * 1024 * 100,
			TableCount:      1,
			IndexCount:      2,
			ConstraintCount: 2,
			InspectedAt:     time.Now(),
		},
	}
}
