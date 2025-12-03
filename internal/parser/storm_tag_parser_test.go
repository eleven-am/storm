package parser

import (
	"strings"
	"testing"
)

func TestStormTagParser_ParseStormTag(t *testing.T) {
	parser := NewStormTagParser()

	tests := []struct {
		name            string
		tag             string
		isRelationship  bool
		expectError     bool
		expectedColumn  string
		expectedType    string
		expectedRelType string
		expectedTarget  string
	}{
		{
			name:           "column with type and primary key",
			tag:            "column:id;type:uuid;primary_key;default:gen_random_uuid()",
			isRelationship: false,
			expectError:    false,
			expectedColumn: "id",
			expectedType:   "uuid",
		},
		{
			name:           "column with constraints",
			tag:            "column:email;type:varchar(255);not_null;unique",
			isRelationship: false,
			expectError:    false,
			expectedColumn: "email",
			expectedType:   "varchar(255)",
		},
		{
			name:            "belongs_to relationship",
			tag:             "relation:belongs_to:User;foreign_key:user_id",
			isRelationship:  true,
			expectError:     false,
			expectedRelType: "belongs_to",
			expectedTarget:  "User",
		},
		{
			name:            "has_many relationship",
			tag:             "relation:has_many:Todo;foreign_key:user_id",
			isRelationship:  true,
			expectError:     false,
			expectedRelType: "has_many",
			expectedTarget:  "Todo",
		},
		{
			name:            "has_many_through relationship",
			tag:             "relation:has_many_through:Tag;join_table:user_tags;source_fk:user_id;target_fk:tag_id",
			isRelationship:  true,
			expectError:     false,
			expectedRelType: "has_many_through",
			expectedTarget:  "Tag",
		},
		{
			name:           "invalid - mixed column and relationship",
			tag:            "column:id;type:uuid;relation:belongs_to:User",
			isRelationship: false,
			expectError:    true,
		},
		{
			name:           "invalid - empty tag",
			tag:            "",
			isRelationship: false,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.ParseStormTag(tt.tag, tt.isRelationship)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.expectedColumn != "" && parsed.Column != tt.expectedColumn {
				t.Errorf("expected column %s, got %s", tt.expectedColumn, parsed.Column)
			}

			if tt.expectedType != "" && parsed.Type != tt.expectedType {
				t.Errorf("expected type %s, got %s", tt.expectedType, parsed.Type)
			}

			if tt.expectedRelType != "" && parsed.RelationType != tt.expectedRelType {
				t.Errorf("expected relation type %s, got %s", tt.expectedRelType, parsed.RelationType)
			}

			if tt.expectedTarget != "" && parsed.RelationTarget != tt.expectedTarget {
				t.Errorf("expected relation target %s, got %s", tt.expectedTarget, parsed.RelationTarget)
			}

			if parsed.IsRelationship != tt.isRelationship {
				t.Errorf("expected isRelationship %v, got %v", tt.isRelationship, parsed.IsRelationship)
			}
		})
	}
}

