package orm

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"strings"
)

// Query provides a fluent interface for building database queries with all features integrated
type Query[T any] struct {
	repo    *Repository[T]
	builder squirrel.SelectBuilder
	err     error
	ctx     context.Context

	// Query options
	limit       *uint64
	offset      *uint64
	orderBy     []string
	whereClause squirrel.And

	// Transaction support
	tx *sqlx.Tx

	// Join support
	joins    []join
	includes []include
}

func (r *Repository[T]) Query(ctx context.Context) *Query[T] {
	query := &Query[T]{
		repo: r,
		builder: squirrel.Select(r.Columns()...).
			From(r.metadata.TableName).
			PlaceholderFormat(squirrel.Dollar),
		ctx:         ctx,
		whereClause: squirrel.And{},
		joins:       make([]join, 0),
		includes:    make([]include, 0),
	}

	for _, authFunc := range r.authorizeFuncs {
		query = authFunc(ctx, query)
	}

	return query
}

func (q *Query[T]) WithTx(tx *sqlx.Tx) *Query[T] {
	q.tx = tx
	return q
}

func (q *Query[T]) Where(condition Condition) *Query[T] {
	if q.err != nil {
		return q
	}
	q.whereClause = append(q.whereClause, condition.ToSqlizer())
	return q
}

func (q *Query[T]) OrderBy(expressions ...string) *Query[T] {
	if q.err != nil {
		return q
	}
	q.orderBy = append(q.orderBy, expressions...)
	return q
}

func (q *Query[T]) Limit(limit uint64) *Query[T] {
	if q.err != nil {
		return q
	}
	q.limit = &limit
	return q
}

func (q *Query[T]) Offset(offset uint64) *Query[T] {
	if q.err != nil {
		return q
	}
	q.offset = &offset
	return q
}

func (q *Query[T]) Join(joinType JoinType, table, condition string) *Query[T] {
	if q.err != nil {
		return q
	}
	q.joins = append(q.joins, join{
		Type:      joinType,
		Table:     table,
		Condition: condition,
	})
	return q
}

func (q *Query[T]) InnerJoin(table, condition string) *Query[T] {
	return q.Join(InnerJoin, table, condition)
}

func (q *Query[T]) LeftJoin(table, condition string) *Query[T] {
	return q.Join(LeftJoin, table, condition)
}

func (q *Query[T]) RightJoin(table, condition string) *Query[T] {
	return q.Join(RightJoin, table, condition)
}

func (q *Query[T]) FullJoin(table, condition string) *Query[T] {
	return q.Join(FullJoin, table, condition)
}

func (q *Query[T]) Include(relationships ...string) *Query[T] {
	if q.err != nil {
		return q
	}
	for _, rel := range relationships {
		q.includes = append(q.includes, include{
			name:       rel,
			conditions: make([]Condition, 0),
		})
	}
	return q
}

func (q *Query[T]) IncludeWhere(relationship string, conditions ...Condition) *Query[T] {
	if q.err != nil {
		return q
	}
	q.includes = append(q.includes, include{
		name:       relationship,
		conditions: conditions,
	})
	return q
}

func (q *Query[T]) buildQuery() (string, []interface{}, error) {
	if q.err != nil {
		return "", nil, q.err
	}

	builder := q.builder

	for _, join := range q.joins {
		switch join.Type {
		case InnerJoin:
			builder = builder.InnerJoin(fmt.Sprintf("%s ON %s", join.Table, join.Condition))
		case LeftJoin:
			builder = builder.LeftJoin(fmt.Sprintf("%s ON %s", join.Table, join.Condition))
		case RightJoin:
			builder = builder.RightJoin(fmt.Sprintf("%s ON %s", join.Table, join.Condition))
		case FullJoin:
			builder = builder.Join(fmt.Sprintf("FULL OUTER JOIN %s ON %s", join.Table, join.Condition))
		}
	}

	if len(q.whereClause) > 0 {
		builder = builder.Where(q.whereClause)
	}

	for _, orderBy := range q.orderBy {
		builder = builder.OrderBy(orderBy)
	}

	if q.limit != nil {
		builder = builder.Limit(*q.limit)
	}

	if q.offset != nil {
		builder = builder.Offset(*q.offset)
	}

	baseSQL, baseArgs, err := builder.ToSql()
	if err != nil {
		return "", nil, err
	}

	return baseSQL, baseArgs, nil
}

