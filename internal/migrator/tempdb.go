package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type TempDBManager struct {
	baseConfig *DBConfig
}

func NewTempDBManager(baseConfig *DBConfig) *TempDBManager {
	return &TempDBManager{
		baseConfig: baseConfig,
	}
}

func (tm *TempDBManager) CreateTempDB(ctx context.Context, tempDBName string) (*sql.DB, func(), error) {

	adminConfig := &DBConfig{
		URL:             tm.buildAdminDBURL(),
		ConnMaxLifetime: tm.baseConfig.ConnMaxLifetime,
		MaxOpenConns:    tm.baseConfig.MaxOpenConns,
		MaxIdleConns:    tm.baseConfig.MaxIdleConns,
	}

	adminDB, err := adminConfig.Connect(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to admin database: %w", err)
	}

	_, err = adminDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", tempDBName))
	if err != nil {
		adminDB.Close()
		return nil, nil, fmt.Errorf("failed to create temp database %s: %w", tempDBName, err)
	}

	tempDBURL := tm.buildTempDBURL(tempDBName)
	tempConfig := &DBConfig{
		URL:             tempDBURL,
		ConnMaxLifetime: 10 * time.Minute,
		MaxOpenConns:    5,
		MaxIdleConns:    2,
	}

	tempDB, err := tempConfig.Connect(ctx)
	if err != nil {
		tm.cleanupTempDB(ctx, adminDB, tempDBName)
		adminDB.Close()
		return nil, nil, fmt.Errorf("failed to connect to temp database: %w", err)
	}

	cleanup := func() {
		tempDB.Close()
		tm.cleanupTempDB(context.Background(), adminDB, tempDBName)
		adminDB.Close()
	}

	return tempDB, cleanup, nil
}

func (tm *TempDBManager) buildAdminDBURL() string {
	baseURL := tm.baseConfig.URL
	if idx := strings.LastIndex(baseURL, "/"); idx != -1 {
		if queryIdx := strings.Index(baseURL[idx:], "?"); queryIdx != -1 {
			return baseURL[:idx+1] + "postgres" + baseURL[idx+queryIdx:]
		}
		return baseURL[:idx+1] + "postgres"
	}
	return baseURL
}

func (tm *TempDBManager) buildTempDBURL(tempDBName string) string {
	baseURL := tm.baseConfig.URL
	if idx := strings.LastIndex(baseURL, "/"); idx != -1 {
		if queryIdx := strings.Index(baseURL[idx:], "?"); queryIdx != -1 {
			return baseURL[:idx+1] + tempDBName + baseURL[idx+queryIdx:]
		}
		return baseURL[:idx+1] + tempDBName
	}
	return baseURL
}

func (tm *TempDBManager) cleanupTempDB(ctx context.Context, adminDB *sql.DB, tempDBName string) {
	_, _ = adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", tempDBName))
}