func TestStormTagParser_ToDBDefAttributes(t *testing.T) {
	parser := NewStormTagParser()

	parsed, err := parser.ParseStormTag("column:id;type:uuid;primary_key;default:gen_random_uuid()", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	attrs := parsed.ToDBDefAttributes()

	expected := map[string]string{
		"type":        "uuid",
		"primary_key": "",
		"default":     "gen_random_uuid()",
	}

	for key, expectedValue := range expected {
		if actualValue, exists := attrs[key]; !exists {
			t.Errorf("missing attribute %s", key)
		} else if actualValue != expectedValue {
			t.Errorf("attribute %s: expected %s, got %s", key, expectedValue, actualValue)
		}
	}
}

func TestStormTagParser_ValidationErrors(t *testing.T) {
	parser := NewStormTagParser()

	errorTests := []struct {
		name           string
		tag            string
		isRelationship bool
		expectError    string
	}{
		{
			name:           "column with relationship attributes",
			tag:            "column:id;type:uuid;relation:belongs_to:User",
			isRelationship: false,
			expectError:    "column fields cannot have relationship attributes",
		},
		{
			name:           "relationship with column attributes",
			tag:            "relation:belongs_to:User;type:uuid;primary_key",
			isRelationship: true,
			expectError:    "relationship fields cannot have column attributes",
		},
		{
			name:           "invalid relationship type",
			tag:            "relation:invalid_type:User",
			isRelationship: true,
			expectError:    "invalid relationship type",
		},
		{
			name:           "missing join_table for has_many_through",
			tag:            "relation:has_many_through:Tag;source_fk:user_id;target_fk:tag_id",
			isRelationship: true,
			expectError:    "join_table is required for has_many_through relationships",
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ParseStormTag(tt.tag, tt.isRelationship)

			if err == nil {
				t.Errorf("expected error containing '%s' but got none", tt.expectError)
				return
			}

			if err.Error() == "" || !contains(err.Error(), tt.expectError) {
				t.Errorf("expected error containing '%s', got: %s", tt.expectError, err.Error())
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s[:len(substr)] == substr ||
		(len(s) > len(substr) && s[len(s)-len(substr):] == substr) ||
		(len(s) > len(substr) && strings.Contains(s, substr)))
}

func TestStormTagParser_ValidateForeignKey(t *testing.T) {
	parser := NewStormTagParser()

	tests := []struct {
		name        string
		tag         string
		expectError bool
	}{
		{
			name:        "valid foreign key",
			tag:         "relation:belongs_to:User;foreign_key:user_id",
			expectError: false,
		},
		{
			name:        "belongs_to without foreign_key",
			tag:         "relation:belongs_to:User",
			expectError: false,
		},
		{
			name:        "has_many with foreign_key",
			tag:         "relation:has_many:Todo;foreign_key:user_id",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ParseStormTag(tt.tag, true)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStormTagParser_ValidateCheck(t *testing.T) {
	parser := NewStormTagParser()

	tests := []struct {
		name        string
		tag         string
		expectError bool
	}{
		{
			name:        "valid check constraint",
			tag:         "column:status;type:varchar;check:status IN ('active', 'inactive')",
			expectError: false,
		},
		{
			name:        "column without check",
			tag:         "column:id;type:uuid;primary_key",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ParseStormTag(tt.tag, false)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStormTagParser_ValidateEnum(t *testing.T) {
	parser := NewStormTagParser()

	tests := []struct {
		name        string
		tag         string
		expectError bool
	}{
		{
			name:        "valid enum",
			tag:         "column:status;type:varchar;enum:active,inactive,pending",
			expectError: false,
		},
		{
			name:        "column without enum",
			tag:         "column:id;type:uuid;primary_key",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ParseStormTag(tt.tag, false)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStormTagParser_ToSnakeCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple camelCase",
			input:    "firstName",
			expected: "first_name",
		},
		{
			name:     "PascalCase",
			input:    "FirstName",
			expected: "first_name",
		},
		{
			name:     "already snake_case",
			input:    "first_name",
			expected: "first_name",
		},
		{
			name:     "single word",
			input:    "user",
			expected: "user",
		},
		{
			name:     "multiple words",
			input:    "UserAccountDetails",
			expected: "user_account_details",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStormTagParser_ToTableLevelAttributes(t *testing.T) {
	parser := NewStormTagParser()

	parsed, err := parser.ParseStormTag("column:id;type:uuid;primary_key;index:idx_user_id", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	attrs := parsed.ToTableLevelAttributes()

	if len(attrs) != 1 {
		t.Errorf("expected 1 table-level attribute, got %d", len(attrs))
	}

	if attrs["index"] != "idx_user_id" {
		t.Errorf("expected index attribute 'idx_user_id', got '%s'", attrs["index"])
	}
}
