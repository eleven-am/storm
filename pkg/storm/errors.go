package storm

import (
	"errors"
	"fmt"
)

// Common errors
var (
	ErrClosed            = errors.New("storm: connection closed")
	ErrNoConnection      = errors.New("storm: no database connection")
	ErrInvalidConfig     = errors.New("storm: invalid configuration")
	ErrMigrationFailed   = errors.New("storm: migration failed")
	ErrSchemaInvalid     = errors.New("storm: invalid schema")
	ErrNotImplemented    = errors.New("storm: not implemented")
	ErrMigrationExists   = errors.New("storm: migration already exists")
	ErrMigrationNotFound = errors.New("storm: migration not found")
	ErrDestructiveChange = errors.New("storm: destructive change detected")
)

// ErrorType represents the type of error
type ErrorType string

const (
	ErrorTypeConnection ErrorType = "connection"
	ErrorTypeConfig     ErrorType = "config"
	ErrorTypeMigration  ErrorType = "migration"
	ErrorTypeSchema     ErrorType = "schema"
	ErrorTypeORM        ErrorType = "orm"
	ErrorTypeGeneration ErrorType = "generation"
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypeUnknown    ErrorType = "unknown"
)

// Error represents a Storm error with context
type Error struct {
	Type    ErrorType
	Op      string
	Err     error
	Details map[string]interface{}
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Op != "" {
		return fmt.Sprintf("storm %s: %s: %v", e.Type, e.Op, e.Err)
	}
	return fmt.Sprintf("storm %s: %v", e.Type, e.Err)
}

// Unwrap implements errors.Unwrap
func (e *Error) Unwrap() error {
	return e.Err
}

// Is implements errors.Is
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Type == t.Type
}

// NewError creates a new Storm error
func NewError(errType ErrorType, op string, err error) *Error {
	return &Error{
		Type:    errType,
		Op:      op,
		Err:     err,
		Details: make(map[string]interface{}),
	}
}

// WithDetails adds details to the error
func (e *Error) WithDetails(key string, value interface{}) *Error {
	e.Details[key] = value
	return e
}

// NewConnectionError creates a connection error
func NewConnectionError(op string, err error) *Error {
	return NewError(ErrorTypeConnection, op, err)
}

// NewConfigError creates a configuration error
func NewConfigError(op string, err error) *Error {
	return NewError(ErrorTypeConfig, op, err)
}

// NewMigrationError creates a migration error
func NewMigrationError(op string, err error) *Error {
	return NewError(ErrorTypeMigration, op, err)
}

// NewSchemaError creates a schema error
func NewSchemaError(op string, err error) *Error {
	return NewError(ErrorTypeSchema, op, err)
}

// NewORMError creates an ORM error
func NewORMError(op string, err error) *Error {
	return NewError(ErrorTypeORM, op, err)
}

// NewGenerationError creates a generation error
func NewGenerationError(op string, err error) *Error {
	return NewError(ErrorTypeGeneration, op, err)
}

// NewValidationError creates a validation error
func NewValidationError(op string, err error) *Error {
	return NewError(ErrorTypeValidation, op, err)
}
