package localstate

import (
	"database/sql"
	"time"
)

// EnsureDefaultUser inserts a single default user for local environment if the Users table is empty.
// No-op if any user already exists.
func EnsureDefaultUser(db *sql.DB) error {
	var cnt int
	if err := db.QueryRow(`SELECT COUNT(1) FROM Users`).Scan(&cnt); err != nil {
		return err
	}
	if cnt > 0 {
		return nil
	}
	now := time.Now().UTC()
	_, err := db.Exec(`INSERT INTO Users (UserId, Email, DisplayName, TimeZone, Status, CreationTime) VALUES (?,?,?,?,?,?)`,
		"local_user", "dev@localhost", "Local Developer", "UTC", "ACTIVE", now)
	return err
}
