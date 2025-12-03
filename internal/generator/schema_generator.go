package generator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/eleven-am/storm/internal/logger"
	parser2 "github.com/eleven-am/storm/internal/parser"
)

// SchemaColumn represents a column in the target database schema
type SchemaColumn struct {
	Name            string
	Type            string
	IsNullable      bool
	DefaultValue    *string
	IsPrimaryKey    bool
	IsUnique        bool
	IsAutoIncrement bool
	ForeignKey      *ForeignKeyRef
	CheckConstraint *string
	EnumValues      []string
}

// ForeignKeyRef represents a foreign key reference
type ForeignKeyRef struct {
	ReferencedTable  string
	ReferencedColumn string
	OnDelete         string
	OnUpdate         string
}

// SchemaTable represents a table in the target database schema
type SchemaTable struct {
	Name        string
	Columns     []SchemaColumn
	Indexes     []SchemaIndex
	Constraints []SchemaConstraint
}

// SchemaIndex represents a database index
type SchemaIndex struct {
	Name      string
	Columns   []string
	IsUnique  bool
	IsPrimary bool
	Type      string
	Where     string
}

// SchemaConstraint represents a table constraint
type SchemaConstraint struct {
	Name       string
	Type       string
	Definition string
	Columns    []string
}

// DatabaseSchema represents the complete target database schema
type DatabaseSchema struct {
	Tables    map[string]SchemaTable
	EnumTypes map[string][]string
}

// SchemaGenerator converts parsed struct definitions to database schema
type SchemaGenerator struct {
	tagParser *parser2.TagParser
}

func NewSchemaGenerator() *SchemaGenerator {
	return &SchemaGenerator{
		tagParser: parser2.NewTagParser(),
	}
}

func (g *SchemaGenerator) GenerateSchema(tables []parser2.TableDefinition) (*DatabaseSchema, error) {
	schema := &DatabaseSchema{
		Tables:    make(map[string]SchemaTable),
		EnumTypes: make(map[string][]string),
	}

	for _, tableDef := range tables {
		schemaTable, err := g.generateTable(tableDef)
		if err != nil {
			return nil, fmt.Errorf("failed to generate schema for table %s: %w", tableDef.TableName, err)
		}

		for _, col := range schemaTable.Columns {
			if len(col.EnumValues) > 0 {
				schema.EnumTypes[col.Type] = col.EnumValues
			}
		}

		schema.Tables[schemaTable.Name] = schemaTable
	}

	if err := g.validateForeignKeys(schema); err != nil {
		return nil, fmt.Errorf("foreign key validation failed: %w", err)
	}

	return schema, nil
}

func (g *SchemaGenerator) generateTable(tableDef parser2.TableDefinition) (SchemaTable, error) {
	table := SchemaTable{
		Name:        tableDef.TableName,
		Columns:     make([]SchemaColumn, 0),
		Indexes:     make([]SchemaIndex, 0),
		Constraints: make([]SchemaConstraint, 0),
	}

	for _, field := range tableDef.Fields {
		column, err := g.generateColumn(field, tableDef.TableName)
		if err != nil {
			return table, fmt.Errorf("failed to generate column %s: %w", field.Name, err)
		}
		table.Columns = append(table.Columns, column)
	}

	err := g.processTableLevel(tableDef.TableLevel, &table)
	if err != nil {
		return table, fmt.Errorf("failed to process table-level definitions: %w", err)
	}

	g.addImplicitConstraints(&table)

	return table, nil
}

