package database

import (
	"fmt"
	"io/fs"
	"log"
	"sort"
	"time"

	"ayam-batu-api/internal/config"
	"ayam-batu-api/migrations"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect opens a PostgreSQL connection via GORM, retrying briefly while a fresh database boots.
func Connect(cfg *config.Config) (*gorm.DB, error) {
	logLevel := logger.Warn
	if cfg.GinMode == "debug" {
		logLevel = logger.Info
	}

	const maxAttempts = 6
	var db *gorm.DB
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		db, err = gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
			Logger: logger.Default.LogMode(logLevel),
		})
		if err == nil {
			return db, nil
		}
		if attempt < maxAttempts {
			log.Printf("waiting for postgres (%d/%d): %v", attempt, maxAttempts, err)
			time.Sleep(2 * time.Second)
		}
	}

	return nil, fmt.Errorf("connect postgres: %w", err)
}

// RunMigrations executes all embedded SQL migration scripts in lexical order.
func RunMigrations(db *gorm.DB) error {
	entries, err := fs.ReadDir(migrations.Files, ".")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}

	var sqlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && len(entry.Name()) > 4 && entry.Name()[len(entry.Name())-4:] == ".sql" {
			sqlFiles = append(sqlFiles, entry.Name())
		}
	}
	sort.Strings(sqlFiles)

	for _, name := range sqlFiles {
		content, err := fs.ReadFile(migrations.Files, name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if err := db.Exec(string(content)).Error; err != nil {
			return fmt.Errorf("execute migration %s: %w", name, err)
		}
	}

	return nil
}
