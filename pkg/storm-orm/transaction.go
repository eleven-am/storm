package orm

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// TransactionOptions configures transaction behavior
type TransactionOptions struct {
	Isolation sql.IsolationLevel
	ReadOnly  bool
}

func DefaultTransactionOptions() *TransactionOptions {
	return &TransactionOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
	}
}

func (o *TransactionOptions) ToTxOptions() *sql.TxOptions {
	if o == nil {
		return nil
	}
	return &sql.TxOptions{
		Isolation: o.Isolation,
		ReadOnly:  o.ReadOnly,
	}
}

// TransactionManager provides utilities for managing transactions across repositories
type TransactionManager struct {
	db *sqlx.DB
}

func NewTransactionManager(db *sqlx.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

func (tm *TransactionManager) WithTransaction(ctx context.Context, fn func(*sqlx.Tx) error) error {
	return tm.WithTransactionOptions(ctx, nil, fn)
}

func (tm *TransactionManager) WithTransactionOptions(ctx context.Context, opts *TransactionOptions, fn func(*sqlx.Tx) error) error {
	if opts == nil {
		opts = DefaultTransactionOptions()
	}

	txOpts := &sql.TxOptions{
		Isolation: opts.Isolation,
		ReadOnly:  opts.ReadOnly,
	}

	tx, err := tm.db.BeginTxx(ctx, txOpts)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if p := recover(); p != nil {
			if !committed {
				tx.Rollback()
			}
			panic(p)
		}
		if !committed {
			if rbErr := tx.Rollback(); rbErr != nil && rbErr.Error() != "sql: transaction has already been committed or rolled back" {

			}
		}
	}()

	err = fn(tx)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	committed = true

	return nil
}

func (r *Repository[T]) GetTransactionManager() (*TransactionManager, error) {
	db, ok := r.db.(*sqlx.DB)
	if !ok {
		return nil, fmt.Errorf("cannot create transaction manager: repository is already using a transaction")
	}
	return NewTransactionManager(db), nil
}

func (r *Repository[T]) WithinTransaction(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tm, err := r.GetTransactionManager()
	if err != nil {
		return err
	}
	return tm.WithTransaction(ctx, fn)
}

func (r *Repository[T]) IsTransaction() bool {
	_, ok := r.db.(*sqlx.Tx)
	return ok
}
