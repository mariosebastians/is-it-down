package db

import (
	"context"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(ctx context.Context, databaseURL string) (*gorm.DB, error) {
	gdb, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("get underlying sql.DB: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return gdb, nil
}

// AutoMigrate enables the extensions the schema depends on and syncs the
// given models into the database. Safe to call from multiple processes.
func AutoMigrate(gdb *gorm.DB, models ...any) error {
	if err := gdb.Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto`).Error; err != nil {
		return fmt.Errorf("enable pgcrypto extension: %w", err)
	}
	if err := gdb.AutoMigrate(models...); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}