func (g *SchemaGenerator) generateColumn(field parser2.FieldDefinition, tableName string) (SchemaColumn, error) {
	column := SchemaColumn{
		Name: field.DBName,
	}

	pgType, err := g.mapGoTypeToPostgreSQL(field.Type, field.DBDef)
	if err != nil {
		return column, fmt.Errorf("failed to map type for field %s: %w", field.Name, err)
	}

	if field.IsArray || strings.HasSuffix(pgType, "[]") {
		if arrayType := g.tagParser.GetArrayType(field.DBDef); arrayType != "" {
			column.Type = arrayType + "[]"
		} else {
			column.Type = pgType
		}
	} else {
		column.Type = pgType
	}

	column.IsNullable = field.IsPointer || !g.tagParser.HasFlag(field.DBDef, "not_null")

	column.IsPrimaryKey = g.tagParser.HasFlag(field.DBDef, "primary_key")
	if column.IsPrimaryKey {
		column.IsNullable = false
	}

	column.IsUnique = g.tagParser.HasFlag(field.DBDef, "unique")

	column.IsAutoIncrement = g.tagParser.HasFlag(field.DBDef, "auto_increment") ||
		strings.Contains(strings.ToLower(column.Type), "serial")

	if defaultVal := g.tagParser.GetDefault(field.DBDef); defaultVal != "" {
		column.DefaultValue = &defaultVal
	}

	if fkRef := g.tagParser.GetForeignKey(field.DBDef); fkRef != "" {
		fk, err := g.parseForeignKeyRef(fkRef)
		if err != nil {
			return column, fmt.Errorf("invalid foreign key reference: %w", err)
		}

		if onDelete, exists := field.DBDef["on_delete"]; exists {
			fk.OnDelete = onDelete
		}
		if onUpdate, exists := field.DBDef["on_update"]; exists {
			fk.OnUpdate = onUpdate
		}

		column.ForeignKey = fk
	}

	if checkExpr, exists := field.DBDef["check"]; exists {
		column.CheckConstraint = &checkExpr
	}

	if enumValues := g.tagParser.GetEnum(field.DBDef); enumValues != nil {
		column.EnumValues = enumValues

		enumTypeName := fmt.Sprintf("%s_%s_enum", tableName, column.Name)
		column.Type = enumTypeName

		enumList := make([]string, len(enumValues))
		for i, v := range enumValues {
			enumList[i] = fmt.Sprintf("'%s'", v)
		}
		checkStr := fmt.Sprintf("%s IN (%s)", column.Name, strings.Join(enumList, ", "))
		column.CheckConstraint = &checkStr
	}

	return column, nil
}

func (g *SchemaGenerator) mapGoTypeToPostgreSQL(goType string, dbDef map[string]string) (string, error) {
	if pgType := g.tagParser.GetType(dbDef); pgType != "" {
		switch strings.ToLower(pgType) {
		case "cuid":
			return "CHAR(25)", nil
		case "cuid2":
			return "VARCHAR(32)", nil
		}
		return pgType, nil
	}

	switch goType {
	case "string":
		return "TEXT", nil
	case "int", "int32":
		return "INTEGER", nil
	case "int64":
		return "BIGINT", nil
	case "int16":
		return "SMALLINT", nil
	case "float32":
		return "REAL", nil
	case "float64":
		return "DOUBLE PRECISION", nil
	case "bool":
		return "BOOLEAN", nil
	case "time.Time":
		return "TIMESTAMPTZ", nil
	case "[]byte":
		return "BYTEA", nil
	case "pq.StringArray":
		return "TEXT[]", nil
	case "pq.Int32Array":
		return "INTEGER[]", nil
	case "pq.Int64Array":
		return "BIGINT[]", nil
	case "pq.Float32Array":
		return "REAL[]", nil
	case "pq.Float64Array":
		return "DOUBLE PRECISION[]", nil
	case "pq.BoolArray":
		return "BOOLEAN[]", nil
	case "[]string":
		return "TEXT[]", nil
	case "[]int", "[]int32":
		return "INTEGER[]", nil
	case "[]int64":
		return "BIGINT[]", nil
	case "[]float32":
		return "REAL[]", nil
	case "[]float64":
		return "DOUBLE PRECISION[]", nil
	case "[]bool":
		return "BOOLEAN[]", nil
	case "json.RawMessage", "JSONB":
		return "JSONB", nil
	case "cuid.CUID", "CUID":
		return "CHAR(25)", nil
	default:
		logger.Schema().Warn("Unknown Go type '%s', defaulting to TEXT", goType)
		return "TEXT", nil
	}
}

