package generator

import (
	"strings"
	"testing"

	"github.com/eleven-am/storm/internal/parser"
)

func TestNewSchemaGenerator(t *testing.T) {
	gen := NewSchemaGenerator()
	if gen == nil {
		t.Error("NewSchemaGenerator should not return nil")
	}
	if gen.tagParser == nil {
		t.Error("NewSchemaGenerator should initialize tagParser")
	}
}

func TestSchemaGenerator_GenerateSchema(t *testing.T) {
	gen := NewSchemaGenerator()

	t.Run("generates simple schema", func(t *testing.T) {
		tables := []parser.TableDefinition{
			{
				TableName: "users",
				Fields: []parser.FieldDefinition{
					{
						Name:      "ID",
						Type:      "int",
						DBName:    "id",
						IsPointer: false,
						DBDef:     map[string]string{"primary_key": "true"},
					},
					{
						Name:      "Email",
						Type:      "string",
						DBName:    "email",
						IsPointer: false,
						DBDef:     map[string]string{"unique": "true"},
					},
				},
				TableLevel: map[string]string{},
			},
		}

		schema, err := gen.GenerateSchema(tables)
		if err != nil {
			t.Fatalf("GenerateSchema failed: %v", err)
		}

		if schema == nil {
			t.Error("schema should not be nil")
		}

		if len(schema.Tables) != 1 {
			t.Errorf("expected 1 table, got %d", len(schema.Tables))
		}

		userTable, exists := schema.Tables["users"]
		if !exists {
			t.Error("users table should exist")
		}

		if userTable.Name != "users" {
			t.Errorf("expected table name 'users', got '%s'", userTable.Name)
		}

		if len(userTable.Columns) != 2 {
			t.Errorf("expected 2 columns, got %d", len(userTable.Columns))
		}
	})

	t.Run("generates schema with enum types", func(t *testing.T) {
		tables := []parser.TableDefinition{
			{
				TableName: "users",
				Fields: []parser.FieldDefinition{
					{
						Name:      "Status",
						Type:      "string",
						DBName:    "status",
						IsPointer: false,
						DBDef: map[string]string{
							"enum": "active,inactive,pending",
						},
					},
				},
				TableLevel: map[string]string{},
			},
		}

		schema, err := gen.GenerateSchema(tables)
		if err != nil {
			t.Fatalf("GenerateSchema failed: %v", err)
		}

		if len(schema.EnumTypes) != 1 {
			t.Errorf("expected 1 enum type, got %d", len(schema.EnumTypes))
		}

		enumType, exists := schema.EnumTypes["users_status_enum"]
		if !exists {
			t.Error("users_status_enum should exist")
		}

		expectedValues := []string{"active", "inactive", "pending"}
		if len(enumType) != len(expectedValues) {
			t.Errorf("expected %d enum values, got %d", len(expectedValues), len(enumType))
		}
	})

	t.Run("validates foreign keys", func(t *testing.T) {
		tables := []parser.TableDefinition{
			{
				TableName: "posts",
				Fields: []parser.FieldDefinition{
					{
						Name:      "UserID",
						Type:      "int",
						DBName:    "user_id",
						IsPointer: false,
						DBDef: map[string]string{
							"foreign_key": "users.id",
						},
					},
				},
				TableLevel: map[string]string{},
			},
		}

		_, err := gen.GenerateSchema(tables)
		if err == nil {
			t.Error("expected error for invalid foreign key reference")
		}
		if !strings.Contains(err.Error(), "foreign key validation failed") {
			t.Errorf("expected foreign key validation error, got: %v", err)
		}
	})

	t.Run("passes validation with valid foreign keys", func(t *testing.T) {
		tables := []parser.TableDefinition{
			{
				TableName: "users",
				Fields: []parser.FieldDefinition{
					{
						Name:      "ID",
						Type:      "int",
						DBName:    "id",
						IsPointer: false,
						DBDef:     map[string]string{"primary_key": "true"},
					},
				},
				TableLevel: map[string]string{},
			},
			{
				TableName: "posts",
				Fields: []parser.FieldDefinition{
					{
						Name:      "UserID",
						Type:      "int",
						DBName:    "user_id",
						IsPointer: false,
						DBDef: map[string]string{
							"foreign_key": "users.id",
						},
					},
				},
				TableLevel: map[string]string{},
			},
		}

		schema, err := gen.GenerateSchema(tables)
		if err != nil {
			t.Fatalf("GenerateSchema failed: %v", err)
		}

		if len(schema.Tables) != 2 {
			t.Errorf("expected 2 tables, got %d", len(schema.Tables))
		}
	})
}

