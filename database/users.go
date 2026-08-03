package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"forum/models"
)

var ErrEmailTaken = errors.New("email already registered")
var ErrNotFound = errors.New("not found")

func CreateUser(db *sql.DB, username, email, passwordHash string) (int, error) {
	res, err := db.Exec(
		`INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)`,
		username, email, passwordHash,
	)
	if err != nil {
		if isUniqueConstraintErr(err) {
			return 0, ErrEmailTaken
		}
		return 0, fmt.Errorf("create user: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func GetUserByEmail(db *sql.DB, email string) (*models.User, error) {
	row := db.QueryRow(
		`SELECT id, username, email, password_hash, created_at FROM users WHERE email = ?`,
		email,
	)
	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &u, nil
}

func GetUserByID(db *sql.DB, id int) (*models.User, error) {
	row := db.QueryRow(
		`SELECT id, username, email, password_hash, created_at FROM users WHERE id = ?`,
		id,
	)
	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}


func EmailExists(db *sql.DB, email string) (bool, error) {
	var exists bool
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)`, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check email exists: %w", err)
	}
	return exists, nil
}

func isUniqueConstraintErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}