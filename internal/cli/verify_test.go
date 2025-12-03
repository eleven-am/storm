package cli

import (
	"strings"
	"testing"
)

func TestRunVerify(t *testing.T) {

	origDbURL := dbURL
	origDbUser := dbUser
	origDbName := dbName
	origDbPassword := dbPassword
	origDbHost := dbHost
	origDbPort := dbPort
	origDbSSLMode := dbSSLMode
	origPackagePath := packagePath
	origDebug := debug
	defer func() {
		dbURL = origDbURL
		dbUser = origDbUser
		dbName = origDbName
		dbPassword = origDbPassword
		dbHost = origDbHost
		dbPort = origDbPort
		dbSSLMode = origDbSSLMode
		packagePath = origPackagePath
		debug = origDebug
	}()

	t.Run("fails when no database credentials provided", func(t *testing.T) {

		dbURL = ""
		dbUser = ""
		dbName = ""
		dbPassword = ""
		dbHost = "localhost"
		dbPort = "5432"
		dbSSLMode = "disable"
		packagePath = "./models"
		debug = false

		err := runVerify(verifyCmd, []string{})
		if err == nil {
			t.Error("expected error when no credentials provided")
		}
		if !strings.Contains(err.Error(), "either --url or both --user and --dbname must be provided") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("fails when only user provided but no dbname", func(t *testing.T) {

		dbURL = ""
		dbUser = "testuser"
		dbName = ""
		dbPassword = "password"
		dbHost = "localhost"
		dbPort = "5432"
		dbSSLMode = "disable"
		packagePath = "./models"
		debug = false

		err := runVerify(verifyCmd, []string{})
		if err == nil {
			t.Error("expected error when only user provided but no dbname")
		}
		if !strings.Contains(err.Error(), "either --url or both --user and --dbname must be provided") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("fails when only dbname provided but no user", func(t *testing.T) {

		dbURL = ""
		dbUser = ""
		dbName = "testdb"
		dbPassword = "password"
		dbHost = "localhost"
		dbPort = "5432"
		dbSSLMode = "disable"
		packagePath = "./models"
		debug = false

		err := runVerify(verifyCmd, []string{})
		if err == nil {
			t.Error("expected error when only dbname provided but no user")
		}
		if !strings.Contains(err.Error(), "either --url or both --user and --dbname must be provided") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("fails with invalid database URL", func(t *testing.T) {

		dbURL = "invalid://url"
		dbUser = ""
		dbName = ""
		dbPassword = ""
		dbHost = "localhost"
		dbPort = "5432"
		dbSSLMode = "disable"
		packagePath = "./models"
		debug = false

		err := runVerify(verifyCmd, []string{})
		if err == nil {
			t.Error("expected error with invalid database URL")
		}

		if !strings.Contains(err.Error(), "failed to") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("fails with unreachable database", func(t *testing.T) {

		dbURL = "postgres://testuser:password@unreachable:5432/testdb?sslmode=disable"
		dbUser = ""
		dbName = ""
		dbPassword = ""
		dbHost = "localhost"
		dbPort = "5432"
		dbSSLMode = "disable"
		packagePath = "./models"
		debug = false

		err := runVerify(verifyCmd, []string{})
		if err == nil {
			t.Error("expected error with unreachable database")
		}

		if !strings.Contains(err.Error(), "failed to") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("builds correct DSN from individual parameters", func(t *testing.T) {

		dbURL = ""
		dbUser = "testuser"
		dbName = "testdb"
		dbPassword = "password"
		dbHost = "localhost"
		dbPort = "5432"
		dbSSLMode = "disable"
		packagePath = "./models"
		debug = false

		err := runVerify(verifyCmd, []string{})
		if err == nil {
			t.Error("expected error due to connection failure")
		}

		if strings.Contains(err.Error(), "either --url or both --user and --dbname must be provided") {
			t.Error("should not fail on credential validation with valid user and dbname")
		}
	})
}

func TestVerifyCommand(t *testing.T) {
	t.Run("command structure", func(t *testing.T) {
		if verifyCmd.Use != "verify" {
			t.Errorf("expected Use to be 'verify', got %s", verifyCmd.Use)
		}

		if verifyCmd.Short != "Verify database schema matches models" {
			t.Errorf("expected Short to be 'Verify database schema matches models', got %s", verifyCmd.Short)
		}

		if verifyCmd.RunE == nil {
			t.Error("expected RunE to be set")
		}
	})

	t.Run("command flags", func(t *testing.T) {
		expectedFlags := []string{
			"url",
			"host",
			"port",
			"user",
			"password",
			"dbname",
			"sslmode",
			"package",
		}

		for _, flagName := range expectedFlags {
			flag := verifyCmd.Flags().Lookup(flagName)
			if flag == nil {
				t.Errorf("expected flag %s to be defined", flagName)
			}
		}

		hostFlag := verifyCmd.Flags().Lookup("host")
		if hostFlag != nil && hostFlag.DefValue != "localhost" {
			t.Errorf("expected host flag default to be 'localhost', got %s", hostFlag.DefValue)
		}

		portFlag := verifyCmd.Flags().Lookup("port")
		if portFlag != nil && portFlag.DefValue != "5432" {
			t.Errorf("expected port flag default to be '5432', got %s", portFlag.DefValue)
		}

		sslmodeFlag := verifyCmd.Flags().Lookup("sslmode")
		if sslmodeFlag != nil && sslmodeFlag.DefValue != "disable" {
			t.Errorf("expected sslmode flag default to be 'disable', got %s", sslmodeFlag.DefValue)
		}

		packageFlag := verifyCmd.Flags().Lookup("package")
		if packageFlag != nil && packageFlag.DefValue != "./models" {
			t.Errorf("expected package flag default to be './models', got %s", packageFlag.DefValue)
		}
	})
}
