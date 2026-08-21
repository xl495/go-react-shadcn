package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(path string) (*gorm.DB, error) {
	return openSQLite(path, sqlitePoolSize(path))
}

func OpenExtra(path string) (*gorm.DB, error) {
	if !sharedSQLite(path) {
		return nil, nil
	}
	return openSQLite(path, 2)
}

func sharedSQLite(path string) bool {
	if path == "" || path == ":memory:" {
		return false
	}
	return !strings.Contains(path, "mode=memory")
}

func sqlitePoolSize(path string) int {
	if !sharedSQLite(path) {
		return 1
	}
	return 4
}

func openSQLite(path string, maxOpen int) (*gorm.DB, error) {
	if path != ":memory:" && path != "" {
		dir := filepath.Dir(path)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0o750); err != nil {
				return nil, fmt.Errorf("create db dir: %w", err)
			}
		}
	}
	dsn := path
	if path != ":memory:" && path != "" && !strings.Contains(path, "?") {
		dsn = "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)"
	}
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if maxOpen < 1 {
		maxOpen = 1
	}
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxOpen)
	return db, nil
}

func Close(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
