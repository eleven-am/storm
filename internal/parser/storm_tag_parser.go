package parser

import (
	"fmt"
	"strings"
)

// StormTagParser handles parsing of unified storm tags
type StormTagParser struct {
	// Cache for parsed tags
	tagCache map[string]*ParsedStormTag
}

// ParsedStormTag represents a parsed storm tag that can contain both column and relationship attributes
type ParsedStormTag struct {
	// Column attributes (from previous dbdef)
	Type       string
	PrimaryKey bool
	NotNull    bool
	Unique     bool
	Default    string
	Check      string
	ForeignKey string
	OnDelete   string
	OnUpdate   string
	Constraint string
	Prev       string
	Enum       []string
	ArrayType  string

	// Relationship attributes (from previous orm)
	RelationType       string   // "belongs_to", "has_one", "has_many", "has_many_through"
	RelationTarget     string   // Target model/table name
	RelationForeignKey string   // Foreign key column
	RelationSourceKey  string   // Source key column (for has_many)
	RelationTargetKey  string   // Target key column (for belongs_to)
	JoinTable          string   // Join table for has_many_through
	SourceFK           string   // Source FK in join table
	TargetFK           string   // Target FK in join table
	Conditions         []string // Additional conditions
	OrderBy            string   // Default ordering
	Dependent          string   // Dependent action (destroy, delete, nullify)
	Inverse            string   // Inverse relationship name
	Polymorphic        string   // Polymorphic association
	Through            string   // Through association
	Validate           bool     // Whether to validate association
	Autosave           bool     // Whether to autosave association
	Counter            string   // Counter cache column

	// Special attributes
	Column    string // Database column name (replaces db tag for relationships)
	Ignore    bool   // Exclude from database operations
	Computed  string // Computed/derived field
	Immutable bool   // Immutable field (create-only)

	// Table-level attributes (for _ struct{} fields)
	Table         string   // Table name
	Indexes       []string // Index definitions
	UniqueIndexes []string // Unique constraints

	// Raw tag value
	Raw string

	// Context - whether this is a column or relationship field
	IsRelationship bool
}

func NewStormTagParser() *StormTagParser {
	return &StormTagParser{
		tagCache: make(map[string]*ParsedStormTag),
	}
}

func (p *StormTagParser) ParseStormTag(tag string, isRelationshipField bool) (*ParsedStormTag, error) {
	if tag == "" {
		return nil, fmt.Errorf("empty storm tag")
	}

	cacheKey := fmt.Sprintf("%s:%t", tag, isRelationshipField)
	if cached, exists := p.tagCache[cacheKey]; exists {
		return cached, nil
	}

	parsed := &ParsedStormTag{
		Raw:            tag,
		IsRelationship: isRelationshipField,
		Validate:       true,
	}

	attributes := strings.Split(tag, ";")
	for _, attr := range attributes {
		attr = strings.TrimSpace(attr)
		if attr == "" {
			continue
		}

		if err := p.parseAttribute(attr, parsed); err != nil {
			return nil, fmt.Errorf("failed to parse attribute '%s': %w", attr, err)
		}
	}

	if err := p.validateAndSetDefaults(parsed); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	p.tagCache[cacheKey] = parsed
	return parsed, nil
}

