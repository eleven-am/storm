package generator

import (
	"strings"
	"testing"
)

func TestSQLGenerator_GenerateCreateTable(t *testing.T) {
	gen := NewSQLGenerator()

	tests := []struct {
		name        string
		table       SchemaTable
		contains    []string
		notContains []string
	}{
		{
			name: "simple table",
			table: SchemaTable{
				Name: "users",
				Columns: []SchemaColumn{
					{
						Name:         "id",
						Type:         "UUID",
						IsPrimaryKey: true,
						DefaultValue: strPtr("gen_random_uuid()"),
					},
					{
						Name:       "email",
						Type:       "VARCHAR(255)",
						IsNullable: false,
						IsUnique:   true,
					},
					{
						Name:         "created_at",
						Type:         "TIMESTAMP",
						IsNullable:   false,
						DefaultValue: strPtr("now()"),
					},
				},
			},
			contains: []string{
				"CREATE TABLE users",
				"id UUID NOT NULL DEFAULT gen_random_uuid()",
				"email VARCHAR(255) NOT NULL",
				"created_at TIMESTAMP NOT NULL DEFAULT now()",
				"PRIMARY KEY (id)",
			},
			notContains: []string{
				"IF NOT EXISTS",
			},
		},
		{
			name: "table with foreign key",
			table: SchemaTable{
				Name: "teams",
				Columns: []SchemaColumn{
					{
						Name:         "id",
						Type:         "UUID",
						IsPrimaryKey: true,
					},
					{
						Name:       "owner_id",
						Type:       "UUID",
						IsNullable: false,
						ForeignKey: &ForeignKeyRef{
							ReferencedTable:  "users",
							ReferencedColumn: "id",
							OnDelete:         "CASCADE",
						},
					},
				},
			},
			contains: []string{
				"CREATE TABLE teams",
				"owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE",
			},
		},
		{
			name: "table with indexes",
			table: SchemaTable{
				Name: "products",
				Columns: []SchemaColumn{
					{
						Name:         "id",
						Type:         "SERIAL",
						IsPrimaryKey: true,
					},
					{
						Name: "sku",
						Type: "VARCHAR(50)",
					},
				},
				Indexes: []SchemaIndex{
					{
						Name:     "idx_products_sku",
						Columns:  []string{"sku"},
						IsUnique: true,
					},
				},
			},
			contains: []string{
				"CREATE TABLE products",
				"CREATE UNIQUE INDEX idx_products_sku ON products (sku)",
			},
		},
		{
			name: "table with check constraint",
			table: SchemaTable{
				Name: "orders",
				Columns: []SchemaColumn{
					{
						Name:         "id",
						Type:         "SERIAL",
						IsPrimaryKey: true,
					},
					{
						Name:            "amount",
						Type:            "NUMERIC(10,2)",
						CheckConstraint: strPtr("amount > 0"),
					},
				},
			},
			contains: []string{
				"amount NUMERIC(10,2) NOT NULL CHECK (amount > 0)",
			},
		},
		{
			name: "table with CUID",
			table: SchemaTable{
				Name: "accounts",
				Columns: []SchemaColumn{
					{
						Name:         "id",
						Type:         "CHAR(25)",
						IsPrimaryKey: true,
						DefaultValue: strPtr("gen_cuid()"),
					},
				},
			},
			contains: []string{
				"id CHAR(25) NOT NULL DEFAULT gen_cuid()",
			},
			notContains: []string{
				"CREATE OR REPLACE FUNCTION gen_cuid()",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := gen.GenerateCreateTable(tt.table)

			for _, expected := range tt.contains {
				if !strings.Contains(sql, expected) {
					t.Errorf("SQL should contain %q\nGot:\n%s", expected, sql)
				}
			}

			for _, unexpected := range tt.notContains {
				if strings.Contains(sql, unexpected) {
					t.Errorf("SQL should not contain %q\nGot:\n%s", unexpected, sql)
				}
			}
		})
	}
}