func (g *SchemaGenerator) parseForeignKeyRef(fkRef string) (*ForeignKeyRef, error) {
	parts := strings.Split(fkRef, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("foreign key must be in format 'table.column', got: %s", fkRef)
	}

	return &ForeignKeyRef{
		ReferencedTable:  strings.TrimSpace(parts[0]),
		ReferencedColumn: strings.TrimSpace(parts[1]),
		OnDelete:         "NO ACTION",
		OnUpdate:         "NO ACTION",
	}, nil
}

func (g *SchemaGenerator) processTableLevel(tableLevelDef map[string]string, table *SchemaTable) error {
	for key, value := range tableLevelDef {
		switch key {
		case "table":
			continue
		case "index":
			indexes, err := g.parseIndexDefinition(value, table.Name)
			if err != nil {
				return fmt.Errorf("failed to parse index definition: %w", err)
			}
			table.Indexes = append(table.Indexes, indexes...)
		case "unique":

			uniqueDefs := strings.Split(value, ";")

			for _, uniqueDef := range uniqueDefs {
				uniqueDef = strings.TrimSpace(uniqueDef)
				if uniqueDef == "" {
					continue
				}

				logger.Schema().Debug("Processing unique constraint definition: %s", uniqueDef)

				if strings.Contains(uniqueDef, "where:") || strings.Contains(uniqueDef, "WHERE:") {
					parts := strings.Split(uniqueDef, ",")
					if len(parts) < 2 {
						return fmt.Errorf("unique constraint must have name and columns: %s", uniqueDef)
					}

					indexName := strings.TrimSpace(parts[0])
					var columns []string
					var whereClause string

					for i := 1; i < len(parts); i++ {
						col := strings.TrimSpace(parts[i])
						if strings.Contains(col, " where:") || strings.Contains(col, " WHERE:") {
							subParts := strings.SplitN(col, " where:", 2)
							if len(subParts) == 2 {
								columns = append(columns, strings.TrimSpace(subParts[0]))
								whereClause = strings.TrimSpace(subParts[1])
							} else {
								subParts = strings.SplitN(col, " WHERE:", 2)
								if len(subParts) == 2 {
									columns = append(columns, strings.TrimSpace(subParts[0]))
									whereClause = strings.TrimSpace(subParts[1])
								}
							}
						} else if strings.HasPrefix(col, "where:") || strings.HasPrefix(col, "WHERE:") {
							whereClause = strings.TrimPrefix(strings.TrimPrefix(col, "where:"), "WHERE:")
						} else if col != "" {
							columns = append(columns, col)
						}
					}

					index := SchemaIndex{
						Name:     indexName,
						Columns:  columns,
						IsUnique: true,
						Where:    whereClause,
					}
					table.Indexes = append(table.Indexes, index)
				} else {
					constraint, err := g.parseUniqueConstraint(uniqueDef, table.Name)
					if err != nil {
						logger.Schema().Warn("Failed to parse unique constraint: %v", err)
						continue
					}

					if len(constraint.Columns) == 1 {
						columnName := constraint.Columns[0]
						skipConstraint := false
						for _, col := range table.Columns {
							if col.Name == columnName && col.IsUnique {
								logger.Schema().Debug("Skipping duplicate unique constraint %s for column %s (column already has UNIQUE)", constraint.Name, columnName)
								skipConstraint = true
								break
							}
						}
						if skipConstraint {
							continue
						}
					}

					logger.Schema().Debug("Parsed unique constraint: Name=%s, Columns=%v", constraint.Name, constraint.Columns)
					table.Constraints = append(table.Constraints, constraint)
				}
			}
		case "check":
			constraint, err := g.parseCheckConstraint(value, table.Name)
			if err != nil {
				return fmt.Errorf("failed to parse check constraint: %w", err)
			}
			table.Constraints = append(table.Constraints, constraint)
		default:
			logger.Schema().Warn("Unknown table-level attribute '%s'", key)
		}
	}

	return nil
}

