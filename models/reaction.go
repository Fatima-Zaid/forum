package models

import "time"

type ReactionType string

const (
	Like    ReactionType = "like"
	Dislike ReactionType = "dislike"
)

type PostReaction struct {
	ID        int          `json:"id"`
	PostID    int          `json:"post_id"`
	UserID    int          `json:"user_id"`
	Type      ReactionType `json:"type"`
	CreatedAt time.Time    `json:"created_at"`
}

type CommentReaction struct {
	ID        int          `json:"id"`
	CommentID int          `json:"comment_id"`
	UserID    int          `json:"user_id"`
	Type      ReactionType `json:"type"`
	CreatedAt time.Time    `json:"created_at"`
}