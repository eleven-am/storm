package cli

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/eleven-am/storm/internal/logger"
	"github.com/eleven-am/storm/internal/migrator"
	"github.com/eleven-am/storm/pkg/storm"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
)

var (
	dbURL      string
	dbHost     string
	dbPort     string
	dbUser     string
	dbPassword string
	dbName     string
	dbSSLMode  string

	outputDir           string
	migratePackagePath  string
	migrationName       string
	dryRun              bool
	createDBIfNotExists bool
	allowDestructive    bool
	pushToDB            bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Generate database migrations",
	Long: `Compare current Go structs with database schema and generate migration files.
Uses Storm's migration engine for schema comparison and migration generation.`,
	RunE: runMigrate,
}

func init() {
	migrateCmd.Flags().StringVar(&dbHost, "host", "localhost", "Database host")
	migrateCmd.Flags().StringVar(&dbPort, "port", "5432", "Database port")
	migrateCmd.Flags().StringVar(&dbUser, "user", "", "Database user")
	migrateCmd.Flags().StringVar(&dbPassword, "password", "", "Database password")
	migrateCmd.Flags().StringVar(&dbName, "dbname", "", "Database name")
	migrateCmd.Flags().StringVar(&dbSSLMode, "sslmode", "disable", "SSL mode (disable, require, verify-ca, verify-full)")

	migrateCmd.Flags().StringVar(&outputDir, "output", "", "Output directory for migration files")
	migrateCmd.Flags().StringVar(&migratePackagePath, "package", "", "Path to package containing models")
	migrateCmd.Flags().StringVar(&migrationName, "name", "", "Migration name (optional)")
	migrateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print migration without creating files")
	migrateCmd.Flags().BoolVar(&createDBIfNotExists, "create-if-not-exists", false, "Create the database if it does not exist")
	migrateCmd.Flags().BoolVar(&allowDestructive, "allow-destructive", false, "Allow potentially destructive operations")
	migrateCmd.Flags().BoolVar(&pushToDB, "push", false, "Execute the generated SQL directly on the database")
}

func runMigrate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if stormConfig != nil {
		if outputDir == "" && stormConfig.Migrations.Directory != "" {
			outputDir = stormConfig.Migrations.Directory
		}
		if migratePackagePath == "" && stormConfig.Models.Package != "" {
			migratePackagePath = stormConfig.Models.Package
		}
	}

	if outputDir == "" {
		outputDir = "./migrations"
	}
	if migratePackagePath == "" {
		migratePackagePath = "./models"
	}

	var dsn string
	if databaseURL != "" {
		dsn = databaseURL
	} else if dbUser != "" && dbName != "" {
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			dbUser, dbPassword, dbHost, dbPort, dbName, dbSSLMode)
	} else {
		return fmt.Errorf("database connection required: use --url flag, individual connection flags, or specify in storm.yaml")
	}

	logger.CLI().Debug("Using database URL: %s", dsn)
	logger.CLI().Debug("Models package: %s", migratePackagePath)
	logger.CLI().Debug("Output directory: %s", outputDir)

	if createDBIfNotExists {
		logger.CLI().Info("Checking if database exists...")
		if err := ensureDatabaseExistsFromURL(ctx, dsn); err != nil {
			return fmt.Errorf("failed to ensure database exists: %w", err)
		}
	}

	logger.CLI().Info("Initializing Storm migration engine...")

	config := storm.NewConfig()
	config.DatabaseURL = dsn
	config.ModelsPackage = migratePackagePath
	config.MigrationsDir = outputDir
	config.Debug = debug

	stormClient, err := storm.NewWithConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Storm client: %w", err)
	}
	defer stormClient.Close()

	if !createDBIfNotExists {
		logger.CLI().Debug("Pinging database to verify connection...")
		if err := stormClient.Ping(ctx); err != nil {
			return fmt.Errorf("failed to ping database: %w", err)
		}
		logger.CLI().Debug("Database connection verified")
	}

	logger.CLI().Info("Generating migration...")

	opts := storm.MigrateOptions{
		PackagePath:         migratePackagePath,
		OutputDir:           outputDir,
		DryRun:              dryRun,
		CreateDBIfNotExists: createDBIfNotExists,
	}

	if pushToDB {

		logger.CLI().Info("Generating and applying migration directly to database...")
		return executePushMigration(ctx, config, createDBIfNotExists, allowDestructive, migratePackagePath)
	}

	if err := stormClient.Migrate(ctx, opts); err != nil {
		return fmt.Errorf("failed to generate migration: %w", err)
	}

	if dryRun {
		logger.CLI().Info("Migration generated (dry run)")
	} else {
		logger.CLI().Info("Migration files generated successfully")
		logger.CLI().Info("Run 'storm migrate --push' to apply the migrations")
	}

	return nil
}

