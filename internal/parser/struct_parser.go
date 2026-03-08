package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strings"
)

// FieldDefinition represents a struct field with database metadata
type FieldDefinition struct {
	Name           string
	DBName         string
	Type           string
	IsPointer      bool
	IsArray        bool
	IsRelationship bool
	DBDef          map[string]string
	DBTag          string
	DBDefTag       string // Deprecated: use StormTag instead
	JSONTag        string
	ORMTag         string // Deprecated: use StormTag instead
	StormTag       string // New unified tag
}

// TableDefinition represents a complete table structure
type TableDefinition struct {
	StructName string
	TableName  string
	Fields     []FieldDefinition
	TableLevel map[string]string
}

// StructParser handles parsing Go struct definitions
type StructParser struct {
	fileSet        *token.FileSet
	tagParser      *TagParser
	stormTagParser *StormTagParser
}

func NewStructParser() *StructParser {
	return &StructParser{
		fileSet:        token.NewFileSet(),
		tagParser:      NewTagParser(),
		stormTagParser: NewStormTagParser(),
	}
}

func (p *StructParser) ParseDirectory(dir string) ([]TableDefinition, error) {
	pattern := filepath.Join(dir, "*.go")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob directory %s: %w", dir, err)
	}

	var allTables []TableDefinition

	for _, file := range matches {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}

		tables, err := p.ParseFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to parse file %s: %w", file, err)
		}

		allTables = append(allTables, tables...)
	}

	return allTables, nil
}

func (p *StructParser) ParseFile(filename string) ([]TableDefinition, error) {
	src, err := parser.ParseFile(p.fileSet, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	var tables []TableDefinition

	ast.Inspect(src, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.TypeSpec:
			if structType, ok := node.Type.(*ast.StructType); ok {
				table, err := p.parseStruct(node.Name.Name, structType)
				if err != nil {
					fmt.Printf("Warning: failed to parse struct %s: %v\n", node.Name.Name, err)
					return true
				}

				if p.isDatabaseStruct(table) {
					tables = append(tables, table)
				}
			}
		}
		return true
	})

	return tables, nil
}

func (p *StructParser) parseStruct(structName string, structType *ast.StructType) (TableDefinition, error) {
	table := TableDefinition{
		StructName: structName,
		TableName:  p.deriveTableName(structName),
		Fields:     make([]FieldDefinition, 0),
		TableLevel: make(map[string]string),
	}

	for _, field := range structType.Fields.List {
		fieldDefs, tableLevelAttrs, err := p.parseField(field)
		if err != nil {
			return table, fmt.Errorf("failed to parse field: %w", err)
		}

		table.Fields = append(table.Fields, fieldDefs...)

		for k, v := range tableLevelAttrs {
			table.TableLevel[k] = v
		}
	}

	if tableName, exists := table.TableLevel["table"]; exists {
		table.TableName = tableName
	}

	return table, nil
}

func (p *StructParser) parseField(field *ast.Field) ([]FieldDefinition, map[string]string, error) {
	var fields []FieldDefinition
	tableLevelAttrs := make(map[string]string)

	if len(field.Names) == 0 {
		if field.Tag != nil {
			tagValue := strings.Trim(field.Tag.Value, "`")
			stormTag := p.extractTag(tagValue, "storm")
			if stormTag != "" {
				parsed, err := p.stormTagParser.ParseStormTag(stormTag, false)
				if err == nil {
					attrs := parsed.ToTableLevelAttributes()
					for k, v := range attrs {
						tableLevelAttrs[k] = v
					}
				}
			} else {
				dbdefTag := p.extractTag(tagValue, "dbdef")
				if dbdefTag != "" {
					attrs := p.tagParser.ParseDBDefTag(dbdefTag)
					for k, v := range attrs {
						tableLevelAttrs[k] = v
					}
				}
			}
		}
		return fields, tableLevelAttrs, nil
	}

	for _, name := range field.Names {
		if !ast.IsExported(name.Name) && name.Name != "_" {
			continue
		}

		if name.Name == "_" && field.Tag != nil {
			tagValue := strings.Trim(field.Tag.Value, "`")
			stormTag := p.extractTag(tagValue, "storm")
			if stormTag != "" {
				parsed, err := p.stormTagParser.ParseStormTag(stormTag, false)
				if err == nil {
					attrs := parsed.ToTableLevelAttributes()
					for k, v := range attrs {
						tableLevelAttrs[k] = v
					}
				}
			} else {
				dbdefTag := p.extractTag(tagValue, "dbdef")
				if dbdefTag != "" {
					attrs := p.tagParser.ParseDBDefTag(dbdefTag)
					for k, v := range attrs {
						tableLevelAttrs[k] = v
					}
				}
			}
			continue
		}

		fieldDef := FieldDefinition{
			Name: name.Name,
		}

		fieldType, isPointer, isArray := p.parseFieldType(field.Type)
		fieldDef.Type = fieldType
		fieldDef.IsPointer = isPointer
		fieldDef.IsArray = isArray

		if field.Tag != nil {
			tagValue := strings.Trim(field.Tag.Value, "`")

			fieldDef.DBTag = p.extractTag(tagValue, "db")
			fieldDef.DBDefTag = p.extractTag(tagValue, "dbdef")
			fieldDef.JSONTag = p.extractTag(tagValue, "json")
			fieldDef.ORMTag = p.extractTag(tagValue, "orm")
			fieldDef.StormTag = p.extractTag(tagValue, "storm")

			if fieldDef.DBTag != "" {
				fieldDef.DBName = fieldDef.DBTag
			} else if fieldDef.StormTag != "" {
				isRelationshipField := fieldDef.IsArray || fieldDef.IsPointer
				parsed, err := p.stormTagParser.ParseStormTag(fieldDef.StormTag, isRelationshipField)
				if err == nil && parsed.Column != "" {
					fieldDef.DBName = parsed.Column
				} else {
					fieldDef.DBName = p.toSnakeCase(fieldDef.Name)
				}
			} else {
				fieldDef.DBName = p.toSnakeCase(fieldDef.Name)
			}

			if fieldDef.StormTag != "" {
				isRelationshipField := fieldDef.IsArray || fieldDef.IsPointer
				parsed, err := p.stormTagParser.ParseStormTag(fieldDef.StormTag, isRelationshipField)
				if err == nil {
					fieldDef.IsRelationship = parsed.IsRelationship
					if !parsed.IsRelationship {
						fieldDef.DBDef = parsed.ToDBDefAttributes()
					} else {
						fieldDef.DBDef = make(map[string]string)
					}
				} else {
					fieldDef.DBDef = make(map[string]string)
				}
			} else if fieldDef.DBDefTag != "" {
				fieldDef.DBDef = p.tagParser.ParseDBDefTag(fieldDef.DBDefTag)
			} else {
				fieldDef.DBDef = make(map[string]string)
			}
		} else {
			fieldDef.DBName = p.toSnakeCase(fieldDef.Name)
			fieldDef.DBDef = make(map[string]string)
		}

		fields = append(fields, fieldDef)
	}

	return fields, tableLevelAttrs, nil
}

