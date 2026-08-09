package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"forum/models"
)

// CreatePost inserts a post and links it to one or more categories in a
// single transaction, so a failure partway through leaves nothing behind.
// imageURL may be "" if the post has no image.
func CreatePost(db *sql.DB, userID int, title, gameTitle, content, imageURL string, categoryIDs []int) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() // no-op if committed

	res, err := tx.Exec(
		`INSERT INTO posts (user_id, title, game_title, content, image_url) VALUES (?, ?, ?, ?, ?)`,
		userID, title, gameTitle, content, nullIfEmpty(imageURL),
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

// UpdatePost overwrites an existing post owned by userID. imageURL is only
// applied if non-empty — pass "" to keep the existing image. Returns
// ErrNotFound if the post doesn't exist or isn't owned by this user.
func UpdatePost(db *sql.DB, postID, userID int, title, gameTitle, content, imageURL string, categoryIDs []int) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var res sql.Result
	if imageURL != "" {
		res, err = tx.Exec(
			`UPDATE posts SET title = ?, game_title = ?, content = ?, image_url = ?, updated_at = CURRENT_TIMESTAMP
			 WHERE id = ? AND user_id = ?`,
			title, gameTitle, content, imageURL, postID, userID,
		)
	} else {
		res, err = tx.Exec(
			`UPDATE posts SET title = ?, game_title = ?, content = ?, updated_at = CURRENT_TIMESTAMP
			 WHERE id = ? AND user_id = ?`,
			title, gameTitle, content, postID, userID,
		)
	}
	if err != nil {
		return fmt.Errorf("update post: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("check update result: %w", err)
	}
	if affected == 0 {
		return ErrNotFound // wrong owner, or post doesn't exist
	}

	if _, err := tx.Exec(`DELETE FROM post_categories WHERE post_id = ?`, postID); err != nil {
		return fmt.Errorf("clear categories: %w", err)
	}
	for _, catID := range categoryIDs {
		if _, err := tx.Exec(
			`INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)`,
			postID, catID,
		); err != nil {
			return fmt.Errorf("link category %d: %w", catID, err)
		}
	}

	return tx.Commit()
}

// basePostSelect is reused by every listing/read query below so the shape
// (and the reaction-count subqueries) stays consistent everywhere.
const basePostSelect = `
	SELECT
		p.id, p.user_id, u.username, p.title, p.game_title, p.content, COALESCE(p.image_url, ''), p.created_at, p.updated_at,
		(SELECT COUNT(*) FROM post_reactions r WHERE r.post_id = p.id AND r.type = 'like') AS like_count,
		(SELECT COUNT(*) FROM post_reactions r WHERE r.post_id = p.id AND r.type = 'dislike') AS dislike_count,
		(SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id) AS comment_count
	FROM posts p
	JOIN users u ON u.id = p.user_id
`

// sqliteTimeLayouts covers the string formats CURRENT_TIMESTAMP and Go's
// time package tend to produce, tried in order.
var sqliteTimeLayouts = []string{
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02T15:04:05.999999999-07:00",
	"2006-01-02 15:04:05",
	time.RFC3339,
	time.RFC3339Nano,
}

// nullableTime scans a nullable datetime column regardless of whether the
// sqlite driver hands it back as a time.Time or a raw string — mattn's
// driver is inconsistent about this for columns added via ALTER TABLE
// (like updated_at), so we don't rely on database/sql's automatic *time.Time
// parsing the way p.CreatedAt does.
type nullableTime struct {
	Time  time.Time
	Valid bool
}

func (t *nullableTime) Scan(value any) error {
	if value == nil {
		t.Valid = false
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		t.Time = v
		t.Valid = true
		return nil
	case string:
		return t.parseString(v)
	case []byte:
		return t.parseString(string(v))
	default:
		return fmt.Errorf("nullableTime: unsupported type %T", value)
	}
}

func (t *nullableTime) parseString(s string) error {
	if s == "" {
		t.Valid = false
		return nil
	}
	for _, layout := range sqliteTimeLayouts {
		if parsed, err := time.Parse(layout, s); err == nil {
			t.Time = parsed
			t.Valid = true
			return nil
		}
	}
	return fmt.Errorf("nullableTime: could not parse %q", s)
}

func scanPost(row interface{ Scan(...any) error }) (models.Post, error) {
	var p models.Post
	var updatedAt nullableTime
	err := row.Scan(
		&p.ID, &p.UserID, &p.Username, &p.Title, &p.GameTitle, &p.Content, &p.ImageURL, &p.CreatedAt, &updatedAt,
		&p.LikeCount, &p.DislikeCount, &p.CommentCount,
	)
	if err != nil {
		return p, err
	}
	if updatedAt.Valid {
		p.UpdatedAt = &updatedAt.Time
	}
	return p, nil
}

// GetPostByID also attaches the post's categories.
func GetPostByID(db *sql.DB, postID int) (*models.Post, error) {
	row := db.QueryRow(basePostSelect+` WHERE p.id = ?`, postID)
	p, err := scanPost(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get post: %w", err)
	}

	// Safe here (unlike collectPosts below): db.QueryRow's Scan already
	// released the connection back to the pool before we get here, so this
	// second query can't deadlock against the 1-connection pool.
	cats, err := getCategoriesForPost(db, postID)
	if err != nil {
		return nil, err
	}
	p.Categories = cats
	return &p, nil
}

// GetAllPosts returns every post, most recent first. Used for the main feed.
func GetAllPosts(db *sql.DB) ([]models.Post, error) {
	rows, err := db.Query(basePostSelect + ` ORDER BY p.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("get all posts: %w", err)
	}
	return collectPosts(db, rows)
}

// GetPostsByCategory implements the "categories" filter (subforum view).
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

// GetPostsByUser implements the "created posts" filter — only meaningful
// for the logged-in user per the project spec.
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

// GetPostsLikedByUser implements the "liked posts" filter.
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

// collectPosts scans rows into a slice, then attaches categories to each
// post in a second pass. IMPORTANT: the category lookups must happen after
// rows.Close(), not while iterating — the connection pool is capped at 1
// (see database.go's SetMaxOpenConns(1)), so issuing a second query while
// these rows are still open would deadlock waiting for a connection that
// never frees up.
func collectPosts(db *sql.DB, rows *sql.Rows) ([]models.Post, error) {
	var posts []models.Post
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan post: %w", err)
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close rows: %w", err)
	}

	for i := range posts {
		cats, err := getCategoriesForPost(db, posts[i].ID)
		if err != nil {
			return nil, err
		}
		posts[i].Categories = cats
	}
	return posts, nil
}

// DeletePost deletes a post only if it belongs to userID (ownership is
// enforced here, not just in the handler, so this function is safe to call
// even if a handler-level check is ever skipped). Returns ErrNotFound if the
// post doesn't exist or isn't owned by this user. Comments, reactions, and
// category links cascade-delete via the schema's ON DELETE CASCADE.
func DeletePost(db *sql.DB, postID, userID int) error {
	res, err := db.Exec(`DELETE FROM posts WHERE id = ? AND user_id = ?`, postID, userID)
	if err != nil {
		return fmt.Errorf("delete post: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("check delete result: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
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