func (q *Query[T]) Find() ([]T, error) {
	if len(q.includes) > 0 {
		return q.findWithRelationships()
	}

	finalBuilder := q.builder

	for _, join := range q.joins {
		switch join.Type {
		case InnerJoin:
			finalBuilder = finalBuilder.InnerJoin(fmt.Sprintf("%s ON %s", join.Table, join.Condition))
		case LeftJoin:
			finalBuilder = finalBuilder.LeftJoin(fmt.Sprintf("%s ON %s", join.Table, join.Condition))
		case RightJoin:
			finalBuilder = finalBuilder.RightJoin(fmt.Sprintf("%s ON %s", join.Table, join.Condition))
		case FullJoin:
			finalBuilder = finalBuilder.Join(fmt.Sprintf("FULL OUTER JOIN %s ON %s", join.Table, join.Condition))
		}
	}

	if len(q.whereClause) > 0 {
		finalBuilder = finalBuilder.Where(q.whereClause)
	}

	for _, orderBy := range q.orderBy {
		finalBuilder = finalBuilder.OrderBy(orderBy)
	}

	if q.limit != nil {
		finalBuilder = finalBuilder.Limit(*q.limit)
	}

	if q.offset != nil {
		finalBuilder = finalBuilder.Offset(*q.offset)
	}

	var records []T
	err := q.repo.executeQueryMiddleware(OpQuery, q.ctx, nil, finalBuilder, func(middlewareCtx *MiddlewareContext) error {
		finalQuery := middlewareCtx.QueryBuilder.(squirrel.SelectBuilder)

		sqlQuery, args, err := finalQuery.ToSql()
		if err != nil {
			return &Error{
				Op:    "find",
				Table: q.repo.metadata.TableName,
				Err:   fmt.Errorf("failed to build query: %w", err),
			}
		}

		var execErr error
		if q.tx != nil {
			execErr = q.tx.SelectContext(q.ctx, &records, sqlQuery, args...)
		} else {
			execErr = q.repo.db.SelectContext(q.ctx, &records, sqlQuery, args...)
		}

		if execErr != nil {
			return &Error{
				Op:    "find",
				Table: q.repo.metadata.TableName,
				Err:   fmt.Errorf("failed to execute query: %w", execErr),
			}
		}

		return nil
	})

	return records, err
}

func (q *Query[T]) First() (*T, error) {
	q.Limit(1)
	records, err := q.Find()
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, &Error{
			Op:    "first",
			Table: q.repo.metadata.TableName,
			Err:   ErrNotFound,
		}
	}

	return &records[0], nil
}

func (q *Query[T]) Count() (int64, error) {
	countBuilder := squirrel.Select("COUNT(*)").
		From(q.repo.metadata.TableName).
		PlaceholderFormat(squirrel.Dollar)

	for _, join := range q.joins {
		switch join.Type {
		case InnerJoin:
			countBuilder = countBuilder.InnerJoin(fmt.Sprintf("%s ON %s", join.Table, join.Condition))
		case LeftJoin:
			countBuilder = countBuilder.LeftJoin(fmt.Sprintf("%s ON %s", join.Table, join.Condition))
		case RightJoin:
			countBuilder = countBuilder.RightJoin(fmt.Sprintf("%s ON %s", join.Table, join.Condition))
		case FullJoin:
			countBuilder = countBuilder.Join(fmt.Sprintf("FULL OUTER JOIN %s ON %s", join.Table, join.Condition))
		}
	}

	if len(q.whereClause) > 0 {
		countBuilder = countBuilder.Where(q.whereClause)
	}

	var count int64
	err := q.repo.executeQueryMiddleware(OpQuery, q.ctx, nil, countBuilder, func(middlewareCtx *MiddlewareContext) error {
		finalQuery := middlewareCtx.QueryBuilder.(squirrel.SelectBuilder)

		sqlQuery, args, err := finalQuery.ToSql()
		if err != nil {
			return &Error{
				Op:    "count",
				Table: q.repo.metadata.TableName,
				Err:   fmt.Errorf("failed to build count query: %w", err),
			}
		}

		var execErr error
		if q.tx != nil {
			execErr = q.tx.GetContext(q.ctx, &count, sqlQuery, args...)
		} else {
			execErr = q.repo.db.GetContext(q.ctx, &count, sqlQuery, args...)
		}

		if execErr != nil {
			return &Error{
				Op:    "count",
				Table: q.repo.metadata.TableName,
				Err:   fmt.Errorf("failed to execute count query: %w", execErr),
			}
		}

		return nil
	})

	return count, err
}