func (p *StormTagParser) parseAttribute(attr string, parsed *ParsedStormTag) error {
	if !strings.Contains(attr, ":") {
		return p.parseFlagAttribute(attr, parsed)
	}

	parts := strings.SplitN(attr, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid attribute format: %s", attr)
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	return p.parseKeyValueAttribute(key, value, parsed)
}

func (p *StormTagParser) parseFlagAttribute(flag string, parsed *ParsedStormTag) error {
	switch flag {
	case "primary_key":
		parsed.PrimaryKey = true
	case "not_null":
		parsed.NotNull = true
	case "unique":
		parsed.Unique = true
	case "ignore":
		parsed.Ignore = true
	case "immutable":
		parsed.Immutable = true
	case "validate":
		parsed.Validate = true
	case "no_validate":
		parsed.Validate = false
	case "autosave":
		parsed.Autosave = true
	case "no_autosave":
		parsed.Autosave = false
	default:
		return fmt.Errorf("unknown flag attribute: %s", flag)
	}
	return nil
}

func (p *StormTagParser) parseKeyValueAttribute(key, value string, parsed *ParsedStormTag) error {
	if value == "" {
		return fmt.Errorf("attribute %s cannot have empty value", key)
	}

	switch key {
	case "column":
		parsed.Column = value
	case "type":
		parsed.Type = value
	case "default":
		parsed.Default = value
	case "check":
		parsed.Check = value
	case "foreign_key":
		parsed.ForeignKey = value
		parsed.RelationForeignKey = value
	case "on_delete":
		parsed.OnDelete = value
	case "on_update":
		parsed.OnUpdate = value
	case "constraint":
		parsed.Constraint = value
	case "prev":
		parsed.Prev = value
	case "enum":
		parsed.Enum = strings.Split(value, ",")
		for i, v := range parsed.Enum {
			parsed.Enum[i] = strings.TrimSpace(v)
		}
	case "array_type":
		parsed.ArrayType = value
	case "computed":
		parsed.Computed = value

	case "table":
		parsed.Table = value
	case "index":
		parsed.Indexes = append(parsed.Indexes, value)
	case "unique":
		parsed.UniqueIndexes = append(parsed.UniqueIndexes, value)

	case "relation":
		return p.parseRelationAttribute(value, parsed)
	case "source_key":
		parsed.RelationSourceKey = value
	case "target_key":
		parsed.RelationTargetKey = value
	case "join_table":
		parsed.JoinTable = value
	case "source_fk":
		parsed.SourceFK = value
	case "target_fk":
		parsed.TargetFK = value
	case "order_by":
		parsed.OrderBy = value
	case "dependent":
		if !isValidDependentAction(value) {
			return fmt.Errorf("invalid dependent action: %s", value)
		}
		parsed.Dependent = value
	case "inverse":
		parsed.Inverse = value
	case "polymorphic":
		parsed.Polymorphic = value
	case "through":
		parsed.Through = value
	case "counter":
		parsed.Counter = value
	case "conditions":
		parsed.Conditions = strings.Split(value, ",")
		for i, v := range parsed.Conditions {
			parsed.Conditions[i] = strings.TrimSpace(v)
		}

	default:
		return fmt.Errorf("unknown attribute: %s", key)
	}

	return nil
}

func (p *StormTagParser) parseRelationAttribute(value string, parsed *ParsedStormTag) error {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid relation format, expected 'type:target', got: %s", value)
	}

	relType := strings.TrimSpace(parts[0])
	target := strings.TrimSpace(parts[1])

	switch relType {
	case "belongs_to", "has_one", "has_many", "has_many_through":
		parsed.RelationType = relType
	default:
		return fmt.Errorf("invalid relationship type: %s", relType)
	}

	if target == "" {
		return fmt.Errorf("relation target cannot be empty")
	}

	parsed.RelationTarget = target
	return nil
}

func (p *StormTagParser) validateAndSetDefaults(parsed *ParsedStormTag) error {
	if parsed.IsRelationship {
		return p.validateRelationship(parsed)
	}
	return p.validateColumn(parsed)
}

func (p *StormTagParser) validateRelationship(parsed *ParsedStormTag) error {
	if parsed.RelationType == "" || parsed.RelationTarget == "" {
		return fmt.Errorf("relationships must specify relation:type:target")
	}

	switch parsed.RelationType {
	case "belongs_to":
		if parsed.RelationForeignKey == "" {
			parsed.RelationForeignKey = toSnakeCase(parsed.RelationTarget) + "_id"
		}
		if parsed.RelationTargetKey == "" {
			parsed.RelationTargetKey = "id"
		}

	case "has_one", "has_many":
		if parsed.RelationForeignKey == "" {
			return fmt.Errorf("foreign_key is required for %s relationships", parsed.RelationType)
		}
		if parsed.RelationSourceKey == "" {
			parsed.RelationSourceKey = "id"
		}

	case "has_many_through":
		if parsed.JoinTable == "" {
			return fmt.Errorf("join_table is required for has_many_through relationships")
		}
		if parsed.SourceFK == "" {
			return fmt.Errorf("source_fk is required for has_many_through relationships")
		}
		if parsed.TargetFK == "" {
			return fmt.Errorf("target_fk is required for has_many_through relationships")
		}
		if parsed.RelationSourceKey == "" {
			parsed.RelationSourceKey = "id"
		}
		if parsed.RelationTargetKey == "" {
			parsed.RelationTargetKey = "id"
		}
	}

	if parsed.Type != "" || parsed.PrimaryKey || parsed.NotNull || parsed.Unique {
		return fmt.Errorf("relationship fields cannot have column attributes (type, primary_key, not_null, unique)")
	}

	return nil
}