func (p *StructParser) parseFieldType(expr ast.Expr) (string, bool, bool) {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name, false, false

	case *ast.StarExpr:
		innerType, _, isArray := p.parseFieldType(t.X)
		return innerType, true, isArray

	case *ast.ArrayType:
		innerType, isPointer, _ := p.parseFieldType(t.Elt)
		return innerType, isPointer, true

	case *ast.SelectorExpr:
		pkg := p.exprToString(t.X)
		return pkg + "." + t.Sel.Name, false, false

	case *ast.IndexExpr:
		baseType := p.exprToString(t.X)
		indexType := p.exprToString(t.Index)
		return baseType + "[" + indexType + "]", false, false
	}
	return "", false, false
}

func (p *StructParser) extractTag(tagString, tagName string) string {
	tag := reflect.StructTag(tagString)
	return tag.Get(tagName)
}

func (p *StructParser) isDatabaseStruct(table TableDefinition) bool {

	for _, field := range table.Fields {
		if field.DBTag != "" || field.DBDefTag != "" || field.StormTag != "" {
			return true
		}
	}

	if len(table.TableLevel) > 0 {
		return true
	}

	return false
}

func (p *StructParser) deriveTableName(structName string) string {

	snake := p.toSnakeCase(structName)

	irregularPlurals := map[string]string{
		"analysis": "analyses",
		"basis":    "bases",
		"datum":    "data",
		"index":    "indexes",
		"matrix":   "matrices",
		"vertex":   "vertices",
		"axis":     "axes",
		"crisis":   "crises",

		"child": "children",
		"foot":  "feet",
		"tooth": "teeth",
		"goose": "geese",
		"man":   "men",
		"woman": "women",
		"mouse": "mice",
	}

	if plural, ok := irregularPlurals[snake]; ok {
		return plural
	}

	if strings.HasSuffix(snake, "y") && !strings.HasSuffix(snake, "ey") && !strings.HasSuffix(snake, "ay") && !strings.HasSuffix(snake, "oy") && !strings.HasSuffix(snake, "uy") {

		return snake[:len(snake)-1] + "ies"
	}
	if strings.HasSuffix(snake, "s") || strings.HasSuffix(snake, "sh") || strings.HasSuffix(snake, "ch") || strings.HasSuffix(snake, "x") || strings.HasSuffix(snake, "z") {
		return snake + "es"
	}
	return snake + "s"
}

func (p *StructParser) toSnakeCase(s string) string {

	edgeCases := map[string]string{
		"OAuth2Token": "oauth2_token",
		"OAuth2":      "oauth2",
		"OAuth":       "oauth",
	}
	if result, ok := edgeCases[s]; ok {
		return result
	}

	var result strings.Builder

	for i, r := range s {
		isUpper := r >= 'A' && r <= 'Z'

		if i > 0 {
			prevIsLower := s[i-1] >= 'a' && s[i-1] <= 'z'
			prevIsDigit := s[i-1] >= '0' && s[i-1] <= '9'
			prevIsUpper := s[i-1] >= 'A' && s[i-1] <= 'Z'

			if isUpper && (prevIsLower || prevIsDigit) {
				result.WriteRune('_')
			} else if isUpper && prevIsUpper && i+1 < len(s) {

				nextIsLower := s[i+1] >= 'a' && s[i+1] <= 'z'
				if nextIsLower {
					result.WriteRune('_')
				}
			} else if (r >= 'a' && r <= 'z') && prevIsDigit {

				if i >= 2 {
					prevPrevIsDigit := s[i-2] >= '0' && s[i-2] <= '9'
					if !prevPrevIsDigit || !isOrdinalSuffix(s[i-1:]) {
						result.WriteRune('_')
					}
				} else {
					result.WriteRune('_')
				}
			}
		}

		if isUpper {
			result.WriteRune(r - 'A' + 'a')
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

func isOrdinalSuffix(s string) bool {
	if len(s) < 2 {
		return false
	}
	suffix := s[:2]
	return suffix == "st" || suffix == "nd" || suffix == "rd" || suffix == "th"
}

func (p *StructParser) exprToString(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return p.exprToString(v.X) + "." + v.Sel.Name
	default:
		return ""
	}
}
