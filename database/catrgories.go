package database

import (
	"database/sql"
	"fmt"
	"strings"

	"forum/models"
)

func GetAllCategories(db *sql.DB) ([]models.Category, error) { //get the DB connection and returns slice of category object or error
	rows, err := db.Query(`SELECT id, name FROM categories ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("get categories: %w", err)
	}
	defer rows.Close() // when the func finish close the stream and clean up rows object, defer lets us write the close write here
	// instead of writting it at the very end of func, it does not close immediately it waits til the func finish.

	var cats []models.Category
	for rows.Next() { // if there is next row after this current row then keep looping
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil { // Scan() is func from "database/sql", it reads the row and copy it, we cant use copy() on a row
			return nil, fmt.Errorf("scan category: %w", err) // Scan() copy data and return the err
		}
		cats = append(cats, c)
	}
	return cats, rows.Err() // check and return the err if any err happend in loop while readin the row
}

func GetOrCreateCategory(db *sql.DB, name string) (int, error) { // return the id of cat and the err
	name = strings.TrimSpace(name) // to not let user create the cat with spaces
	if name == "" {
		return 0, fmt.Errorf("category name is empty") // for a case where use write only spaces and after trim is it empty str or not
	}

	var id int
	err := db.QueryRow(`SELECT id FROM categories WHERE name = ? COLLATE NOCASE`, name).Scan(&id) // get the id of cat by name, expects one row, check if it is exist in db and copy id
	if err == nil {                                                                               // COLLATE NOCASE means to ignore the uppercase and lowercase
		return id, nil // case 1: cat found, func ends
	}
	// case 2: cat not exist, err == sql.ErrNoRows ,actual condition "sql.ErrNoRows != sql.ErrNoRows" then never enter the if

	if err != sql.ErrNoRows { // case 3: err = db is locked stop the func
		return 0, fmt.Errorf("lookup category: %w", err)
	}

	// for case 2 we come here and insert the new cat in db
	res, err := db.Exec(`INSERT INTO categories (name) VALUES (?)`, name) // Exec() return err and result
	if err != nil {
		return 0, fmt.Errorf("create category: %w", err)
	}

	lastID, err := res.LastInsertId() // the cat will be the last one in table, get the last id of it
	return int(lastID), err
}
