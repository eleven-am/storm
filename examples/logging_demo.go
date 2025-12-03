package main

import (
	"github.com/eleven-am/storm/internal/logger"
)

func main() {

	logger.Debug("This is a debug message - only shown with --verbose")
	logger.Info("This is an info message - shown with --debug or --verbose")
	logger.Warn("This is a warning - always shown unless silent")
	logger.Error("This is an error - always shown")

	logger.Schema().Debug("Processing table schema")
	logger.SQL().Info("Generated CREATE TABLE statement")
	logger.Migration().Warn("Found potentially destructive change")
	logger.Atlas().Error("Failed to connect to database")

	logger.WithField("table", "users").Info("Processing table")
	logger.WithFields(map[string]interface{}{
		"table":       "users",
		"columns":     5,
		"constraints": 2,
	}).Debug("Table details")

	logger.StartProgress("Generating migrations")

	logger.UpdateProgress("Processing table: users")

	logger.UpdateProgress("Processing table: posts")

	logger.EndProgress(true)

}
