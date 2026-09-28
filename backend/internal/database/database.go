package database

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// driverName is modernc.org/sqlite's registered name: a pure-Go SQLite, so
// the binary cross-compiles with CGO_ENABLED=0 and ships as a single file.
const driverName = "sqlite"

func init() {
	// sqlx only knows "sqlite3" as a ?-placeholder driver; without this,
	// Rebind (used after sqlx.In) wouldn't know how to treat "sqlite".
	sqlx.BindDriver(driverName, sqlx.QUESTION)
}

// Connect opens (creating if needed) the SQLite database at path.
func Connect(path string) (*sqlx.DB, error) {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create database dir: %w", err)
		}
	}

	db, err := sqlx.Connect(driverName, dsn(path))
	if err != nil {
		return nil, fmt.Errorf("connect sqlite: %w", err)
	}

	// SQLite has a single writer; WAL lets readers proceed alongside it and
	// busy_timeout queues writers instead of failing them. A handful of
	// connections is plenty and keeps memory flat.
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(0)

	return db, nil
}

func dsn(path string) string {
	q := url.Values{}
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "synchronous(NORMAL)")
	q.Add("_pragma", "busy_timeout(5000)")
	// Every transaction takes the write lock up front. This is what stands in
	// for MySQL's SELECT ... FOR UPDATE: two concurrent read-modify-write
	// transactions (e.g. license activation) serialize
	// instead of one failing with SQLITE_BUSY halfway through.
	q.Set("_txlock", "immediate")
	// Write time.Time as "2006-01-02 15:04:05.999999999-07:00" — the format
	// SQLite's own date functions understand and that sorts correctly as
	// text next to CURRENT_TIMESTAMP defaults. All times are written in UTC.
	q.Set("_time_format", "sqlite")
	return "file:" + filepath.ToSlash(path) + "?" + q.Encode()
}

// IsUniqueViolation reports whether err is a UNIQUE or PRIMARY KEY
// constraint failure — the equivalent of MySQL's error 1062.
func IsUniqueViolation(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	code := sqliteErr.Code()
	return code == sqlite3.SQLITE_CONSTRAINT_UNIQUE || code == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY
}

// Backup writes a consistent snapshot of the open database to dest. Safe to
// run while the app is serving traffic, unlike copying a WAL-mode file.
func Backup(db *sqlx.DB, dest string) error {
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("backup target %s already exists", dest)
	}
	if _, err := db.Exec(`VACUUM INTO ?`, dest); err != nil {
		return fmt.Errorf("vacuum into: %w", err)
	}
	return nil
}
