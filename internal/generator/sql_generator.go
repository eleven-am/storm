package generator

import (
	"fmt"
	"strings"

	"github.com/eleven-am/storm/internal/logger"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SQLGenerator generates SQL DDL from database schema
type SQLGenerator struct{}

func NewSQLGenerator() *SQLGenerator {
	return &SQLGenerator{}
}

func (g *SQLGenerator) GenerateCreateTable(table SchemaTable) string {
	var sql strings.Builder

	sql.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", table.Name))

	columns := make([]string, 0, len(table.Columns))
	for _, col := range table.Columns {
		columns = append(columns, g.generateColumnDDL(col))
	}

	constraints := make([]string, 0)

	var pkColumns []string
	for _, col := range table.Columns {
		if col.IsPrimaryKey {
			pkColumns = append(pkColumns, g.quoteColumnNameIfNeeded(col.Name))
		}
	}
	if len(pkColumns) > 0 {
		constraints = append(constraints, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pkColumns, ", ")))
	}

	for _, constraint := range table.Constraints {
		logger.SQL().Debug("Processing constraint for table %s: Type=%s, Name=%s, Columns=%v",
			table.Name, constraint.Type, constraint.Name, constraint.Columns)
		switch constraint.Type {
		case "UNIQUE":

			quotedColumns := make([]string, len(constraint.Columns))
			for i, col := range constraint.Columns {
				quotedColumns[i] = g.quoteColumnNameIfNeeded(col)
			}
			constraintSQL := fmt.Sprintf("CONSTRAINT %s UNIQUE (%s)",
				constraint.Name, strings.Join(quotedColumns, ", "))
			logger.SQL().Debug("Generated UNIQUE constraint: %s", constraintSQL)
			constraints = append(constraints, constraintSQL)
		case "CHECK":
			constraints = append(constraints, fmt.Sprintf("CONSTRAINT %s CHECK (%s)",
				constraint.Name, constraint.Definition))
		case "FOREIGN KEY":
			continue
		}
	}

	allDefs := append(columns, constraints...)
	logger.SQL().Debug("All definitions for %s: %d columns, %d constraints", table.Name, len(columns), len(constraints))
	for i, def := range allDefs {
		logger.SQL().Debug("Definition %d: %s", i, def)
	}
	joinedDefs := strings.Join(allDefs, ",\n    ")
	sql.WriteString("    " + joinedDefs)
	sql.WriteString("\n);\n")

	for _, idx := range table.Indexes {
		if !g.isImplicitIndex(idx, table) {
			sql.WriteString("\n" + g.GenerateIndexDDL(table.Name, idx))
		}
	}

	return sql.String()
}

func (g *SQLGenerator) generateColumnDDL(col SchemaColumn) string {
	var parts []string

	colName := g.quoteColumnNameIfNeeded(col.Name)
	parts = append(parts, colName, col.Type)

	if !col.IsNullable {
		parts = append(parts, "NOT NULL")
	}

	if col.DefaultValue != nil {
		defaultValue := g.formatDefaultValue(col.Type, *col.DefaultValue)
		parts = append(parts, fmt.Sprintf("DEFAULT %s", defaultValue))
		logger.SQL().Debug("Column %s type %s default %s -> %s", col.Name, col.Type, *col.DefaultValue, defaultValue)
	}

	if col.IsUnique && !col.IsPrimaryKey {
		parts = append(parts, "UNIQUE")
	}

	if col.ForeignKey != nil {
		parts = append(parts, fmt.Sprintf("REFERENCES %s(%s)",
			col.ForeignKey.ReferencedTable, col.ForeignKey.ReferencedColumn))

		if col.ForeignKey.OnDelete != "" && col.ForeignKey.OnDelete != "NO ACTION" {
			parts = append(parts, fmt.Sprintf("ON DELETE %s", col.ForeignKey.OnDelete))
		}
		if col.ForeignKey.OnUpdate != "" && col.ForeignKey.OnUpdate != "NO ACTION" {
			parts = append(parts, fmt.Sprintf("ON UPDATE %s", col.ForeignKey.OnUpdate))
		}
	}

	if col.CheckConstraint != nil {
		parts = append(parts, fmt.Sprintf("CHECK (%s)", *col.CheckConstraint))
	}

	return strings.Join(parts, " ")
}

