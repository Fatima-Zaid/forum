package database

import (
	"database/sql"
	"errors"
	"fmt"

	"forum/models"
)


func SetPostReaction(db *sql.DB, postID, userID int, reactionType models.ReactionType) error {
	if reactionType != models.Like && reactionType != models.Dislike {
		return fmt.Errorf("invalid reaction type %q", reactionType)
	}

	var existing models.ReactionType
	err := db.QueryRow(
		`SELECT type FROM post_reactions WHERE post_id = ? AND user_id = ?`,
		postID, userID,
	).Scan(&existing)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, err = db.Exec(
			`INSERT INTO post_reactions (post_id, user_id, type) VALUES (?, ?, ?)`,
			postID, userID, reactionType,
		)
		if err != nil {
			return fmt.Errorf("insert post reaction: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("check post reaction: %w", err)
	case existing == reactionType:
		_, err = db.Exec(`DELETE FROM post_reactions WHERE post_id = ? AND user_id = ?`, postID, userID)
		if err != nil {
			return fmt.Errorf("remove post reaction: %w", err)
		}
		return nil
	default:
		_, err = db.Exec(
			`UPDATE post_reactions SET type = ? WHERE post_id = ? AND user_id = ?`,
			reactionType, postID, userID,
		)
		if err != nil {
			return fmt.Errorf("update post reaction: %w", err)
		}
		return nil
	}
}

func SetCommentReaction(db *sql.DB, commentID, userID int, reactionType models.ReactionType) error {
	if reactionType != models.Like && reactionType != models.Dislike {
		return fmt.Errorf("invalid reaction type %q", reactionType)
	}

	var existing models.ReactionType
	err := db.QueryRow(
		`SELECT type FROM comment_reactions WHERE comment_id = ? AND user_id = ?`,
		commentID, userID,
	).Scan(&existing)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, err = db.Exec(
			`INSERT INTO comment_reactions (comment_id, user_id, type) VALUES (?, ?, ?)`,
			commentID, userID, reactionType,
		)
		if err != nil {
			return fmt.Errorf("insert comment reaction: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("check comment reaction: %w", err)
	case existing == reactionType:
		_, err = db.Exec(`DELETE FROM comment_reactions WHERE comment_id = ? AND user_id = ?`, commentID, userID)
		if err != nil {
			return fmt.Errorf("remove comment reaction: %w", err)
		}
		return nil
	default:
		_, err = db.Exec(
			`UPDATE comment_reactions SET type = ? WHERE comment_id = ? AND user_id = ?`,
			reactionType, commentID, userID,
		)
		if err != nil {
			return fmt.Errorf("update comment reaction: %w", err)
		}
		return nil
	}
}