// ensureDatabaseExistsFromURL creates the database if it doesn't exist
func ensureDatabaseExistsFromURL(ctx context.Context, databaseURL string) error {
	dbName := extractDatabaseNameFromURL(databaseURL)
	if dbName == "" {
		return fmt.Errorf("could not extract database name from URL")
	}

	adminURL := buildAdminDatabaseURLFromURL(databaseURL)

	adminDB, err := sql.Open("postgres", adminURL)
	if err != nil {
		return fmt.Errorf("failed to open admin database connection: %w", err)
	}
	defer adminDB.Close()

	if err := adminDB.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping admin database: %w", err)
	}

	var exists bool
	checkSQL := "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)"
	err = adminDB.QueryRowContext(ctx, checkSQL, dbName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if database exists: %w", err)
	}

	if !exists {

		createSQL := fmt.Sprintf("CREATE DATABASE %s", quoteIdentifierCLI(dbName))
		logger.DB().Info("Creating database: %s", dbName)

		if _, err := adminDB.ExecContext(ctx, createSQL); err != nil {
			return fmt.Errorf("failed to create database %s: %w", dbName, err)
		}
	} else {
		logger.DB().Info("Database %s already exists", dbName)
	}

	return nil
}

// extractDatabaseNameFromURL extracts the database name from a database URL
func extractDatabaseNameFromURL(databaseURL string) string {
	if strings.HasPrefix(databaseURL, "postgres://") || strings.HasPrefix(databaseURL, "postgresql://") {
		parts := strings.Split(databaseURL, "/")
		if len(parts) >= 4 {
			dbPart := parts[len(parts)-1]
			if idx := strings.Index(dbPart, "?"); idx != -1 {
				return dbPart[:idx]
			}
			return dbPart
		}
	}
	return ""
}

// buildAdminDatabaseURLFromURL builds a URL for connecting to the admin database
func buildAdminDatabaseURLFromURL(databaseURL string) string {
	if strings.HasPrefix(databaseURL, "postgres://") || strings.HasPrefix(databaseURL, "postgresql://") {
		parts := strings.Split(databaseURL, "/")
		if len(parts) >= 4 {

			dbPart := parts[len(parts)-1]
			if idx := strings.Index(dbPart, "?"); idx != -1 {
				queryPart := dbPart[idx:]
				parts[len(parts)-1] = "postgres" + queryPart
			} else {
				parts[len(parts)-1] = "postgres"
			}
			return strings.Join(parts, "/")
		}
	}
	return databaseURL
}

// quoteIdentifierCLI properly quotes PostgreSQL identifiers
func quoteIdentifierCLI(name string) string {
	return fmt.Sprintf("\"%s\"", name)
}

// executePushMigration executes migration directly using Atlas migrator
func executePushMigration(ctx context.Context, config *storm.Config, createDBIfNotExists bool, allowDestructive bool, packagePath string) error {
	logger.CLI().Info("Executing push migration...")

	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	dbConfig := migrator.NewDBConfig(config.DatabaseURL)
	atlasMigrator := migrator.NewAtlasMigrator(dbConfig)

	opts := migrator.MigrationOptions{
		PackagePath:         packagePath,
		OutputDir:           "",
		DryRun:              false,
		AllowDestructive:    allowDestructive,
		PushToDB:            true,
		CreateDBIfNotExists: createDBIfNotExists,
	}

	result, err := atlasMigrator.GenerateMigration(ctx, db, opts)
	if err != nil {
		return fmt.Errorf("failed to execute push migration: %w", err)
	}

	if len(result.Changes) == 0 {
		logger.CLI().Info("No schema changes detected! Database is up to date.")
	}

	return nil
}