func (g *SQLGenerator) GenerateIndexDDL(tableName string, idx SchemaIndex) string {
	var sql strings.Builder

	if idx.IsUnique {
		sql.WriteString("CREATE UNIQUE INDEX ")
	} else {
		sql.WriteString("CREATE INDEX ")
	}

	sql.WriteString(idx.Name)
	sql.WriteString(" ON ")
	sql.WriteString(tableName)

	if idx.Type != "" && idx.Type != "btree" {
		sql.WriteString(" USING ")
		sql.WriteString(idx.Type)
	}

	sql.WriteString(" (")

	quotedColumns := make([]string, len(idx.Columns))
	for i, col := range idx.Columns {
		quotedColumns[i] = g.quoteColumnNameIfNeeded(col)
	}
	sql.WriteString(strings.Join(quotedColumns, ", "))
	sql.WriteString(")")

	if idx.Where != "" {
		sql.WriteString(" WHERE ")
		sql.WriteString(idx.Where)
	}

	sql.WriteString(";\n")

	return sql.String()
}

func (g *SQLGenerator) isImplicitIndex(idx SchemaIndex, table SchemaTable) bool {
	if idx.IsPrimary {
		return true
	}

	if idx.IsUnique && len(idx.Columns) == 1 {
		for _, col := range table.Columns {
			if col.Name == idx.Columns[0] && col.IsUnique {
				return true
			}
		}
	}

	return false
}

func (g *SQLGenerator) generateEnumType(typeName string, values []string) string {
	var sql strings.Builder

	sql.WriteString("CREATE TYPE ")
	sql.WriteString(typeName)
	sql.WriteString(" AS ENUM (")

	quotedValues := make([]string, len(values))
	for i, v := range values {
		quotedValues[i] = fmt.Sprintf("'%s'", v)
	}

	sql.WriteString(strings.Join(quotedValues, ", "))
	sql.WriteString(");")

	return sql.String()
}

