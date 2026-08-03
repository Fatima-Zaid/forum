package database

import (
	"database/sql"
	"errors"
	"fmt"

	"forum/models"
)


func CreatePost(db *sql.DB, userID int, title, gameTitle, content string, categoryIDs []int) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() 

	res, err := tx.Exec(
		`INSERT INTO posts (user_id, title, game_title, content) VALUES (?, ?, ?, ?)`,
		userID, title, gameTitle, content,
	)
	if err != nil {
		return 0, fmt.Errorf("insert post: %w", err)
	}
	postID64, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get post id: %w", err)
	}
	postID := int(postID64)

	for _, catID := range categoryIDs {
		if _, err := tx.Exec(
			`INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)`,
			postID, catID,
		); err != nil {
			return 0, fmt.Errorf("link category %d: %w", catID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}
	return postID, nil
}


const basePostSelect = `
	SELECT
		p.id, p.user_id, u.username, p.title, p.game_title, p.content, p.created_at,
		(SELECT COUNT(*) FROM post_reactions r WHERE r.post_id = p.id AND r.type = 'like') AS like_count,
		(SELECT COUNT(*) FROM post_reactions r WHERE r.post_id = p.id AND r.type = 'dislike') AS dislike_count,
		(SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id) AS comment_count
	FROM posts p
	JOIN users u ON u.id = p.user_id
`

func scanPost(row interface{ Scan(...any) error }) (models.Post, error) {
	var p models.Post
	err := row.Scan(
		&p.ID, &p.UserID, &p.Username, &p.Title, &p.GameTitle, &p.Content, &p.CreatedAt,
		&p.LikeCount, &p.DislikeCount, &p.CommentCount,
	)
	return p, err
}

func GetPostByID(db *sql.DB, postID int) (*models.Post, error) {
	row := db.QueryRow(basePostSelect+` WHERE p.id = ?`, postID)
	p, err := scanPost(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get post: %w", err)
	}

	cats, err := getCategoriesForPost(db, postID)
	if err != nil {
		return nil, err
	}
	p.Categories = cats
	return &p, nil
}

func GetAllPosts(db *sql.DB) ([]models.Post, error) {
	rows, err := db.Query(basePostSelect + ` ORDER BY p.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("get all posts: %w", err)
	}
	return collectPosts(db, rows)
}

func GetPostsByCategory(db *sql.DB, categoryID int) ([]models.Post, error) {
	rows, err := db.Query(
		basePostSelect+`
		JOIN post_categories pc ON pc.post_id = p.id
		WHERE pc.category_id = ?
		ORDER BY p.created_at DESC`,
		categoryID,
	)
	if err != nil {
		return nil, fmt.Errorf("get posts by category: %w", err)
	}
	return collectPosts(db, rows)
}


func GetPostsByUser(db *sql.DB, userID int) ([]models.Post, error) {
	rows, err := db.Query(
		basePostSelect+` WHERE p.user_id = ? ORDER BY p.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get posts by user: %w", err)
	}
	return collectPosts(db, rows)
}


func GetPostsLikedByUser(db *sql.DB, userID int) ([]models.Post, error) {
	rows, err := db.Query(
		basePostSelect+`
		JOIN post_reactions r ON r.post_id = p.id
		WHERE r.user_id = ? AND r.type = 'like'
		ORDER BY p.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get liked posts: %w", err)
	}
	return collectPosts(db, rows)
}


func collectPosts(db *sql.DB, rows *sql.Rows) ([]models.Post, error) {
	defer rows.Close()
	var posts []models.Post
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}
		cats, err := getCategoriesForPost(db, p.ID)
		if err != nil {
			return nil, err
		}
		p.Categories = cats
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

func getCategoriesForPost(db *sql.DB, postID int) ([]models.Category, error) {
	rows, err := db.Query(
		`SELECT c.id, c.name FROM categories c
		 JOIN post_categories pc ON pc.category_id = c.id
		 WHERE pc.post_id = ?`,
		postID,
	)
	if err != nil {
		return nil, fmt.Errorf("get categories for post: %w", err)
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