func (q *Query[T]) Exists() (bool, error) {
	count, err := q.Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (q *Query[T]) Delete() (int64, error) {
	deleteBuilder := squirrel.Delete(q.repo.metadata.TableName).
		PlaceholderFormat(squirrel.Dollar)

	if len(q.whereClause) > 0 {
		deleteBuilder = deleteBuilder.Where(q.whereClause)
	}

	var rowsAffected int64
	err := q.repo.executeQueryMiddleware(OpDelete, q.ctx, nil, deleteBuilder, func(middlewareCtx *MiddlewareContext) error {
		finalQuery := middlewareCtx.QueryBuilder.(squirrel.DeleteBuilder)

		sqlQuery, args, err := finalQuery.ToSql()
		if err != nil {
			return &Error{
				Op:    "delete",
				Table: q.repo.metadata.TableName,
				Err:   fmt.Errorf("failed to build delete query: %w", err),
			}
		}

		var result sql.Result
		if q.tx != nil {
			result, err = q.tx.ExecContext(q.ctx, sqlQuery, args...)
		} else {
			result, err = q.repo.db.ExecContext(q.ctx, sqlQuery, args...)
		}

		if err != nil {
			return parsePostgreSQLError(err, "delete", q.repo.metadata.TableName)
		}

		rowsAffected, err = result.RowsAffected()
		if err != nil {
			return &Error{
				Op:    "delete",
				Table: q.repo.metadata.TableName,
				Err:   fmt.Errorf("failed to get rows affected: %w", err),
			}
		}

		return nil
	})

	return rowsAffected, err
}

// Update updates records using type-safe Action operations
func (q *Query[T]) Update(actions ...Action) (int64, error) {
	if len(actions) == 0 {
		return 0, &Error{
			Op:    "update",
			Table: q.repo.metadata.TableName,
			Err:   fmt.Errorf("no actions provided"),
		}
	}

	var setParts []string
	var args []interface{}
	argIndex := 1

	for _, action := range actions {
		expression := action.Expression()
		value := action.Value()

		if value != nil {

			if valueSlice, ok := value.([]interface{}); ok {
				for _, v := range valueSlice {
					expression = strings.Replace(expression, "?", fmt.Sprintf("$%d", argIndex), 1)
					args = append(args, v)
					argIndex++
				}
			} else {

				for strings.Contains(expression, "?") {
					expression = strings.Replace(expression, "?", fmt.Sprintf("$%d", argIndex), 1)
					args = append(args, value)
					argIndex++
				}
			}
		}
		setParts = append(setParts, expression)
	}

	baseSQL := fmt.Sprintf("UPDATE %s SET %s", q.repo.metadata.TableName, strings.Join(setParts, ", "))

	if len(q.whereClause) > 0 {
		whereBuilder := squirrel.Select("1").Where(q.whereClause).PlaceholderFormat(squirrel.Dollar)
		_, whereArgs, err := whereBuilder.ToSql()
		if err != nil {
			return 0, &Error{
				Op:    "update",
				Table: q.repo.metadata.TableName,
				Err:   fmt.Errorf("failed to build where clause: %w", err),
			}
		}

		dummySQL, _, _ := whereBuilder.ToSql()
		whereStart := strings.Index(dummySQL, "WHERE")
		if whereStart != -1 {
			whereClause := dummySQL[whereStart:]

			for i := range whereArgs {
				placeholder := fmt.Sprintf("$%d", i+1)
				newPlaceholder := fmt.Sprintf("$%d", argIndex+i)
				whereClause = strings.Replace(whereClause, placeholder, newPlaceholder, -1)
			}
			baseSQL += " " + whereClause
			args = append(args, whereArgs...)
		}
	}

	var rowsAffected int64
	err := q.repo.executeQueryMiddleware(OpUpdateMany, q.ctx, actions, baseSQL, func(middlewareCtx *MiddlewareContext) error {
		middlewareCtx.Query = baseSQL
		middlewareCtx.Args = args

		var result sql.Result
		var err error
		if q.tx != nil {
			result, err = q.tx.ExecContext(q.ctx, baseSQL, args...)
		} else {
			result, err = q.repo.db.ExecContext(q.ctx, baseSQL, args...)
		}

		if err != nil {
			return parsePostgreSQLError(err, "update", q.repo.metadata.TableName)
		}

		rowsAffected, err = result.RowsAffected()
		if err != nil {
			return &Error{
				Op:    "update",
				Table: q.repo.metadata.TableName,
				Err:   fmt.Errorf("failed to get rows affected: %w", err),
			}
		}

		return nil
	})

	return rowsAffected, err
}

func (q *Query[T]) findWithRelationships() ([]T, error) {

	originalIncludes := q.includes
	q.includes = nil

	records, err := q.Find()
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return records, nil
	}

	for _, include := range originalIncludes {
		if err := q.loadRelationship(records, include); err != nil {
			return nil, fmt.Errorf("failed to load relationship %s: %w", include.name, err)
		}
	}

	return records, nil
}

