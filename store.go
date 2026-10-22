package main

import (
	"database/sql"
	"fmt"
)

// LockStore persists per-channel command locks in SQLite.
type LockStore struct {
	db *sql.DB
}

// NewLockStore opens the database and ensures the command_locks table exists.
func NewLockStore(path string) (*LockStore, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	const createTableSQL = `
    CREATE TABLE IF NOT EXISTS command_locks (
        command_name TEXT NOT NULL,
        channel_id TEXT NOT NULL,
        is_locked INTEGER NOT NULL DEFAULT 0,
        PRIMARY KEY (command_name, channel_id)
    );`
	if _, err = db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create command_locks table: %w", err)
	}
	return &LockStore{db: db}, nil
}

// Close releases the underlying database handle.
func (s *LockStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// SetLocked updates the locked status of a command for a channel.
func (s *LockStore) SetLocked(command, channelID string, locked bool) error {
	lockValue := 0
	if locked {
		lockValue = 1
	}
	stmt, err := s.db.Prepare("INSERT OR REPLACE INTO command_locks (command_name, channel_id, is_locked) VALUES (?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	if _, err = stmt.Exec(command, channelID, lockValue); err != nil {
		return fmt.Errorf("failed to set command lock status for channel %s: %w", channelID, err)
	}
	return nil
}

// IsLocked reports whether a command is locked in a channel. Commands with no
// row are considered unlocked.
func (s *LockStore) IsLocked(command, channelID string) (bool, error) {
	var locked int
	row := s.db.QueryRow("SELECT is_locked FROM command_locks WHERE command_name = ? AND channel_id = ?", command, channelID)
	switch err := row.Scan(&locked); err {
	case nil:
		return locked == 1, nil
	case sql.ErrNoRows:
		return false, nil
	default:
		return false, fmt.Errorf("failed to query command lock status: %w", err)
	}
}