func (g *SchemaGenerator) parseIndexDefinition(indexDef, tableName string) ([]SchemaIndex, error) {
	var indexes []SchemaIndex

	indexDefs := strings.Split(indexDef, ";")

	for _, def := range indexDefs {
		def = strings.TrimSpace(def)
		if def == "" {
			continue
		}

		var whereClause string
		if whereIdx := strings.Index(def, " where:"); whereIdx != -1 {
			whereClause = def[whereIdx+7:]
			def = def[:whereIdx]
		}

		var indexType string
		if usingIdx := strings.Index(def, " using:"); usingIdx != -1 {
			indexType = def[usingIdx+7:]
			def = def[:usingIdx]
		}

		parts := strings.Split(def, ",")
		if len(parts) < 2 {
			return nil, fmt.Errorf("index definition must have at least name and one column: %s", def)
		}

		index := SchemaIndex{
			Name:     strings.TrimSpace(parts[0]),
			Columns:  make([]string, 0),
			IsUnique: false,
		}

		if whereClause != "" {
			index.Where = whereClause
		}
		if indexType != "" {
			index.Type = indexType
		}

		for i := 1; i < len(parts); i++ {
			part := strings.TrimSpace(parts[i])

			if part == "" {
				continue
			}

			if strings.ToLower(part) == "unique" {
				index.IsUnique = true
				continue
			}

			column := part
			if strings.HasSuffix(strings.ToLower(part), " desc") {
				column = part[:len(part)-5] + " DESC"
			} else if strings.HasSuffix(strings.ToLower(part), " asc") {
				column = part[:len(part)-4] + " ASC"
			}

			index.Columns = append(index.Columns, column)
		}

		if len(index.Columns) == 0 {
			return nil, fmt.Errorf("index must have at least one column: %s", def)
		}

		indexes = append(indexes, index)
	}

	return indexes, nil
}

func (g *SchemaGenerator) parseUniqueConstraint(uniqueDef, tableName string) (SchemaConstraint, error) {
	parts := strings.Split(uniqueDef, ",")
	if len(parts) < 2 {
		return SchemaConstraint{}, fmt.Errorf("unique constraint must have name and columns: %s", uniqueDef)
	}

	constraint := SchemaConstraint{
		Name:    strings.TrimSpace(parts[0]),
		Type:    "UNIQUE",
		Columns: make([]string, 0),
	}

	var hasWhere bool
	for i := 1; i < len(parts); i++ {
		col := strings.TrimSpace(parts[i])
		if strings.HasPrefix(col, "where:") || strings.HasPrefix(col, "WHERE:") {
			hasWhere = true
			break
		}
		if col != "" {
			constraint.Columns = append(constraint.Columns, col)
		}
	}

	if hasWhere {
		return SchemaConstraint{}, fmt.Errorf("partial unique constraints should be created as indexes")
	}

	return constraint, nil
}

func (g *SchemaGenerator) parseCheckConstraint(checkDef, tableName string) (SchemaConstraint, error) {
	parts := strings.SplitN(checkDef, ",", 2)
	if len(parts) != 2 {
		return SchemaConstraint{}, fmt.Errorf("check constraint must have name and expression: %s", checkDef)
	}

	return SchemaConstraint{
		Name:       strings.TrimSpace(parts[0]),
		Type:       "CHECK",
		Definition: strings.TrimSpace(parts[1]),
	}, nil
}

