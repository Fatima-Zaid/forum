package database

import (
	"database/sql"
	"fmt"

	"forum/models"
)


func CreateComment(db *sql.DB, postID, userID int, parentCommentID *int, content, mediaURL, mediaType string) (int, error) {
	res, err := db.Exec(
		`INSERT INTO comments (post_id, user_id, parent_comment_id, content, media_url, media_type)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		postID, userID, parentCommentID, content, nullIfEmpty(mediaURL), nullIfEmpty(mediaType),
	)
	if err != nil {
		return 0, fmt.Errorf("create comment: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}


func GetCommentsByPost(db *sql.DB, postID int) ([]models.Comment, error) {
	rows, err := db.Query(`
		SELECT
			c.id, c.post_id, c.user_id, u.username, c.parent_comment_id,
			c.content, COALESCE(c.media_url, ''), COALESCE(c.media_type, ''), c.created_at,
			(SELECT COUNT(*) FROM comment_reactions r WHERE r.comment_id = c.id AND r.type = 'like') AS like_count,
			(SELECT COUNT(*) FROM comment_reactions r WHERE r.comment_id = c.id AND r.type = 'dislike') AS dislike_count
		FROM comments c
		JOIN users u ON u.id = c.user_id
		WHERE c.post_id = ?
		ORDER BY c.created_at ASC`,
		postID,
	)
	if err != nil {
		return nil, fmt.Errorf("get comments: %w", err)
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var c models.Comment
		if err := rows.Scan(
			&c.ID, &c.PostID, &c.UserID, &c.Username, &c.ParentCommentID,
			&c.Content, &c.MediaURL, &c.MediaType, &c.CreatedAt,
			&c.LikeCount, &c.DislikeCount,
		); err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}