package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Level represents the logging level
type Level int

const (
	// DebugLevel logs everything
	DebugLevel Level = iota
	// InfoLevel logs info, warnings, and errors
	InfoLevel
	// WarnLevel logs warnings and errors
	WarnLevel
	// ErrorLevel logs only errors
	ErrorLevel
	// SilentLevel logs nothing
	SilentLevel
)

// Logger is the main logger interface
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Fatal(format string, args ...interface{})

	// Structured logging
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger

	// Progress indicators
	StartProgress(message string)
	UpdateProgress(message string)
	EndProgress(success bool)

	// Set output writer
	SetOutput(w io.Writer)
	SetLevel(level Level)
}

// defaultLogger implements the Logger interface
type defaultLogger struct {
	level      Level
	output     io.Writer
	fields     map[string]interface{}
	prefix     string
	inProgress bool
}

var (
	// Global logger instance
	global Logger = &defaultLogger{
		level:  InfoLevel,
		output: os.Stdout,
		fields: make(map[string]interface{}),
	}
)

// SetGlobalLogger sets the global logger instance
func SetGlobalLogger(logger Logger) {
	global = logger
}

// SetLevel sets the global logging level
func SetLevel(level Level) {
	global.SetLevel(level)
}

// SetVerbose enables verbose logging (debug level)
func SetVerbose(verbose bool) {
	if verbose {
		global.SetLevel(DebugLevel)
	}
}

// Debug logs a debug message
func Debug(format string, args ...interface{}) {
	global.Debug(format, args...)
}

// Info logs an info message
func Info(format string, args ...interface{}) {
	global.Info(format, args...)
}

// Warn logs a warning message
func Warn(format string, args ...interface{}) {
	global.Warn(format, args...)
}

// Error logs an error message
func Error(format string, args ...interface{}) {
	global.Error(format, args...)
}

// Fatal logs a fatal message and exits
func Fatal(format string, args ...interface{}) {
	global.Fatal(format, args...)
}

// WithField adds a field to the logger
func WithField(key string, value interface{}) Logger {
	return global.WithField(key, value)
}

// WithFields adds multiple fields to the logger
func WithFields(fields map[string]interface{}) Logger {
	return global.WithFields(fields)
}

// StartProgress starts a progress indicator
func StartProgress(message string) {
	global.StartProgress(message)
}

// UpdateProgress updates the progress message
func UpdateProgress(message string) {
	global.UpdateProgress(message)
}

// EndProgress ends the progress indicator
func EndProgress(success bool) {
	global.EndProgress(success)
}

// Implementation of defaultLogger methods

func (l *defaultLogger) SetOutput(w io.Writer) {
	l.output = w
}

func (l *defaultLogger) SetLevel(level Level) {
	l.level = level
}

func (l *defaultLogger) Debug(format string, args ...interface{}) {
	if l.level <= DebugLevel {
		l.log("DEBUG", format, args...)
	}
}

func (l *defaultLogger) Info(format string, args ...interface{}) {
	if l.level <= InfoLevel {
		l.log("INFO", format, args...)
	}
}

func (l *defaultLogger) Warn(format string, args ...interface{}) {
	if l.level <= WarnLevel {
		l.log("WARN", format, args...)
	}
}

func (l *defaultLogger) Error(format string, args ...interface{}) {
	if l.level <= ErrorLevel {
		l.log("ERROR", format, args...)
	}
}

func (l *defaultLogger) Fatal(format string, args ...interface{}) {
	l.log("FATAL", format, args...)
	os.Exit(1)
}

func (l *defaultLogger) WithField(key string, value interface{}) Logger {
	newFields := make(map[string]interface{}, len(l.fields)+1)
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &defaultLogger{
		level:  l.level,
		output: l.output,
		fields: newFields,
		prefix: l.prefix,
	}
}

func (l *defaultLogger) WithFields(fields map[string]interface{}) Logger {
	newFields := make(map[string]interface{}, len(l.fields)+len(fields))
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &defaultLogger{
		level:  l.level,
		output: l.output,
		fields: newFields,
		prefix: l.prefix,
	}
}

func (l *defaultLogger) StartProgress(message string) {
	if l.level <= InfoLevel {
		l.inProgress = true
		fmt.Fprintf(l.output, "⏳ %s...", message)
	}
}

func (l *defaultLogger) UpdateProgress(message string) {
	if l.level <= InfoLevel && l.inProgress {
		fmt.Fprintf(l.output, "\r⏳ %s...", message)
	}
}

func (l *defaultLogger) EndProgress(success bool) {
	if l.level <= InfoLevel && l.inProgress {
		l.inProgress = false
		if success {
			fmt.Fprintf(l.output, "\r✅\n")
		} else {
			fmt.Fprintf(l.output, "\r❌\n")
		}
	}
}

func (l *defaultLogger) log(level, format string, args ...interface{}) {
	if l.inProgress {
		fmt.Fprintf(l.output, "\r")
		l.inProgress = false
	}

	timestamp := time.Now().Format("15:04:05")
	message := fmt.Sprintf(format, args...)

	var fieldStr string
	if len(l.fields) > 0 {
		var fields []string
		for k, v := range l.fields {
			fields = append(fields, fmt.Sprintf("%s=%v", k, v))
		}
		fieldStr = " " + strings.Join(fields, " ")
	}

	var levelColor string
	switch level {
	case "DEBUG":
		levelColor = "\033[36m"
	case "INFO":
		levelColor = "\033[32m"
	case "WARN":
		levelColor = "\033[33m"
	case "ERROR", "FATAL":
		levelColor = "\033[31m"
	}

	reset := "\033[0m"

	fmt.Fprintf(l.output, "[%s] %s%s%s %s%s\n",
		timestamp, levelColor, level, reset, message, fieldStr)
}

// ParseLevel parses a string into a Level
func ParseLevel(level string) Level {
	switch strings.ToLower(level) {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	case "silent":
		return SilentLevel
	default:
		return InfoLevel
	}
}