func TestSchemaGenerator_generateTable(t *testing.T) {
	gen := NewSchemaGenerator()

	t.Run("generates table with columns", func(t *testing.T) {
		tableDef := parser.TableDefinition{
			TableName: "users",
			Fields: []parser.FieldDefinition{
				{
					Name:      "ID",
					Type:      "int",
					DBName:    "id",
					IsPointer: false,
					DBDef:     map[string]string{"primary_key": "true"},
				},
				{
					Name:      "Email",
					Type:      "string",
					DBName:    "email",
					IsPointer: false,
					DBDef:     map[string]string{"unique": "true"},
				},
			},
			TableLevel: map[string]string{},
		}

		table, err := gen.generateTable(tableDef)
		if err != nil {
			t.Fatalf("generateTable failed: %v", err)
		}

		if table.Name != "users" {
			t.Errorf("expected table name 'users', got '%s'", table.Name)
		}

		if len(table.Columns) != 2 {
			t.Errorf("expected 2 columns, got %d", len(table.Columns))
		}

		idColumn := table.Columns[0]
		if idColumn.Name != "id" {
			t.Errorf("expected column name 'id', got '%s'", idColumn.Name)
		}
		if !idColumn.IsPrimaryKey {
			t.Error("id column should be primary key")
		}

		emailColumn := table.Columns[1]
		if emailColumn.Name != "email" {
			t.Errorf("expected column name 'email', got '%s'", emailColumn.Name)
		}
		if !emailColumn.IsUnique {
			t.Error("email column should be unique")
		}
	})

	t.Run("processes table-level definitions", func(t *testing.T) {
		tableDef := parser.TableDefinition{
			TableName: "users",
			Fields: []parser.FieldDefinition{
				{
					Name:      "ID",
					Type:      "int",
					DBName:    "id",
					IsPointer: false,
					DBDef:     map[string]string{"primary_key": "true"},
				},
			},
			TableLevel: map[string]string{
				"index": "idx_users_id,id",
			},
		}

		table, err := gen.generateTable(tableDef)
		if err != nil {
			t.Fatalf("generateTable failed: %v", err)
		}

		if len(table.Indexes) != 1 {
			t.Errorf("expected 1 index, got %d", len(table.Indexes))
		}

		index := table.Indexes[0]
		if index.Name != "idx_users_id" {
			t.Errorf("expected index name 'idx_users_id', got '%s'", index.Name)
		}
	})

	t.Run("adds implicit constraints", func(t *testing.T) {
		tableDef := parser.TableDefinition{
			TableName: "users",
			Fields: []parser.FieldDefinition{
				{
					Name:      "ID",
					Type:      "int",
					DBName:    "id",
					IsPointer: false,
					DBDef:     map[string]string{"primary_key": "true"},
				},
			},
			TableLevel: map[string]string{},
		}

		table, err := gen.generateTable(tableDef)
		if err != nil {
			t.Fatalf("generateTable failed: %v", err)
		}

		foundPK := false
		for _, constraint := range table.Constraints {
			if constraint.Type == "PRIMARY KEY" {
				foundPK = true
				break
			}
		}
		if !foundPK {
			t.Error("should have primary key constraint")
		}
	})
}