func (p *StormTagParser) validateColumn(parsed *ParsedStormTag) error {
	if parsed.RelationType != "" || parsed.RelationTarget != "" {
		return fmt.Errorf("column fields cannot have relationship attributes (relation)")
	}

	if parsed.Type != "" {
		if err := p.validateType(parsed.Type); err != nil {
			return fmt.Errorf("invalid type '%s': %w", parsed.Type, err)
		}
	}

	if parsed.Default != "" {
		if err := p.validateDefault(parsed.Default); err != nil {
			return fmt.Errorf("invalid default '%s': %w", parsed.Default, err)
		}
	}

	if parsed.ForeignKey != "" {
		if err := p.validateForeignKey(parsed.ForeignKey); err != nil {
			return fmt.Errorf("invalid foreign key '%s': %w", parsed.ForeignKey, err)
		}
	}

	if parsed.Check != "" {
		if err := p.validateCheck(parsed.Check); err != nil {
			return fmt.Errorf("invalid check constraint '%s': %w", parsed.Check, err)
		}
	}

	if len(parsed.Enum) > 0 {
		if err := p.validateEnum(parsed.Enum); err != nil {
			return fmt.Errorf("invalid enum: %w", err)
		}
	}

	return nil
}

func (p *StormTagParser) validateType(typeValue string) error {
	tagParser := NewTagParser()
	return tagParser.validateType(typeValue)
}

func (p *StormTagParser) validateDefault(defaultValue string) error {
	tagParser := NewTagParser()
	return tagParser.validateDefault(defaultValue)
}

func (p *StormTagParser) validateForeignKey(fkValue string) error {
	tagParser := NewTagParser()
	return tagParser.validateForeignKey(fkValue)
}

func (p *StormTagParser) validateCheck(checkValue string) error {
	tagParser := NewTagParser()
	return tagParser.validateCheck(checkValue)
}

func (p *StormTagParser) validateEnum(enumValues []string) error {
	enumString := strings.Join(enumValues, ",")
	tagParser := NewTagParser()
	return tagParser.validateEnum(enumString)
}

func isValidDependentAction(action string) bool {
	validActions := []string{"destroy", "delete", "nullify", "restrict"}
	for _, valid := range validActions {
		if action == valid {
			return true
		}
	}
	return false
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		if r >= 'A' && r <= 'Z' {
			result.WriteRune(r - 'A' + 'a')
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func (p *ParsedStormTag) ToDBDefAttributes() map[string]string {
	attrs := make(map[string]string)

	if p.Type != "" {
		attrs["type"] = p.Type
	}
	if p.PrimaryKey {
		attrs["primary_key"] = ""
	}
	if p.NotNull {
		attrs["not_null"] = ""
	}
	if p.Unique {
		attrs["unique"] = ""
	}
	if p.Default != "" {
		attrs["default"] = p.Default
	}
	if p.Check != "" {
		attrs["check"] = p.Check
	}
	if p.ForeignKey != "" {
		attrs["foreign_key"] = p.ForeignKey
	}
	if p.OnDelete != "" {
		attrs["on_delete"] = p.OnDelete
	}
	if p.OnUpdate != "" {
		attrs["on_update"] = p.OnUpdate
	}
	if p.Constraint != "" {
		attrs["constraint"] = p.Constraint
	}
	if p.Prev != "" {
		attrs["prev"] = p.Prev
	}
	if len(p.Enum) > 0 {
		attrs["enum"] = strings.Join(p.Enum, ",")
	}
	if p.ArrayType != "" {
		attrs["array_type"] = p.ArrayType
	}

	return attrs
}

func (p *ParsedStormTag) ToTableLevelAttributes() map[string]string {
	attrs := make(map[string]string)

	if p.Table != "" {
		attrs["table"] = p.Table
	}
	for _, index := range p.Indexes {
		if existing, exists := attrs["index"]; exists {
			attrs["index"] = existing + ";" + index
		} else {
			attrs["index"] = index
		}
	}
	for _, unique := range p.UniqueIndexes {
		if existing, exists := attrs["unique"]; exists {
			attrs["unique"] = existing + ";" + unique
		} else {
			attrs["unique"] = unique
		}
	}

	return attrs
}
