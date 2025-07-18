package orm

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// AuthorizeFunc defines a callback function for applying authorization to queries
// It receives the request context and query object, and returns a modified query
type AuthorizeFunc[T any] func(ctx context.Context, query *Query[T]) *Query[T]

// Repository provides type-safe database operations for a specific model type
type Repository[T any] struct {
	db       DBExecutor // Keep this accessible for internal packages
	metadata *ModelMetadata

	// Middleware management
	middlewareManager *middlewareManager

	// Authorization functions
	authorizeFuncs []AuthorizeFunc[T]
}

func NewRepository[T any](db *sqlx.DB, metadata *ModelMetadata) (*Repository[T], error) {
	if db == nil {
		return nil, &Error{
			Op:    "initialize",
			Table: "",
			Err:   fmt.Errorf("database cannot be nil"),
		}
	}
	return NewRepositoryWithExecutor[T](db, metadata)
}

func NewRepositoryWithTx[T any](tx *sqlx.Tx, metadata *ModelMetadata) (*Repository[T], error) {
	return NewRepositoryWithExecutor[T](tx, metadata)
}

func NewRepositoryWithExecutor[T any](executor DBExecutor, metadata *ModelMetadata) (*Repository[T], error) {
	if executor == nil {
		return nil, &Error{
			Op:    "initialize",
			Table: "",
			Err:   fmt.Errorf("database cannot be nil"),
		}
	}

	if metadata == nil {
		return nil, &Error{
			Op:    "initialize",
			Table: "",
			Err:   fmt.Errorf("metadata cannot be nil"),
		}
	}

	repo := &Repository[T]{
		db:             executor,
		metadata:       metadata,
		authorizeFuncs: make([]AuthorizeFunc[T], 0),
	}

	if err := repo.initialize(); err != nil {
		return nil, &Error{
			Op:    "initialize",
			Table: metadata.TableName,
			Err:   err,
		}
	}

	return repo, nil
}

func (r *Repository[T]) initialize() error {
	if r.metadata == nil {
		return fmt.Errorf("metadata is nil")
	}

	if len(r.metadata.PrimaryKeys) == 0 {
		return ErrNoPrimaryKey
	}

	r.middlewareManager = newMiddlewareManager()

	return nil
}

func (r *Repository[T]) TableName() string {
	return r.metadata.TableName
}

func (r *Repository[T]) PrimaryKeys() []string {
	return r.metadata.PrimaryKeys
}

func (r *Repository[T]) Columns() []string {
	columns := make([]string, 0, len(r.metadata.Columns))
	for _, col := range r.metadata.Columns {
		columns = append(columns, col.DBName)
	}
	return columns
}

// getRelationship returns the relationship metadata for the given relationship name
func (r *Repository[T]) getRelationship(name string) *RelationshipMetadata {
	if r.metadata.Relationships == nil {
		return nil
	}
	return r.metadata.Relationships[name]
}

func (r *Repository[T]) WithRelationships(ctx context.Context) *Query[T] {
	return r.Query(ctx)
}

// Authorize returns a new Repository instance with an additional authorization function
// Multiple authorization functions can be chained and will be applied in order
func (r *Repository[T]) Authorize(fn AuthorizeFunc[T]) *Repository[T] {
	newFuncs := make([]AuthorizeFunc[T], len(r.authorizeFuncs)+1)
	copy(newFuncs, r.authorizeFuncs)
	newFuncs[len(r.authorizeFuncs)] = fn

	return &Repository[T]{
		db:                r.db,
		metadata:          r.metadata,
		middlewareManager: r.middlewareManager,
		authorizeFuncs:    newFuncs,
	}
}

func (r *Repository[T]) getInsertFields(model T) (columns []string, values []interface{}) {
	for _, colMeta := range r.metadata.Columns {
		if colMeta.IsAutoGenerated {
			continue
		}

		if colMeta.GetValue == nil {
			continue
		}

		if colMeta.IsPointer && colMeta.IsNil != nil {
			if colMeta.IsNil(model) {
				continue // Skip nil pointers (let DB use default)
			}
		}

		value := colMeta.GetValue(model)

		columns = append(columns, colMeta.DBName)
		values = append(values, value)
	}

	return columns, values
}

func (r *Repository[T]) getAutoGeneratedColumns() []string {
	var cols []string
	for _, col := range r.metadata.Columns {
		if col.IsAutoGenerated {
			cols = append(cols, col.DBName)
		}
	}
	return cols
}

func (r *Repository[T]) getPrimaryKeyValues(record T) map[string]interface{} {
	pkValues := make(map[string]interface{})
	for _, pkCol := range r.metadata.PrimaryKeys {
		fieldName := r.metadata.ReverseMap[pkCol]
		if colMeta, exists := r.metadata.Columns[fieldName]; exists && colMeta.GetValue != nil {
			pkValues[pkCol] = colMeta.GetValue(record)
		}
	}
	return pkValues
}

func (r *Repository[T]) getUpdateFields(model T) map[string]interface{} {
	fields := make(map[string]interface{})

	for _, colMeta := range r.metadata.Columns {
		if colMeta.IsPrimaryKey {
			continue
		}

		if colMeta.IsAutoGenerated {
			continue
		}

		if colMeta.GetValue == nil {
			continue
		}

		value := colMeta.GetValue(model)
		fields[colMeta.DBName] = value
	}

	return fields
}