func (g *SQLGenerator) GenerateSchema(schema *DatabaseSchema) string {
	var sql strings.Builder

	logger.SQL().Debug("Starting schema generation for %d tables", len(schema.Tables))

	sql.WriteString("-- Generated by webhook-router migration tool\n")
	sql.WriteString("-- Enable required extensions\n")
	sql.WriteString("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";\n")
	sql.WriteString("CREATE EXTENSION IF NOT EXISTS \"pgcrypto\";\n\n")

	logger.SQL().Debug("Added extensions")

	if len(schema.EnumTypes) > 0 {
		sql.WriteString("-- Enum types\n")
		for typeName, values := range schema.EnumTypes {
			sql.WriteString(g.generateEnumType(typeName, values))
			sql.WriteString("\n")
		}
		sql.WriteString("\n")
	}

	if g.schemaUsesCUIDs(schema) {
		logger.SQL().Debug("Schema uses CUIDs, but CUID functions will be handled by the migrator")
		sql.WriteString("-- CUID functions will be generated by the migration system\n\n")
	}

	tableNames := schema.GetTableNames()
	logger.SQL().Debug("Generating %d tables: %v", len(tableNames), tableNames)

	for _, tableName := range tableNames {
		table := schema.Tables[tableName]
		logger.SQL().Debug("Processing table %s with %d columns", tableName, len(table.Columns))
		sql.WriteString(fmt.Sprintf("-- Table: %s\n", tableName))
		tableSQL := g.GenerateCreateTable(table)
		logger.SQL().Debug("Generated SQL for %s: %s", tableName, tableSQL[:min(200, len(tableSQL))])
		sql.WriteString(tableSQL)
		sql.WriteString("\n")
	}

	finalSQL := sql.String()
	logger.SQL().Debug("Final SQL length: %d characters", len(finalSQL))
	logger.SQL().Debug("First 500 chars: %s", finalSQL[:min(500, len(finalSQL))])
	return finalSQL
}

// formatDefaultValue properly formats default values based on column type
func (g *SQLGenerator) formatDefaultValue(colType, defaultValue string) string {

	lower := strings.ToLower(defaultValue)
	if strings.Contains(lower, "()") ||
		lower == "true" || lower == "false" ||
		strings.HasPrefix(lower, "nextval(") ||
		strings.HasPrefix(lower, "'{") ||
		strings.HasPrefix(lower, "'") ||
		strings.HasPrefix(lower, "\"") {
		return defaultValue
	}

	colTypeLower := strings.ToLower(colType)
	if strings.Contains(colTypeLower, "varchar") ||
		strings.Contains(colTypeLower, "text") ||
		strings.Contains(colTypeLower, "char") {
		return fmt.Sprintf("'%s'", defaultValue)
	}

	if strings.ContainsAny(defaultValue, "0123456789") &&
		len(strings.Fields(defaultValue)) == 1 {

		if _, err := fmt.Sscanf(defaultValue, "%f", new(float64)); err == nil {
			return defaultValue
		}
	}

	return defaultValue
}

func (g *SQLGenerator) schemaUsesCUIDs(schema *DatabaseSchema) bool {
	for _, table := range schema.Tables {
		for _, col := range table.Columns {
			colType := strings.ToUpper(col.Type)
			if strings.Contains(colType, "CUID") ||
				col.Type == "CHAR(25)" ||
				col.Type == "VARCHAR(32)" {
				return true
			}
			if col.DefaultValue != nil && strings.Contains(strings.ToLower(*col.DefaultValue), "cuid") {
				return true
			}
		}
	}
	return false
}

// quoteColumnNameIfNeeded quotes column names that are PostgreSQL reserved keywords
func (g *SQLGenerator) quoteColumnNameIfNeeded(name string) string {

	reservedKeywords := map[string]bool{
		"user":       true,
		"order":      true,
		"group":      true,
		"table":      true,
		"column":     true,
		"select":     true,
		"insert":     true,
		"update":     true,
		"delete":     true,
		"from":       true,
		"where":      true,
		"join":       true,
		"left":       true,
		"right":      true,
		"inner":      true,
		"outer":      true,
		"on":         true,
		"as":         true,
		"by":         true,
		"desc":       true,
		"asc":        true,
		"limit":      true,
		"offset":     true,
		"union":      true,
		"all":        true,
		"distinct":   true,
		"between":    true,
		"like":       true,
		"in":         true,
		"exists":     true,
		"case":       true,
		"when":       true,
		"then":       true,
		"else":       true,
		"end":        true,
		"null":       true,
		"not":        true,
		"and":        true,
		"or":         true,
		"primary":    true,
		"foreign":    true,
		"key":        true,
		"references": true,
		"unique":     true,
		"index":      true,
		"default":    true,
		"check":      true,
		"constraint": true,
		"trigger":    true,
		"procedure":  true,
		"function":   true,
		"view":       true,
		"grant":      true,
		"revoke":     true,
		"role":       true,
		"password":   true,
		"timestamp":  true,
		"date":       true,
		"time":       true,
		"interval":   true,
		"array":      true,
		"json":       true,
		"jsonb":      true,
		"uuid":       true,
		"serial":     true,
		"sequence":   true,
		"cascade":    true,
		"restrict":   true,
		"action":     true,
		"session":    true,
		"current":    true,
		"true":       true,
		"false":      true,
		"boolean":    true,
		"integer":    true,
		"decimal":    true,
		"numeric":    true,
		"real":       true,
		"double":     true,
		"precision":  true,
		"varchar":    true,
		"char":       true,
		"text":       true,
		"bytea":      true,
		"bit":        true,
		"values":     true,
		"using":      true,
		"returning":  true,
		"with":       true,
		"recursive":  true,
		"window":     true,
		"partition":  true,
		"over":       true,
		"rows":       true,
		"range":      true,
		"groups":     true,
		"exclude":    true,
		"others":     true,
		"ties":       true,
		"rollup":     true,
		"cube":       true,
		"grouping":   true,
		"sets":       true,
	}

	if reservedKeywords[strings.ToLower(name)] {
		return fmt.Sprintf(`"%s"`, name)
	}

	return name
}