func TestSchemaGenerator_generateColumn(t *testing.T) {
	gen := NewSchemaGenerator()

	t.Run("generates basic column", func(t *testing.T) {
		field := parser.FieldDefinition{
			Name:      "Email",
			Type:      "string",
			DBName:    "email",
			IsPointer: false,
			DBDef:     map[string]string{},
		}

		column, err := gen.generateColumn(field, "users")
		if err != nil {
			t.Fatalf("generateColumn failed: %v", err)
		}

		if column.Name != "email" {
			t.Errorf("expected column name 'email', got '%s'", column.Name)
		}
		if column.Type != "TEXT" {
			t.Errorf("expected column type 'TEXT', got '%s'", column.Type)
		}
		if !column.IsNullable {
			t.Error("column should be nullable by default")
		}
	})

	t.Run("generates primary key column", func(t *testing.T) {
		field := parser.FieldDefinition{
			Name:      "ID",
			Type:      "int",
			DBName:    "id",
			IsPointer: false,
			DBDef:     map[string]string{"primary_key": "true"},
		}

		column, err := gen.generateColumn(field, "users")
		if err != nil {
			t.Fatalf("generateColumn failed: %v", err)
		}

		if !column.IsPrimaryKey {
			t.Error("column should be primary key")
		}
		if column.IsNullable {
			t.Error("primary key column should not be nullable")
		}
	})

	t.Run("generates nullable column", func(t *testing.T) {
		field := parser.FieldDefinition{
			Name:      "Email",
			Type:      "string",
			DBName:    "email",
			IsPointer: true,
			DBDef:     map[string]string{},
		}

		column, err := gen.generateColumn(field, "users")
		if err != nil {
			t.Fatalf("generateColumn failed: %v", err)
		}

		if !column.IsNullable {
			t.Error("pointer field should be nullable")
		}
	})

	t.Run("generates column with default value", func(t *testing.T) {
		field := parser.FieldDefinition{
			Name:      "Status",
			Type:      "string",
			DBName:    "status",
			IsPointer: false,
			DBDef:     map[string]string{"default": "'active'"},
		}

		column, err := gen.generateColumn(field, "users")
		if err != nil {
			t.Fatalf("generateColumn failed: %v", err)
		}

		if column.DefaultValue == nil {
			t.Error("column should have default value")
		}
		if *column.DefaultValue != "'active'" {
			t.Errorf("expected default value \"'active'\", got '%s'", *column.DefaultValue)
		}
	})

	t.Run("generates column with foreign key", func(t *testing.T) {
		field := parser.FieldDefinition{
			Name:      "UserID",
			Type:      "int",
			DBName:    "user_id",
			IsPointer: false,
			DBDef: map[string]string{
				"foreign_key": "users.id",
				"on_delete":   "CASCADE",
				"on_update":   "RESTRICT",
			},
		}

		column, err := gen.generateColumn(field, "posts")
		if err != nil {
			t.Fatalf("generateColumn failed: %v", err)
		}

		if column.ForeignKey == nil {
			t.Error("column should have foreign key")
		}
		if column.ForeignKey.ReferencedTable != "users" {
			t.Errorf("expected referenced table 'users', got '%s'", column.ForeignKey.ReferencedTable)
		}
		if column.ForeignKey.ReferencedColumn != "id" {
			t.Errorf("expected referenced column 'id', got '%s'", column.ForeignKey.ReferencedColumn)
		}
		if column.ForeignKey.OnDelete != "CASCADE" {
			t.Errorf("expected on delete 'CASCADE', got '%s'", column.ForeignKey.OnDelete)
		}
		if column.ForeignKey.OnUpdate != "RESTRICT" {
			t.Errorf("expected on update 'RESTRICT', got '%s'", column.ForeignKey.OnUpdate)
		}
	})

	t.Run("generates column with check constraint", func(t *testing.T) {
		field := parser.FieldDefinition{
			Name:      "Age",
			Type:      "int",
			DBName:    "age",
			IsPointer: false,
			DBDef:     map[string]string{"check": "age > 0"},
		}

		column, err := gen.generateColumn(field, "users")
		if err != nil {
			t.Fatalf("generateColumn failed: %v", err)
		}

		if column.CheckConstraint == nil {
			t.Error("column should have check constraint")
		}
		if *column.CheckConstraint != "age > 0" {
			t.Errorf("expected check constraint 'age > 0', got '%s'", *column.CheckConstraint)
		}
	})

	t.Run("generates column with enum values", func(t *testing.T) {
		field := parser.FieldDefinition{
			Name:      "Status",
			Type:      "string",
			DBName:    "status",
			IsPointer: false,
			DBDef:     map[string]string{"enum": "active,inactive,pending"},
		}

		column, err := gen.generateColumn(field, "users")
		if err != nil {
			t.Fatalf("generateColumn failed: %v", err)
		}

		if len(column.EnumValues) != 3 {
			t.Errorf("expected 3 enum values, got %d", len(column.EnumValues))
		}
		expectedValues := []string{"active", "inactive", "pending"}
		for i, expected := range expectedValues {
			if column.EnumValues[i] != expected {
				t.Errorf("expected enum value '%s', got '%s'", expected, column.EnumValues[i])
			}
		}
		if column.Type != "users_status_enum" {
			t.Errorf("expected type 'users_status_enum', got '%s'", column.Type)
		}
	})

	t.Run("generates array column", func(t *testing.T) {
		field := parser.FieldDefinition{
			Name:      "Tags",
			Type:      "[]string",
			DBName:    "tags",
			IsPointer: false,
			IsArray:   true,
			DBDef:     map[string]string{},
		}

		column, err := gen.generateColumn(field, "posts")
		if err != nil {
			t.Fatalf("generateColumn failed: %v", err)
		}

		if column.Type != "TEXT[]" {
			t.Errorf("expected type 'TEXT[]', got '%s'", column.Type)
		}
	})
}

