package orm

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

// QueryLogger defines the interface for logging SQL queries
type QueryLogger interface {
	LogQuery(query string, args []interface{}, duration time.Duration, err error)
}

// SimpleQueryLogger is a basic implementation of QueryLogger that logs to stdout
type SimpleQueryLogger struct{}

func (s *SimpleQueryLogger) LogQuery(query string, args []interface{}, duration time.Duration, err error) {
	status := "SUCCESS"
	if err != nil {
		status = fmt.Sprintf("ERROR: %v", err)
	}
	fmt.Printf("[SQL] [%v] [%s] %s %v\n", duration, status, query, args)
}

// Storm is the main entry point for all ORM operations
// It holds all repositories and manages database connections
type Storm struct {
	db       DBExecutor
	executor DBExecutor  // Current executor (DB or TX)
	logger   QueryLogger // Optional query logger

	// Repository registry - will be populated by code generation
	repositories map[string]interface{}
}

func NewStorm(db *sqlx.DB, logger ...QueryLogger) *Storm {
	storm := &Storm{
		db:           db,
		repositories: make(map[string]interface{}),
	}

	if len(logger) > 0 {
		storm.logger = logger[0]
		storm.executor = &loggingExecutor{executor: db, logger: logger[0]}
	} else {
		storm.executor = db
	}

	storm.initializeRepositories()

	return storm
}

func newStormWithExecutor(db *sqlx.DB, executor DBExecutor, logger QueryLogger) *Storm {
	storm := &Storm{
		db:           db,
		logger:       logger,
		repositories: make(map[string]interface{}),
	}

	if logger != nil {
		storm.executor = &loggingExecutor{executor: executor, logger: logger}
	} else {
		storm.executor = executor
	}

	storm.initializeRepositories()
	return storm
}


// loggingExecutor wraps a DBExecutor to add query logging functionality
type loggingExecutor struct {
	executor DBExecutor
	logger   QueryLogger
}

func (l *loggingExecutor) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := l.executor.ExecContext(ctx, query, args...)
	duration := time.Since(start)
	l.logger.LogQuery(query, args, duration, err)
	return result, err
}

func (l *loggingExecutor) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := l.executor.QueryContext(ctx, query, args...)
	duration := time.Since(start)
	l.logger.LogQuery(query, args, duration, err)
	return rows, err
}

func (l *loggingExecutor) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	row := l.executor.QueryRowContext(ctx, query, args...)
	duration := time.Since(start)
	l.logger.LogQuery(query, args, duration, nil)
	return row
}

func (l *loggingExecutor) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	start := time.Now()
	err := l.executor.GetContext(ctx, dest, query, args...)
	duration := time.Since(start)
	l.logger.LogQuery(query, args, duration, err)
	return err
}

func (l *loggingExecutor) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	start := time.Now()
	err := l.executor.SelectContext(ctx, dest, query, args...)
	duration := time.Since(start)
	l.logger.LogQuery(query, args, duration, err)
	return err
}

func (l *loggingExecutor) QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	start := time.Now()
	rows, err := l.executor.QueryxContext(ctx, query, args...)
	duration := time.Since(start)
	l.logger.LogQuery(query, args, duration, err)
	return rows, err
}

func (l *loggingExecutor) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	start := time.Now()
	row := l.executor.QueryRowxContext(ctx, query, args...)
	duration := time.Since(start)
	l.logger.LogQuery(query, args, duration, nil)
	return row
}

func (l *loggingExecutor) NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := l.executor.NamedExecContext(ctx, query, arg)
	duration := time.Since(start)
	l.logger.LogQuery(query, []interface{}{arg}, duration, err)
	return result, err
}

func (l *loggingExecutor) BindNamed(query string, arg interface{}) (string, []interface{}, error) {
	return l.executor.BindNamed(query, arg)
}

func (l *loggingExecutor) PreparexContext(ctx context.Context, query string) (*sqlx.Stmt, error) {
	return l.executor.PreparexContext(ctx, query)
}

func (l *loggingExecutor) PrepareNamedContext(ctx context.Context, query string) (*sqlx.NamedStmt, error) {
	return l.executor.PrepareNamedContext(ctx, query)
}

func (l *loggingExecutor) Rebind(query string) string {
	return l.executor.Rebind(query)
}

func (l *loggingExecutor) DriverName() string {
	return l.executor.DriverName()
}

// isInTransaction checks if the current executor is a transaction
func (s *Storm) isInTransaction() bool {

	if _, isTransaction := s.executor.(*sqlx.Tx); isTransaction {
		return true
	}

	if loggingExec, ok := s.executor.(*loggingExecutor); ok {
		if _, isTransaction := loggingExec.executor.(*sqlx.Tx); isTransaction {
			return true
		}
	}
	return false
}

func (s *Storm) WithTransaction(ctx context.Context, fn func(*Storm) error) error {

	if s.isInTransaction() {
		return fn(s)
	}

	db, ok := s.db.(*sqlx.DB)
	if !ok {
		return fmt.Errorf("cannot start transaction: executor is not a database connection")
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			if rbErr := tx.Rollback(); rbErr != nil && rbErr.Error() != "sql: transaction has already been committed or rolled back" {

			}
		}
	}()

	txStorm := newStormWithExecutor(db, tx, s.logger)
	if err := fn(txStorm); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	committed = true

	return nil
}

func (s *Storm) WithTransactionOptions(ctx context.Context, opts *TransactionOptions, fn func(*Storm) error) error {

	if s.isInTransaction() {
		return fn(s)
	}

	db, ok := s.db.(*sqlx.DB)
	if !ok {
		return fmt.Errorf("cannot start transaction: executor is not a database connection")
	}

	txOpts := opts.ToTxOptions()
	tx, err := db.BeginTxx(ctx, txOpts)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			if rbErr := tx.Rollback(); rbErr != nil && rbErr.Error() != "sql: transaction has already been committed or rolled back" {

			}
		}
	}()

	txStorm := newStormWithExecutor(db, tx, s.logger)
	if err := fn(txStorm); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	committed = true

	return nil
}

func (s *Storm) GetExecutor() DBExecutor {
	return s.executor
}

func And(conditions ...Condition) Condition {
	sqlizers := make([]squirrel.Sqlizer, len(conditions))
	for i, c := range conditions {
		sqlizers[i] = c.condition
	}
	return Condition{squirrel.And(sqlizers)}
}

func Or(conditions ...Condition) Condition {
	sqlizers := make([]squirrel.Sqlizer, len(conditions))
	for i, c := range conditions {
		sqlizers[i] = c.condition
	}
	return Condition{squirrel.Or(sqlizers)}
}

func Not(condition Condition) Condition {
	return Condition{squirrel.Expr("NOT (?)", condition.ToSqlizer())}
}

func (s *Storm) GetDB() *sqlx.DB {
	if db, ok := s.db.(*sqlx.DB); ok {
		return db
	}
	return nil
}

// GetLogger returns the query logger if set
func (s *Storm) GetLogger() QueryLogger {
	return s.logger
}

func (s *Storm) initializeRepositories() {

}
