package database

import (
	"database/sql"
	"fmt"

	"forum/models"
)

func GetAllCategories(db *sql.DB) ([]models.Category, error) {
	rows, err := db.Query(`SELECT id, name FROM categories ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("get categories: %w", err)
	}
	defer rows.Close()

	var cats []models.Category
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}


func GetOrCreateCategory(db *sql.DB, name string) (int, error) {
	var id int
	err := db.QueryRow(`SELECT id FROM categories WHERE name = ?`, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("lookup category: %w", err)
	}
	res, err := db.Exec(`INSERT INTO categories (name) VALUES (?)`, name)
	if err != nil {
		return 0, fmt.Errorf("create category: %w", err)
	}
	lastID, err := res.LastInsertId()
	return int(lastID), err
}