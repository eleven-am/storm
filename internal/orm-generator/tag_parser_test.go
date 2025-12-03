package orm_generator

import (
	"reflect"
	"testing"
)

func TestParseORMTag(t *testing.T) {
	parser := NewORMTagParser()

	tests := []struct {
		name    string
		tag     string
		want    *ParsedORMTag
		wantErr bool
	}{
		{
			name: "belongs_to with all options",
			tag:  "belongs_to:User,foreign_key:user_id,target_key:id",
			want: &ParsedORMTag{
				Type:       "belongs_to",
				Target:     "User",
				ForeignKey: "user_id",
				TargetKey:  "id",
			},
			wantErr: false,
		},
		{
			name: "has_many with foreign key",
			tag:  "has_many:Post,foreign_key:user_id",
			want: &ParsedORMTag{
				Type:       "has_many",
				Target:     "Post",
				ForeignKey: "user_id",
				SourceKey:  "id",
			},
			wantErr: false,
		},
		{
			name: "has_one simple",
			tag:  "has_one:Profile,foreign_key:user_id",
			want: &ParsedORMTag{
				Type:       "has_one",
				Target:     "Profile",
				ForeignKey: "user_id",
				SourceKey:  "id",
			},
			wantErr: false,
		},
		{
			name: "has_many_through with all options",
			tag:  "has_many_through:Tag,join_table:post_tags,source_fk:post_id,target_fk:tag_id",
			want: &ParsedORMTag{
				Type:      "has_many_through",
				Target:    "Tag",
				SourceKey: "id",
				TargetKey: "id",
				JoinTable: "post_tags",
				SourceFK:  "post_id",
				TargetFK:  "tag_id",
				Validate:  true,
				Raw:       "has_many_through:Tag,join_table:post_tags,source_fk:post_id,target_fk:tag_id",
			},
			wantErr: false,
		},
		{
			name:    "empty tag",
			tag:     "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			tag:     "invalid",
			wantErr: true,
		},
		{
			name:    "unknown relationship type",
			tag:     "unknown_type:Model",
			wantErr: true,
		},
		{
			name:    "missing target",
			tag:     "belongs_to:",
			wantErr: true,
		},
		{
			name: "extra whitespace",
			tag:  "belongs_to: User , foreign_key: user_id ",
			want: &ParsedORMTag{
				Type:       "belongs_to",
				Target:     "User",
				ForeignKey: "user_id",
				TargetKey:  "id",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.ParseORMTag(tt.tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseORMTag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !relationshipsEqual(got, tt.want) {
				t.Errorf("ParseORMTag() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseModel(t *testing.T) {
	type Post struct {
		ID     int    `db:"id" dbdef:"primary_key"`
		UserID int    `db:"user_id"`
		Title  string `db:"title"`
	}

	type TestModel struct {
		ID      int    `db:"id" dbdef:"primary_key"`
		Name    string `db:"name" dbdef:"not_null"`
		Email   string `db:"email" dbdef:"unique"`
		Posts   []Post `db:"-" orm:"has_many:Post,foreign_key:user_id"`
		Profile *Post  `db:"-" orm:"has_one:Profile,foreign_key:user_id"`
		Ignored string
	}

	parser := NewORMTagParser()
	modelType := reflect.TypeOf(TestModel{})
	metadata := &ModelMetadata{
		Name:          modelType.Name(),
		Fields:        make([]FieldMetadata, 0),
		Relationships: make([]FieldMetadata, 0),
		Columns:       make([]FieldMetadata, 0),
		PrimaryKeys:   make([]string, 0),
	}

	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		fieldMeta, err := parser.parseField(field)
		if err != nil {
			t.Fatalf("parseField() error = %v", err)
		}

		metadata.Fields = append(metadata.Fields, fieldMeta)

		if fieldMeta.Relationship != nil {
			metadata.Relationships = append(metadata.Relationships, fieldMeta)
		} else if fieldMeta.DBName != "" {
			metadata.Columns = append(metadata.Columns, fieldMeta)
			if fieldMeta.IsPrimaryKey {
				metadata.PrimaryKeys = append(metadata.PrimaryKeys, fieldMeta.DBName)
			}
		}
	}

	if metadata.Name != "TestModel" {
		t.Errorf("expected model name TestModel, got %s", metadata.Name)
	}

	expectedColumns := 4
	actualColumns := len(metadata.Columns)
	if actualColumns != expectedColumns {
		t.Logf("Columns found: %d", actualColumns)
		for _, col := range metadata.Columns {
			t.Logf("Column: %s (%s)", col.Name, col.DBName)
		}
		t.Errorf("expected %d columns, got %d", expectedColumns, actualColumns)
	}

	if len(metadata.PrimaryKeys) != 1 || metadata.PrimaryKeys[0] != "id" {
		t.Errorf("expected primary key [id], got %v", metadata.PrimaryKeys)
	}

	expectedRelationships := 2
	if len(metadata.Relationships) != expectedRelationships {
		t.Errorf("expected %d relationships, got %d", expectedRelationships, len(metadata.Relationships))
	}

	for _, col := range metadata.Columns {
		switch col.Name {
		case "ID":
			if !col.IsPrimaryKey {
				t.Error("ID should be primary key")
			}
		case "Email":
			if !col.IsUnique {
				t.Error("Email should be unique")
			}
		}
	}
}

func TestDefaultRelationshipValues(t *testing.T) {
	parser := NewORMTagParser()

	tests := []struct {
		name     string
		tag      string
		validate func(t *testing.T, rel *ParsedORMTag)
	}{
		{
			name: "belongs_to defaults",
			tag:  "belongs_to:User",
			validate: func(t *testing.T, rel *ParsedORMTag) {
				if rel.ForeignKey != "user_id" {
					t.Errorf("expected foreign_key user_id, got %s", rel.ForeignKey)
				}
				if rel.TargetKey != "id" {
					t.Errorf("expected target_key id, got %s", rel.TargetKey)
				}
			},
		},
		{
			name: "has_many defaults",
			tag:  "has_many:Post,foreign_key:user_id",
			validate: func(t *testing.T, rel *ParsedORMTag) {
				if rel.SourceKey != "id" {
					t.Errorf("expected source_key id, got %s", rel.SourceKey)
				}
				if rel.ForeignKey != "user_id" {
					t.Errorf("expected foreign_key user_id, got %s", rel.ForeignKey)
				}
			},
		},
		{
			name: "has_many_through defaults",
			tag:  "has_many_through:Tag,join_table:post_tags,source_fk:post_id,target_fk:tag_id",
			validate: func(t *testing.T, rel *ParsedORMTag) {
				if rel.JoinTable != "post_tags" {
					t.Errorf("expected join_table post_tags, got %s", rel.JoinTable)
				}
				if rel.SourceFK != "post_id" {
					t.Errorf("expected source_fk post_id, got %s", rel.SourceFK)
				}
				if rel.TargetFK != "tag_id" {
					t.Errorf("expected target_fk tag_id, got %s", rel.TargetFK)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rel, err := parser.ParseORMTag(tt.tag)
			if err != nil {
				t.Fatalf("ParseORMTag() error = %v", err)
			}
			tt.validate(t, rel)
		})
	}
}

func TestParseComplexModel(t *testing.T) {
	type Category struct {
		ID   int    `db:"id" dbdef:"primary_key"`
		Name string `db:"name"`
	}

	type Tag struct {
		ID   int    `db:"id" dbdef:"primary_key"`
		Name string `db:"name"`
	}

	type Article struct {
		ID         int       `db:"id" dbdef:"primary_key;auto_increment"`
		Title      string    `db:"title" dbdef:"not_null"`
		Content    string    `db:"content" dbdef:"type:text"`
		CategoryID int       `db:"category_id" dbdef:"not_null"`
		Category   *Category `db:"-" orm:"belongs_to:Category,foreign_key:category_id"`
		Tags       []Tag     `db:"-" orm:"has_many_through:Tag,join_table:article_tags,source_fk:article_id,target_fk:tag_id"`
	}

	parser := NewORMTagParser()
	modelType := reflect.TypeOf(Article{})
	metadata := &ModelMetadata{
		Name:          modelType.Name(),
		Fields:        make([]FieldMetadata, 0),
		Relationships: make([]FieldMetadata, 0),
		Columns:       make([]FieldMetadata, 0),
		PrimaryKeys:   make([]string, 0),
	}

	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		fieldMeta, err := parser.parseField(field)
		if err != nil {
			t.Fatalf("parseField() error = %v", err)
		}

		metadata.Fields = append(metadata.Fields, fieldMeta)

		if fieldMeta.Relationship != nil {
			metadata.Relationships = append(metadata.Relationships, fieldMeta)
		} else if fieldMeta.DBName != "" {
			metadata.Columns = append(metadata.Columns, fieldMeta)
			if fieldMeta.IsPrimaryKey {
				metadata.PrimaryKeys = append(metadata.PrimaryKeys, fieldMeta.DBName)
			}
		}
	}

	if len(metadata.Relationships) != 2 {
		t.Errorf("expected 2 relationships, got %d", len(metadata.Relationships))
	}

	var categoryRel *FieldMetadata
	for _, rel := range metadata.Relationships {
		if rel.Name == "Category" {
			categoryRel = &rel
			break
		}
	}
	if categoryRel == nil {
		t.Fatal("Category relationship not found")
	}
	if categoryRel.Relationship.Type != "belongs_to" {
		t.Errorf("expected belongs_to relationship, got %s", categoryRel.Relationship.Type)
	}

	var tagsRel *FieldMetadata
	for _, rel := range metadata.Relationships {
		if rel.Name == "Tags" {
			tagsRel = &rel
			break
		}
	}
	if tagsRel == nil {
		t.Fatal("Tags relationship not found")
	}
	if tagsRel.Relationship.Type != "has_many_through" {
		t.Errorf("expected has_many_through relationship, got %s", tagsRel.Relationship.Type)
	}
	if tagsRel.Relationship.JoinTable != "article_tags" {
		t.Errorf("expected join_table article_tags, got %s", tagsRel.Relationship.JoinTable)
	}
}

// Helper function to compare relationships
func relationshipsEqual(a, b *ParsedORMTag) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Type == b.Type &&
		a.Target == b.Target &&
		a.ForeignKey == b.ForeignKey &&
		a.SourceKey == b.SourceKey &&
		a.TargetKey == b.TargetKey &&
		a.JoinTable == b.JoinTable &&
		a.SourceFK == b.SourceFK &&
		a.TargetFK == b.TargetFK
}