func TestSchemaGenerator_mapGoTypeToPostgreSQL(t *testing.T) {
	gen := NewSchemaGenerator()

	tests := []struct {
		name     string
		goType   string
		dbDef    map[string]string
		expected string
	}{
		{"string", "string", map[string]string{}, "TEXT"},
		{"int", "int", map[string]string{}, "INTEGER"},
		{"int32", "int32", map[string]string{}, "INTEGER"},
		{"int64", "int64", map[string]string{}, "BIGINT"},
		{"int16", "int16", map[string]string{}, "SMALLINT"},
		{"float32", "float32", map[string]string{}, "REAL"},
		{"float64", "float64", map[string]string{}, "DOUBLE PRECISION"},
		{"bool", "bool", map[string]string{}, "BOOLEAN"},
		{"time.Time", "time.Time", map[string]string{}, "TIMESTAMPTZ"},
		{"[]byte", "[]byte", map[string]string{}, "BYTEA"},
		{"custom type with explicit db type", "CustomType", map[string]string{"type": "VARCHAR(255)"}, "VARCHAR(255)"},
		{"CUID type", "string", map[string]string{"type": "cuid"}, "CHAR(25)"},
		{"CUID2 type", "string", map[string]string{"type": "cuid2"}, "VARCHAR(32)"},
		{"unknown type", "UnknownType", map[string]string{}, "TEXT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := gen.mapGoTypeToPostgreSQL(tt.goType, tt.dbDef)
			if err != nil {
				t.Fatalf("mapGoTypeToPostgreSQL failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestSchemaGenerator_parseForeignKeyRef(t *testing.T) {
	gen := NewSchemaGenerator()

	t.Run("parses valid foreign key reference", func(t *testing.T) {
		fkRef, err := gen.parseForeignKeyRef("users.id")
		if err != nil {
			t.Fatalf("parseForeignKeyRef failed: %v", err)
		}

		if fkRef.ReferencedTable != "users" {
			t.Errorf("expected referenced table 'users', got '%s'", fkRef.ReferencedTable)
		}
		if fkRef.ReferencedColumn != "id" {
			t.Errorf("expected referenced column 'id', got '%s'", fkRef.ReferencedColumn)
		}
		if fkRef.OnDelete != "NO ACTION" {
			t.Errorf("expected on delete 'NO ACTION', got '%s'", fkRef.OnDelete)
		}
		if fkRef.OnUpdate != "NO ACTION" {
			t.Errorf("expected on update 'NO ACTION', got '%s'", fkRef.OnUpdate)
		}
	})

	t.Run("fails with invalid format", func(t *testing.T) {
		_, err := gen.parseForeignKeyRef("invalid")
		if err == nil {
			t.Error("expected error for invalid foreign key format")
		}
		if !strings.Contains(err.Error(), "foreign key must be in format 'table.column'") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("handles whitespace", func(t *testing.T) {
		fkRef, err := gen.parseForeignKeyRef("  users  .  id  ")
		if err != nil {
			t.Fatalf("parseForeignKeyRef failed: %v", err)
		}

		if fkRef.ReferencedTable != "users" {
			t.Errorf("expected referenced table 'users', got '%s'", fkRef.ReferencedTable)
		}
		if fkRef.ReferencedColumn != "id" {
			t.Errorf("expected referenced column 'id', got '%s'", fkRef.ReferencedColumn)
		}
	})
}

func TestSchemaGenerator_processTableLevel(t *testing.T) {
	gen := NewSchemaGenerator()

	t.Run("processes index definition", func(t *testing.T) {
		table := &SchemaTable{
			Name:        "users",
			Columns:     []SchemaColumn{},
			Indexes:     []SchemaIndex{},
			Constraints: []SchemaConstraint{},
		}

		tableLevelDef := map[string]string{
			"index": "idx_users_email,email",
		}

		err := gen.processTableLevel(tableLevelDef, table)
		if err != nil {
			t.Fatalf("processTableLevel failed: %v", err)
		}

		if len(table.Indexes) != 1 {
			t.Errorf("expected 1 index, got %d", len(table.Indexes))
		}

		index := table.Indexes[0]
		if index.Name != "idx_users_email" {
			t.Errorf("expected index name 'idx_users_email', got '%s'", index.Name)
		}
		if len(index.Columns) != 1 || index.Columns[0] != "email" {
			t.Errorf("expected index columns ['email'], got %v", index.Columns)
		}
	})

	t.Run("processes unique constraint", func(t *testing.T) {
		table := &SchemaTable{
			Name:        "users",
			Columns:     []SchemaColumn{},
			Indexes:     []SchemaIndex{},
			Constraints: []SchemaConstraint{},
		}

		tableLevelDef := map[string]string{
			"unique": "uq_users_email,email",
		}

		err := gen.processTableLevel(tableLevelDef, table)
		if err != nil {
			t.Fatalf("processTableLevel failed: %v", err)
		}

		if len(table.Constraints) != 1 {
			t.Errorf("expected 1 constraint, got %d", len(table.Constraints))
		}

		constraint := table.Constraints[0]
		if constraint.Type != "UNIQUE" {
			t.Errorf("expected constraint type 'UNIQUE', got '%s'", constraint.Type)
		}
		if constraint.Name != "uq_users_email" {
			t.Errorf("expected constraint name 'uq_users_email', got '%s'", constraint.Name)
		}
	})

	t.Run("processes check constraint", func(t *testing.T) {
		table := &SchemaTable{
			Name:        "users",
			Columns:     []SchemaColumn{},
			Indexes:     []SchemaIndex{},
			Constraints: []SchemaConstraint{},
		}

		tableLevelDef := map[string]string{
			"check": "chk_users_age,age > 0",
		}

		err := gen.processTableLevel(tableLevelDef, table)
		if err != nil {
			t.Fatalf("processTableLevel failed: %v", err)
		}

		if len(table.Constraints) != 1 {
			t.Errorf("expected 1 constraint, got %d", len(table.Constraints))
		}

		constraint := table.Constraints[0]
		if constraint.Type != "CHECK" {
			t.Errorf("expected constraint type 'CHECK', got '%s'", constraint.Type)
		}
		if constraint.Definition != "age > 0" {
			t.Errorf("expected constraint definition 'age > 0', got '%s'", constraint.Definition)
		}
	})

	t.Run("handles unique constraint with where clause", func(t *testing.T) {
		table := &SchemaTable{
			Name:        "users",
			Columns:     []SchemaColumn{},
			Indexes:     []SchemaIndex{},
			Constraints: []SchemaConstraint{},
		}

		tableLevelDef := map[string]string{
			"unique": "idx_active_users,email where:active = true",
		}

		err := gen.processTableLevel(tableLevelDef, table)
		if err != nil {
			t.Fatalf("processTableLevel failed: %v", err)
		}

		if len(table.Indexes) != 1 {
			t.Errorf("expected 1 index, got %d", len(table.Indexes))
		}

		index := table.Indexes[0]
		if !index.IsUnique {
			t.Error("index should be unique")
		}
		if index.Where != "active = true" {
			t.Errorf("expected where clause 'active = true', got '%s'", index.Where)
		}
	})

	t.Run("ignores unknown table-level attributes", func(t *testing.T) {
		table := &SchemaTable{
			Name:        "users",
			Columns:     []SchemaColumn{},
			Indexes:     []SchemaIndex{},
			Constraints: []SchemaConstraint{},
		}

		tableLevelDef := map[string]string{
			"unknown": "value",
		}

		err := gen.processTableLevel(tableLevelDef, table)
		if err != nil {
			t.Fatalf("processTableLevel failed: %v", err)
		}

		if len(table.Indexes) != 0 {
			t.Errorf("expected 0 indexes, got %d", len(table.Indexes))
		}
		if len(table.Constraints) != 0 {
			t.Errorf("expected 0 constraints, got %d", len(table.Constraints))
		}
	})
}

func TestSchemaGenerator_parseIndexDefinition(t *testing.T) {
	gen := NewSchemaGenerator()

	t.Run("parses basic index", func(t *testing.T) {
		indexes, err := gen.parseIndexDefinition("idx_users_email,email", "users")
		if err != nil {
			t.Fatalf("parseIndexDefinition failed: %v", err)
		}

		if len(indexes) != 1 {
			t.Errorf("expected 1 index, got %d", len(indexes))
		}

		index := indexes[0]
		if index.Name != "idx_users_email" {
			t.Errorf("expected index name 'idx_users_email', got '%s'", index.Name)
		}
		if len(index.Columns) != 1 || index.Columns[0] != "email" {
			t.Errorf("expected columns ['email'], got %v", index.Columns)
		}
		if index.IsUnique {
			t.Error("index should not be unique")
		}
	})

	t.Run("parses composite index", func(t *testing.T) {
		indexes, err := gen.parseIndexDefinition("idx_users_name,first_name,last_name", "users")
		if err != nil {
			t.Fatalf("parseIndexDefinition failed: %v", err)
		}

		if len(indexes) != 1 {
			t.Errorf("expected 1 index, got %d", len(indexes))
		}

		index := indexes[0]
		if len(index.Columns) != 2 {
			t.Errorf("expected 2 columns, got %d", len(index.Columns))
		}
		if index.Columns[0] != "first_name" || index.Columns[1] != "last_name" {
			t.Errorf("expected columns ['first_name', 'last_name'], got %v", index.Columns)
		}
	})

	t.Run("parses index with unique flag", func(t *testing.T) {
		indexes, err := gen.parseIndexDefinition("idx_users_email,email,unique", "users")
		if err != nil {
			t.Fatalf("parseIndexDefinition failed: %v", err)
		}

		if len(indexes) != 1 {
			t.Errorf("expected 1 index, got %d", len(indexes))
		}

		index := indexes[0]
		if !index.IsUnique {
			t.Error("index should be unique")
		}
	})

	t.Run("parses index with where clause", func(t *testing.T) {
		indexes, err := gen.parseIndexDefinition("idx_active_users,email where:active = true", "users")
		if err != nil {
			t.Fatalf("parseIndexDefinition failed: %v", err)
		}

		if len(indexes) != 1 {
			t.Errorf("expected 1 index, got %d", len(indexes))
		}

		index := indexes[0]
		if index.Where != "active = true" {
			t.Errorf("expected where clause 'active = true', got '%s'", index.Where)
		}
	})

	t.Run("parses index with using clause", func(t *testing.T) {
		indexes, err := gen.parseIndexDefinition("idx_users_data,data using:gin", "users")
		if err != nil {
			t.Fatalf("parseIndexDefinition failed: %v", err)
		}

		if len(indexes) != 1 {
			t.Errorf("expected 1 index, got %d", len(indexes))
		}

		index := indexes[0]
		if index.Type != "gin" {
			t.Errorf("expected index type 'gin', got '%s'", index.Type)
		}
	})

	t.Run("parses multiple indexes", func(t *testing.T) {
		indexes, err := gen.parseIndexDefinition("idx_users_email,email;idx_users_name,name", "users")
		if err != nil {
			t.Fatalf("parseIndexDefinition failed: %v", err)
		}

		if len(indexes) != 2 {
			t.Errorf("expected 2 indexes, got %d", len(indexes))
		}

		if indexes[0].Name != "idx_users_email" || indexes[1].Name != "idx_users_name" {
			t.Errorf("unexpected index names: %s, %s", indexes[0].Name, indexes[1].Name)
		}
	})

	t.Run("handles column ordering", func(t *testing.T) {
		indexes, err := gen.parseIndexDefinition("idx_users_name,name desc", "users")
		if err != nil {
			t.Fatalf("parseIndexDefinition failed: %v", err)
		}

		if len(indexes) != 1 {
			t.Errorf("expected 1 index, got %d", len(indexes))
		}

		index := indexes[0]
		if index.Columns[0] != "name DESC" {
			t.Errorf("expected column 'name DESC', got '%s'", index.Columns[0])
		}
	})

	t.Run("fails with invalid format", func(t *testing.T) {
		_, err := gen.parseIndexDefinition("invalid", "users")
		if err == nil {
			t.Error("expected error for invalid index format")
		}
		if !strings.Contains(err.Error(), "index definition must have at least name and one column") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestDatabaseSchema_HasTable(t *testing.T) {
	schema := &DatabaseSchema{
		Tables: map[string]SchemaTable{
			"users": {Name: "users"},
		},
	}

	if !schema.HasTable("users") {
		t.Error("schema should have users table")
	}
	if schema.HasTable("posts") {
		t.Error("schema should not have posts table")
	}
}

func TestDatabaseSchema_GetTable(t *testing.T) {
	userTable := SchemaTable{Name: "users"}
	schema := &DatabaseSchema{
		Tables: map[string]SchemaTable{
			"users": userTable,
		},
	}

	table, exists := schema.GetTable("users")
	if !exists {
		t.Error("users table should exist")
	}
	if table.Name != "users" {
		t.Errorf("expected table name 'users', got '%s'", table.Name)
	}

	_, exists = schema.GetTable("posts")
	if exists {
		t.Error("posts table should not exist")
	}
}

func TestSchemaGenerator_validateForeignKeys(t *testing.T) {
	gen := NewSchemaGenerator()

	t.Run("passes with valid foreign keys", func(t *testing.T) {
		schema := &DatabaseSchema{
			Tables: map[string]SchemaTable{
				"users": {
					Name: "users",
					Columns: []SchemaColumn{
						{Name: "id", Type: "INTEGER"},
					},
				},
				"posts": {
					Name: "posts",
					Columns: []SchemaColumn{
						{
							Name: "user_id",
							Type: "INTEGER",
							ForeignKey: &ForeignKeyRef{
								ReferencedTable:  "users",
								ReferencedColumn: "id",
							},
						},
					},
				},
			},
		}

		err := gen.validateForeignKeys(schema)
		if err != nil {
			t.Errorf("validateForeignKeys failed: %v", err)
		}
	})

	t.Run("fails with non-existent table", func(t *testing.T) {
		schema := &DatabaseSchema{
			Tables: map[string]SchemaTable{
				"posts": {
					Name: "posts",
					Columns: []SchemaColumn{
						{
							Name: "user_id",
							Type: "INTEGER",
							ForeignKey: &ForeignKeyRef{
								ReferencedTable:  "users",
								ReferencedColumn: "id",
							},
						},
					},
				},
			},
		}

		err := gen.validateForeignKeys(schema)
		if err == nil {
			t.Error("expected error for non-existent table")
		}
		if !strings.Contains(err.Error(), "foreign key references non-existent table") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("fails with non-existent column", func(t *testing.T) {
		schema := &DatabaseSchema{
			Tables: map[string]SchemaTable{
				"users": {
					Name: "users",
					Columns: []SchemaColumn{
						{Name: "id", Type: "INTEGER"},
					},
				},
				"posts": {
					Name: "posts",
					Columns: []SchemaColumn{
						{
							Name: "user_id",
							Type: "INTEGER",
							ForeignKey: &ForeignKeyRef{
								ReferencedTable:  "users",
								ReferencedColumn: "user_id",
							},
						},
					},
				},
			},
		}

		err := gen.validateForeignKeys(schema)
		if err == nil {
			t.Error("expected error for non-existent column")
		}
		if !strings.Contains(err.Error(), "foreign key references non-existent column") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestSchemaGenerator_addImplicitConstraints(t *testing.T) {
	gen := NewSchemaGenerator()

	t.Run("adds primary key constraint", func(t *testing.T) {
		table := &SchemaTable{
			Name: "users",
			Columns: []SchemaColumn{
				{Name: "id", Type: "INTEGER", IsPrimaryKey: true},
			},
			Constraints: []SchemaConstraint{},
		}

		gen.addImplicitConstraints(table)

		foundPK := false
		for _, constraint := range table.Constraints {
			if constraint.Type == "PRIMARY KEY" {
				foundPK = true
				if constraint.Name != "users_pkey" {
					t.Errorf("expected constraint name 'users_pkey', got '%s'", constraint.Name)
				}
				if len(constraint.Columns) != 1 || constraint.Columns[0] != "id" {
					t.Errorf("expected columns ['id'], got %v", constraint.Columns)
				}
				break
			}
		}
		if !foundPK {
			t.Error("should have added primary key constraint")
		}
	})

	t.Run("adds unique constraint", func(t *testing.T) {
		table := &SchemaTable{
			Name: "users",
			Columns: []SchemaColumn{
				{Name: "email", Type: "TEXT", IsUnique: true},
			},
			Constraints: []SchemaConstraint{},
		}

		gen.addImplicitConstraints(table)

		foundUnique := false
		for _, constraint := range table.Constraints {
			if constraint.Type == "UNIQUE" {
				foundUnique = true
				if constraint.Name != "users_email_key" {
					t.Errorf("expected constraint name 'users_email_key', got '%s'", constraint.Name)
				}
				break
			}
		}
		if !foundUnique {
			t.Error("should have added unique constraint")
		}
	})

	t.Run("adds foreign key constraint", func(t *testing.T) {
		table := &SchemaTable{
			Name: "posts",
			Columns: []SchemaColumn{
				{
					Name: "user_id",
					Type: "INTEGER",
					ForeignKey: &ForeignKeyRef{
						ReferencedTable:  "users",
						ReferencedColumn: "id",
						OnDelete:         "CASCADE",
						OnUpdate:         "RESTRICT",
					},
				},
			},
			Constraints: []SchemaConstraint{},
		}

		gen.addImplicitConstraints(table)

		foundFK := false
		for _, constraint := range table.Constraints {
			if constraint.Type == "FOREIGN KEY" {
				foundFK = true
				if constraint.Name != "posts_user_id_fkey" {
					t.Errorf("expected constraint name 'posts_user_id_fkey', got '%s'", constraint.Name)
				}
				expectedDef := "FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE RESTRICT"
				if constraint.Definition != expectedDef {
					t.Errorf("expected definition '%s', got '%s'", expectedDef, constraint.Definition)
				}
				break
			}
		}
		if !foundFK {
			t.Error("should have added foreign key constraint")
		}
	})
}

func TestDatabaseSchema_sortTablesByDependencies(t *testing.T) {
	schema := &DatabaseSchema{
		Tables: map[string]SchemaTable{
			"users": {
				Name: "users",
				Columns: []SchemaColumn{
					{Name: "id", Type: "INTEGER"},
				},
			},
			"posts": {
				Name: "posts",
				Columns: []SchemaColumn{
					{
						Name: "user_id",
						Type: "INTEGER",
						ForeignKey: &ForeignKeyRef{
							ReferencedTable:  "users",
							ReferencedColumn: "id",
						},
					},
				},
			},
			"comments": {
				Name: "comments",
				Columns: []SchemaColumn{
					{
						Name: "post_id",
						Type: "INTEGER",
						ForeignKey: &ForeignKeyRef{
							ReferencedTable:  "posts",
							ReferencedColumn: "id",
						},
					},
				},
			},
		},
	}

	tables := []string{"comments", "posts", "users"}
	sorted := schema.sortTablesByDependencies(tables)

	usersPos := -1
	postsPos := -1
	commentsPos := -1
	for i, table := range sorted {
		switch table {
		case "users":
			usersPos = i
		case "posts":
			postsPos = i
		case "comments":
			commentsPos = i
		}
	}

	if usersPos == -1 || postsPos == -1 || commentsPos == -1 {
		t.Error("all tables should be present in sorted result")
	}

	if usersPos > postsPos {
		t.Error("users should come before posts (dependency order)")
	}
	if postsPos > commentsPos {
		t.Error("posts should come before comments (dependency order)")
	}
}

func TestDatabaseSchema_GetTableNames(t *testing.T) {
	schema := &DatabaseSchema{
		Tables: map[string]SchemaTable{
			"users": {
				Name: "users",
				Columns: []SchemaColumn{
					{Name: "id", Type: "INTEGER"},
				},
			},
			"posts": {
				Name: "posts",
				Columns: []SchemaColumn{
					{
						Name: "user_id",
						Type: "INTEGER",
						ForeignKey: &ForeignKeyRef{
							ReferencedTable:  "users",
							ReferencedColumn: "id",
						},
					},
				},
			},
		},
	}

	tableNames := schema.GetTableNames()

	if len(tableNames) != 2 {
		t.Errorf("expected 2 table names, got %d", len(tableNames))
	}

	usersPos := -1
	postsPos := -1
	for i, name := range tableNames {
		switch name {
		case "users":
			usersPos = i
		case "posts":
			postsPos = i
		}
	}

	if usersPos == -1 || postsPos == -1 {
		t.Error("both table names should be present")
	}
	if usersPos > postsPos {
		t.Error("users should come before posts in dependency order")
	}
}