func (q *Query[T]) loadRelationship(records []T, include include) error {
	if len(records) == 0 {
		return nil
	}

	relationship := q.repo.getRelationship(include.name)
	if relationship == nil {
		return fmt.Errorf("relationship %s not found", include.name)
	}

	if relationship.ScanToModel == nil {
		return fmt.Errorf("relationship %s does not have ScanToModel function", include.name)
	}

	for i := range records {

		recordQuery, recordArgs, err := q.buildSingleRecordQuery(relationship, records[i], include)
		if err != nil {
			return err
		}

		if recordQuery != "" {
			if err := q.executeSingleRelationshipQuery(relationship, recordQuery, recordArgs, &records[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

func (q *Query[T]) executeSingleRelationshipQuery(relationship *RelationshipMetadata, query string, args []interface{}, record *T) error {

	return q.repo.executeQueryMiddleware(OpQuery, q.ctx, record, query, func(middlewareCtx *MiddlewareContext) error {

		var executor DBExecutor
		if q.tx != nil {
			executor = q.tx
		} else {
			executor = q.repo.db
		}

		if err := relationship.ScanToModel(q.ctx, executor, query, args, record); err != nil {
			return &Error{
				Op:    "load_relationship",
				Table: relationship.Target,
				Err:   fmt.Errorf("failed to load relationship %s: %w", relationship.Name, err),
			}
		}

		return nil
	})
}

func (q *Query[T]) buildSingleRecordQuery(relationship *RelationshipMetadata, record T, include include) (string, []interface{}, error) {
	switch relationship.Type {
	case "belongs_to":
		return q.buildBelongsToSingleQuery(relationship, record, include)
	case "has_one":
		return q.buildHasOneSingleQuery(relationship, record, include)
	case "has_many":
		return q.buildHasManySingleQuery(relationship, record, include)
	case "has_many_through":
		return q.buildHasManyThroughSingleQuery(relationship, record, include)
	default:
		return "", nil, fmt.Errorf("unsupported relationship type: %s", relationship.Type)
	}
}

func (q *Query[T]) buildBelongsToSingleQuery(relationship *RelationshipMetadata, record T, include include) (string, []interface{}, error) {

	fkFieldName, ok := q.repo.metadata.ReverseMap[relationship.ForeignKey]
	if !ok {
		fkFieldName = relationship.ForeignKey
		if _, exists := q.repo.metadata.Columns[fkFieldName]; !exists {
			return "", nil, fmt.Errorf("foreign key %s not found", relationship.ForeignKey)
		}
	}

	fkColumn := q.repo.metadata.Columns[fkFieldName]
	if fkColumn == nil {
		return "", nil, fmt.Errorf("foreign key column %s not found", fkFieldName)
	}

	fkValue := fkColumn.GetValue(record)
	if fkValue == nil || isZeroValue(fkValue) {
		return "", nil, nil
	}

	tableName := relationship.TargetTable
	if tableName == "" {
		tableName = relationship.Target
	}

	query := squirrel.Select("*").
		From(tableName).
		Where(squirrel.Eq{relationship.TargetKey: fkValue}).
		PlaceholderFormat(squirrel.Dollar)

	for _, condition := range include.conditions {
		query = query.Where(condition.ToSqlizer())
	}

	return query.ToSql()
}

func (q *Query[T]) buildHasOneSingleQuery(relationship *RelationshipMetadata, record T, include include) (string, []interface{}, error) {

	sourceKey := relationship.SourceKey
	if sourceKey == "" {
		sourceKey = "id"
	}

	sourceFieldName, ok := q.repo.metadata.ReverseMap[sourceKey]
	if !ok {
		sourceFieldName = sourceKey
		if _, exists := q.repo.metadata.Columns[sourceFieldName]; !exists {
			return "", nil, fmt.Errorf("source key %s not found", sourceKey)
		}
	}

	sourceColumn := q.repo.metadata.Columns[sourceFieldName]
	if sourceColumn == nil {
		return "", nil, fmt.Errorf("source key column %s not found", sourceFieldName)
	}

	sourceValue := sourceColumn.GetValue(record)
	if sourceValue == nil || isZeroValue(sourceValue) {
		return "", nil, nil
	}

	tableName := relationship.TargetTable
	if tableName == "" {
		tableName = relationship.Target
	}

	query := squirrel.Select("*").
		From(tableName).
		Where(squirrel.Eq{relationship.ForeignKey: sourceValue}).
		PlaceholderFormat(squirrel.Dollar)

	for _, condition := range include.conditions {
		query = query.Where(condition.ToSqlizer())
	}

	return query.ToSql()
}

func (q *Query[T]) buildHasManySingleQuery(relationship *RelationshipMetadata, record T, include include) (string, []interface{}, error) {

	sourceKey := relationship.SourceKey
	if sourceKey == "" {
		sourceKey = "id"
	}

	sourceFieldName, ok := q.repo.metadata.ReverseMap[sourceKey]
	if !ok {
		sourceFieldName = sourceKey
		if _, exists := q.repo.metadata.Columns[sourceFieldName]; !exists {
			return "", nil, fmt.Errorf("source key %s not found", sourceKey)
		}
	}

	sourceColumn := q.repo.metadata.Columns[sourceFieldName]
	if sourceColumn == nil {
		return "", nil, fmt.Errorf("source key column %s not found", sourceFieldName)
	}

	sourceValue := sourceColumn.GetValue(record)
	if sourceValue == nil || isZeroValue(sourceValue) {
		return "", nil, nil
	}

	tableName := relationship.TargetTable
	if tableName == "" {
		tableName = relationship.Target
	}

	query := squirrel.Select("*").
		From(tableName).
		Where(squirrel.Eq{relationship.ForeignKey: sourceValue}).
		PlaceholderFormat(squirrel.Dollar)

	for _, condition := range include.conditions {
		query = query.Where(condition.ToSqlizer())
	}

	return query.ToSql()
}

func (q *Query[T]) buildHasManyThroughSingleQuery(relationship *RelationshipMetadata, record T, include include) (string, []interface{}, error) {

	sourceKey := relationship.SourceKey
	if sourceKey == "" {
		sourceKey = "id"
	}

	sourceFieldName, ok := q.repo.metadata.ReverseMap[sourceKey]
	if !ok {
		sourceFieldName = sourceKey
		if _, exists := q.repo.metadata.Columns[sourceFieldName]; !exists {
			return "", nil, fmt.Errorf("source key %s not found", sourceKey)
		}
	}

	sourceColumn := q.repo.metadata.Columns[sourceFieldName]
	if sourceColumn == nil {
		return "", nil, fmt.Errorf("source key column %s not found", sourceFieldName)
	}

	sourceValue := sourceColumn.GetValue(record)
	if sourceValue == nil || isZeroValue(sourceValue) {
		return "", nil, nil
	}

	tableName := relationship.TargetTable
	if tableName == "" {
		tableName = relationship.Target
	}

	query := squirrel.Select("t.*").
		From(tableName + " t").
		InnerJoin(fmt.Sprintf("%s jt ON t.%s = jt.%s",
			relationship.Through,
			relationship.TargetKey,
			relationship.ThroughTK)).
		Where(squirrel.Eq{"jt." + relationship.ThroughFK: sourceValue}).
		PlaceholderFormat(squirrel.Dollar)

	for _, condition := range include.conditions {
		query = query.Where(condition.ToSqlizer())
	}

	return query.ToSql()
}

// isZeroValue checks if a value is the zero value for its type
func isZeroValue(v interface{}) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case string:
		return val == ""
	case int:
		return val == 0
	case int64:
		return val == 0
	case int32:
		return val == 0
	case float64:
		return val == 0
	case float32:
		return val == 0
	case bool:
		return !val
	default:

		return false
	}
}

func (q *Query[T]) ExecuteRaw(query string, args ...interface{}) ([]T, error) {
	finalQuery, finalArgs := q.buildFinalQuery(query, args)

	var records []T
	var err error
	if q.tx != nil {
		err = q.tx.SelectContext(q.ctx, &records, finalQuery, finalArgs...)
	} else {
		err = q.repo.db.SelectContext(q.ctx, &records, finalQuery, finalArgs...)
	}

	if err != nil {
		return nil, &Error{
			Op:    "executeRaw",
			Table: q.repo.metadata.TableName,
			Err:   fmt.Errorf("failed to execute raw query: %w", err),
		}
	}

	return records, nil
}

func (q *Query[T]) buildFinalQuery(query string, args []interface{}) (string, []interface{}) {
	return query, args
}