func TestSQLGenerator_GenerateIndexDDL(t *testing.T) {
	gen := NewSQLGenerator()

	tests := []struct {
		name      string
		tableName string
		index     SchemaIndex
		expected  string
	}{
		{
			name:      "simple index",
			tableName: "users",
			index: SchemaIndex{
				Name:    "idx_users_email",
				Columns: []string{"email"},
			},
			expected: "CREATE INDEX idx_users_email ON users (email);",
		},
		{
			name:      "unique index",
			tableName: "users",
			index: SchemaIndex{
				Name:     "idx_users_email",
				Columns:  []string{"email"},
				IsUnique: true,
			},
			expected: "CREATE UNIQUE INDEX idx_users_email ON users (email);",
		},
		{
			name:      "composite index",
			tableName: "users",
			index: SchemaIndex{
				Name:    "idx_users_name",
				Columns: []string{"first_name", "last_name"},
			},
			expected: "CREATE INDEX idx_users_name ON users (first_name, last_name);",
		},
		{
			name:      "partial index",
			tableName: "users",
			index: SchemaIndex{
				Name:    "idx_active_users",
				Columns: []string{"email"},
				Where:   "is_active = true",
			},
			expected: "CREATE INDEX idx_active_users ON users (email) WHERE is_active = true;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.GenerateIndexDDL(tt.tableName, tt.index)

			result = strings.TrimSuffix(result, "\n")
			if result != tt.expected {
				t.Errorf("Got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSQLGenerator_HasCUID(t *testing.T) {
	gen := NewSQLGenerator()

	tests := []struct {
		name     string
		schema   DatabaseSchema
		expected bool
	}{
		{
			name: "schema with CUID",
			schema: DatabaseSchema{
				Tables: map[string]SchemaTable{
					"users": {
						Columns: []SchemaColumn{
							{Type: "CHAR(25)", DefaultValue: strPtr("gen_cuid()")},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "schema with CUID2",
			schema: DatabaseSchema{
				Tables: map[string]SchemaTable{
					"users": {
						Columns: []SchemaColumn{
							{Type: "VARCHAR(32)", DefaultValue: strPtr("gen_cuid2()")},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "schema without CUID",
			schema: DatabaseSchema{
				Tables: map[string]SchemaTable{
					"users": {
						Columns: []SchemaColumn{
							{Type: "UUID", DefaultValue: strPtr("gen_random_uuid()")},
						},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.schemaUsesCUIDs(&tt.schema)
			if result != tt.expected {
				t.Errorf("hasCUID() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSQLGenerator_GenerateCreateDatabase(t *testing.T) {
	gen := NewSQLGenerator()

	schema := DatabaseSchema{
		Tables: map[string]SchemaTable{
			"users": {
				Name: "users",
				Columns: []SchemaColumn{
					{
						Name:         "id",
						Type:         "UUID",
						IsPrimaryKey: true,
					},
				},
			},
			"teams": {
				Name: "teams",
				Columns: []SchemaColumn{
					{
						Name:         "id",
						Type:         "UUID",
						IsPrimaryKey: true,
					},
					{
						Name: "owner_id",
						Type: "UUID",
						ForeignKey: &ForeignKeyRef{
							ReferencedTable:  "users",
							ReferencedColumn: "id",
						},
					},
				},
			},
		},
	}

	sql := gen.GenerateSchema(&schema)

	usersIndex := strings.Index(sql, "CREATE TABLE users")
	teamsIndex := strings.Index(sql, "CREATE TABLE teams")

	if usersIndex == -1 || teamsIndex == -1 {
		t.Error("Both tables should be created")
	}

	if usersIndex > teamsIndex {
		t.Error("users table should be created before teams table (dependency order)")
	}
}

func TestSQLGenerator_generateEnumType(t *testing.T) {
	gen := NewSQLGenerator()

	t.Run("generates enum type", func(t *testing.T) {
		sql := gen.generateEnumType("user_status_enum", []string{"active", "inactive", "pending"})

		expected := "CREATE TYPE user_status_enum AS ENUM ('active', 'inactive', 'pending');"
		if sql != expected {
			t.Errorf("Expected %q, got %q", expected, sql)
		}
	})

	t.Run("generates enum type with single value", func(t *testing.T) {
		sql := gen.generateEnumType("single_enum", []string{"only"})

		expected := "CREATE TYPE single_enum AS ENUM ('only');"
		if sql != expected {
			t.Errorf("Expected %q, got %q", expected, sql)
		}
	})

	t.Run("generates enum type with empty values", func(t *testing.T) {
		sql := gen.generateEnumType("empty_enum", []string{})

		expected := "CREATE TYPE empty_enum AS ENUM ();"
		if sql != expected {
			t.Errorf("Expected %q, got %q", expected, sql)
		}
	})
}

func TestSQLGenerator_GenerateSchema_WithEnums(t *testing.T) {
	gen := NewSQLGenerator()

	schema := DatabaseSchema{
		Tables: map[string]SchemaTable{
			"users": {
				Name: "users",
				Columns: []SchemaColumn{
					{
						Name:         "id",
						Type:         "SERIAL",
						IsPrimaryKey: true,
					},
					{
						Name: "status",
						Type: "user_status_enum",
					},
				},
			},
		},
		EnumTypes: map[string][]string{
			"user_status_enum": {"active", "inactive", "pending"},
		},
	}

	sql := gen.GenerateSchema(&schema)

	if !strings.Contains(sql, "CREATE TYPE user_status_enum AS ENUM") {
		t.Error("SQL should contain enum type creation")
	}
	if !strings.Contains(sql, "'active', 'inactive', 'pending'") {
		t.Error("SQL should contain enum values")
	}
}

func TestSQLGenerator_GenerateSchema_WithCUIDs(t *testing.T) {
	gen := NewSQLGenerator()

	schema := DatabaseSchema{
		Tables: map[string]SchemaTable{
			"users": {
				Name: "users",
				Columns: []SchemaColumn{
					{
						Name:         "id",
						Type:         "CHAR(25)",
						IsPrimaryKey: true,
						DefaultValue: strPtr("gen_cuid()"),
					},
				},
			},
		},
	}

	sql := gen.GenerateSchema(&schema)

	if !strings.Contains(sql, "CUID functions will be generated by the migration system") {
		t.Error("SQL should contain CUID migration system comment")
	}
}

func TestSQLGenerator_GenerateSchema_Extensions(t *testing.T) {
	gen := NewSQLGenerator()

	schema := DatabaseSchema{
		Tables: map[string]SchemaTable{
			"users": {
				Name: "users",
				Columns: []SchemaColumn{
					{
						Name:         "id",
						Type:         "UUID",
						IsPrimaryKey: true,
					},
				},
			},
		},
	}

	sql := gen.GenerateSchema(&schema)

	if !strings.Contains(sql, "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"") {
		t.Error("SQL should contain uuid-ossp extension")
	}
	if !strings.Contains(sql, "CREATE EXTENSION IF NOT EXISTS \"pgcrypto\"") {
		t.Error("SQL should contain pgcrypto extension")
	}
}

// Helper function
func strPtr(s string) *string {
	return &s
}
