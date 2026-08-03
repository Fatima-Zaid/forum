package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"forum/models"
)


func CreateSession(db *sql.DB, id string, userID int, expiresAt time.Time) error {
	_, err := db.Exec(
		`INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)`,
		id, userID, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}


func GetValidSession(db *sql.DB, sessionID string) (*models.Session, error) {
	row := db.QueryRow(
		`SELECT id, user_id, expires_at, created_at FROM sessions WHERE id = ?`,
		sessionID,
	)
	var s models.Session
	err := row.Scan(&s.ID, &s.UserID, &s.ExpiresAt, &s.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if s.IsExpired() {
		_, _ = db.Exec(`DELETE FROM sessions WHERE id = ?`, sessionID)
		return nil, ErrNotFound
	}
	return &s, nil
}

func DeleteSession(db *sql.DB, sessionID string) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE id = ?`, sessionID)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}


func DeleteExpiredSessions(db *sql.DB) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE expires_at <= ?`, time.Now())
	if err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}
	return nil
}