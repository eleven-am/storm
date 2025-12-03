package orm_generator

import (
	"fmt"
	"strings"
	"testing"
)

func TestValidateModel(t *testing.T) {
	tests := []struct {
		name    string
		model   *ModelMetadata
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid model",
			model: &ModelMetadata{
				Name:      "User",
				TableName: "users",
				Columns: []FieldMetadata{
					{Name: "ID", DBName: "id", Type: "int", IsPrimaryKey: true},
					{Name: "Name", DBName: "name", Type: "string"},
				},
				PrimaryKeys: []string{"id"},
			},
			wantErr: false,
		},
		{
			name: "model without primary key",
			model: &ModelMetadata{
				Name:      "User",
				TableName: "users",
				Columns: []FieldMetadata{
					{Name: "Name", DBName: "name", Type: "string"},
				},
				PrimaryKeys: []string{},
			},
			wantErr: true,
			errMsg:  "no primary key",
		},
		{
			name: "model with empty name",
			model: &ModelMetadata{
				Name:      "",
				TableName: "users",
				Columns: []FieldMetadata{
					{Name: "ID", DBName: "id", Type: "int", IsPrimaryKey: true},
				},
				PrimaryKeys: []string{"id"},
			},
			wantErr: true,
			errMsg:  "empty model name",
		},
		{
			name: "model with duplicate columns",
			model: &ModelMetadata{
				Name:      "User",
				TableName: "users",
				Columns: []FieldMetadata{
					{Name: "ID", DBName: "id", Type: "int", IsPrimaryKey: true},
					{Name: "Email", DBName: "email", Type: "string"},
					{Name: "Email2", DBName: "email", Type: "string"},
				},
				PrimaryKeys: []string{"id"},
			},
			wantErr: true,
			errMsg:  "duplicate column",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateModelMetadata(tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateModelMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

func TestValidateRelationship(t *testing.T) {

	userModel := &ModelMetadata{
		Name:      "User",
		TableName: "users",
		Columns: []FieldMetadata{
			{Name: "ID", DBName: "id", Type: "int", IsPrimaryKey: true},
			{Name: "Name", DBName: "name", Type: "string"},
		},
		PrimaryKeys: []string{"id"},
	}

	postModel := &ModelMetadata{
		Name:      "Post",
		TableName: "posts",
		Columns: []FieldMetadata{
			{Name: "ID", DBName: "id", Type: "int", IsPrimaryKey: true},
			{Name: "UserID", DBName: "user_id", Type: "int"},
			{Name: "Title", DBName: "title", Type: "string"},
		},
		PrimaryKeys: []string{"id"},
	}

	models := map[string]*ModelMetadata{
		"User": userModel,
		"Post": postModel,
	}

	tests := []struct {
		name         string
		relationship FieldMetadata
		sourceModel  *ModelMetadata
		wantErr      bool
		errMsg       string
	}{
		{
			name: "valid belongs_to",
			relationship: FieldMetadata{
				Name: "User",
				Relationship: &ParsedORMTag{
					Type:       "belongs_to",
					Target:     "User",
					ForeignKey: "user_id",
					TargetKey:  "id",
				},
			},
			sourceModel: postModel,
			wantErr:     false,
		},
		{
			name: "belongs_to with missing foreign key",
			relationship: FieldMetadata{
				Name: "User",
				Relationship: &ParsedORMTag{
					Type:       "belongs_to",
					Target:     "User",
					ForeignKey: "missing_id",
					TargetKey:  "id",
				},
			},
			sourceModel: postModel,
			wantErr:     true,
			errMsg:      "foreign key column missing_id not found",
		},
		{
			name: "has_many with valid foreign key",
			relationship: FieldMetadata{
				Name: "Posts",
				Relationship: &ParsedORMTag{
					Type:       "has_many",
					Target:     "Post",
					ForeignKey: "user_id",
					SourceKey:  "id",
				},
			},
			sourceModel: userModel,
			wantErr:     false,
		},
		{
			name: "relationship to non-existent model",
			relationship: FieldMetadata{
				Name: "Comments",
				Relationship: &ParsedORMTag{
					Type:       "has_many",
					Target:     "Comment",
					ForeignKey: "post_id",
					SourceKey:  "id",
				},
			},
			sourceModel: postModel,
			wantErr:     true,
			errMsg:      "target model Comment not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRelationshipMetadata(tt.sourceModel, tt.relationship, models)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRelationshipMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

func TestValidateFieldMetadata(t *testing.T) {
	tests := []struct {
		name    string
		field   FieldMetadata
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid field",
			field: FieldMetadata{
				Name:   "Email",
				DBName: "email",
				Type:   "string",
			},
			wantErr: false,
		},
		{
			name: "field with empty name",
			field: FieldMetadata{
				Name:   "",
				DBName: "email",
				Type:   "string",
			},
			wantErr: true,
			errMsg:  "empty field name",
		},
		{
			name: "field with empty db name",
			field: FieldMetadata{
				Name:   "Email",
				DBName: "",
				Type:   "string",
			},
			wantErr: true,
			errMsg:  "empty db name",
		},
		{
			name: "field with empty type",
			field: FieldMetadata{
				Name:   "Email",
				DBName: "email",
				Type:   "",
			},
			wantErr: true,
			errMsg:  "empty type",
		},
		{
			name: "field with invalid db name",
			field: FieldMetadata{
				Name:   "Email",
				DBName: "user-email",
				Type:   "string",
			},
			wantErr: true,
			errMsg:  "invalid db name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFieldMetadata(tt.field)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFieldMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

// Helper validation functions that would be in the actual implementation

func validateModelMetadata(model *ModelMetadata) error {
	if model.Name == "" {
		return errorf("empty model name")
	}
	if len(model.PrimaryKeys) == 0 {
		return errorf("model %s has no primary key", model.Name)
	}

	columnNames := make(map[string]bool)
	for _, col := range model.Columns {
		if columnNames[col.DBName] {
			return errorf("duplicate column %s in model %s", col.DBName, model.Name)
		}
		columnNames[col.DBName] = true
	}

	return nil
}

func validateRelationshipMetadata(sourceModel *ModelMetadata, rel FieldMetadata, models map[string]*ModelMetadata) error {
	if rel.Relationship == nil {
		return errorf("nil relationship metadata")
	}

	targetModel, exists := models[rel.Relationship.Target]
	if !exists {
		return errorf("target model %s not found", rel.Relationship.Target)
	}

	switch rel.Relationship.Type {
	case "belongs_to":
		if !hasColumn(sourceModel, rel.Relationship.ForeignKey) {
			return errorf("foreign key column %s not found in model %s", rel.Relationship.ForeignKey, sourceModel.Name)
		}
		if !hasColumn(targetModel, rel.Relationship.TargetKey) {
			return errorf("target key column %s not found in model %s", rel.Relationship.TargetKey, targetModel.Name)
		}

	case "has_one", "has_many":
		if !hasColumn(sourceModel, rel.Relationship.SourceKey) {
			return errorf("source key column %s not found in model %s", rel.Relationship.SourceKey, sourceModel.Name)
		}
		if !hasColumn(targetModel, rel.Relationship.ForeignKey) {
			return errorf("foreign key column %s not found in model %s", rel.Relationship.ForeignKey, targetModel.Name)
		}
	}

	return nil
}

func validateFieldMetadata(field FieldMetadata) error {
	if field.Name == "" {
		return errorf("empty field name")
	}
	if field.DBName == "" {
		return errorf("empty db name for field %s", field.Name)
	}
	if field.Type == "" {
		return errorf("empty type for field %s", field.Name)
	}

	for _, r := range field.DBName {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_') {
			return errorf("invalid db name %s for field %s", field.DBName, field.Name)
		}
	}

	return nil
}

// Helper functions

func hasColumn(model *ModelMetadata, columnName string) bool {
	for _, col := range model.Columns {
		if col.DBName == columnName {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstringInValidator(s, substr)
}

// Renamed to avoid conflict with codegen_test.go
func findSubstringInValidator(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func errorf(format string, args ...interface{}) error {
	return &ValidationError{Message: sprintf(format, args...)}
}

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func sprintf(format string, args ...interface{}) string {
	if len(args) == 0 {
		return format
	}
	result := format
	for i, arg := range args {
		if i == 0 {
			result = strings.Replace(result, "%s", fmt.Sprintf("%v", arg), 1)
		} else {
			result = strings.Replace(result, "%s", fmt.Sprintf("%v", arg), 1)
		}
	}
	return result
}