func (g *SchemaGenerator) addImplicitConstraints(table *SchemaTable) {
	var primaryKeyColumns []string

	for _, column := range table.Columns {
		if column.IsPrimaryKey {
			primaryKeyColumns = append(primaryKeyColumns, column.Name)
		}

		if column.IsUnique && !column.IsPrimaryKey {

			hasExistingConstraint := false
			for _, existingConstraint := range table.Constraints {
				if existingConstraint.Type == "UNIQUE" && len(existingConstraint.Columns) == 1 && existingConstraint.Columns[0] == column.Name {
					hasExistingConstraint = true
					break
				}
			}

			if !hasExistingConstraint {
				constraintName := fmt.Sprintf("%s_%s_key", table.Name, column.Name)
				constraint := SchemaConstraint{
					Name:    constraintName,
					Type:    "UNIQUE",
					Columns: []string{column.Name},
				}
				table.Constraints = append(table.Constraints, constraint)
			}
		}

		if column.ForeignKey != nil {
			constraintName := fmt.Sprintf("%s_%s_fkey", table.Name, column.Name)

			definition := fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s(%s)",
				column.Name,
				column.ForeignKey.ReferencedTable,
				column.ForeignKey.ReferencedColumn)

			if column.ForeignKey.OnDelete != "" && column.ForeignKey.OnDelete != "NO ACTION" {
				definition += fmt.Sprintf(" ON DELETE %s", column.ForeignKey.OnDelete)
			}
			if column.ForeignKey.OnUpdate != "" && column.ForeignKey.OnUpdate != "NO ACTION" {
				definition += fmt.Sprintf(" ON UPDATE %s", column.ForeignKey.OnUpdate)
			}

			constraint := SchemaConstraint{
				Name:       constraintName,
				Type:       "FOREIGN KEY",
				Columns:    []string{column.Name},
				Definition: definition,
			}
			table.Constraints = append(table.Constraints, constraint)
		}
	}

	if len(primaryKeyColumns) > 0 {
		pkConstraintName := fmt.Sprintf("%s_pkey", table.Name)
		constraint := SchemaConstraint{
			Name:    pkConstraintName,
			Type:    "PRIMARY KEY",
			Columns: primaryKeyColumns,
		}
		table.Constraints = append(table.Constraints, constraint)

	}
}

func (s *DatabaseSchema) GetTableNames() []string {
	var names []string
	for name := range s.Tables {
		names = append(names, name)
	}

	sorted := s.sortTablesByDependencies(names)
	return sorted
}

func (s *DatabaseSchema) sortTablesByDependencies(tables []string) []string {
	dependencies := make(map[string][]string)
	dependents := make(map[string][]string)

	for _, table := range tables {
		dependencies[table] = []string{}
		dependents[table] = []string{}
	}

	for _, tableName := range tables {
		table := s.Tables[tableName]
		for _, col := range table.Columns {
			if col.ForeignKey != nil {
				refTable := col.ForeignKey.ReferencedTable
				dependents[tableName] = append(dependents[tableName], refTable)
				dependencies[refTable] = append(dependencies[refTable], tableName)
			}
		}
	}

	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	var result []string

	var visit func(string) bool
	visit = func(table string) bool {
		if visiting[table] {
			return false
		}
		if visited[table] {
			return true
		}

		visiting[table] = true

		for _, dep := range dependents[table] {
			if !visit(dep) {
				return false
			}
		}

		visiting[table] = false
		visited[table] = true
		result = append(result, table)
		return true
	}

	for _, table := range tables {
		if !visited[table] {
			if !visit(table) {
				sort.Strings(tables)
				return tables
			}
		}
	}

	return result
}

func (s *DatabaseSchema) HasTable(tableName string) bool {
	_, exists := s.Tables[tableName]
	return exists
}

func (s *DatabaseSchema) GetTable(tableName string) (SchemaTable, bool) {
	table, exists := s.Tables[tableName]
	return table, exists
}

func (g *SchemaGenerator) validateForeignKeys(schema *DatabaseSchema) error {
	var errors []string

	for tableName, table := range schema.Tables {
		for _, column := range table.Columns {
			if column.ForeignKey != nil {
				referencedTable := column.ForeignKey.ReferencedTable

				if !schema.HasTable(referencedTable) {
					errors = append(errors, fmt.Sprintf(
						"table '%s', column '%s': foreign key references non-existent table '%s'",
						tableName, column.Name, referencedTable))
					continue
				}

				refTable := schema.Tables[referencedTable]
				columnExists := false
				for _, refCol := range refTable.Columns {
					if refCol.Name == column.ForeignKey.ReferencedColumn {
						columnExists = true
						break
					}
				}

				if !columnExists {
					errors = append(errors, fmt.Sprintf(
						"table '%s', column '%s': foreign key references non-existent column '%s.%s'",
						tableName, column.Name, referencedTable, column.ForeignKey.ReferencedColumn))
				}
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("invalid foreign key references found:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}
