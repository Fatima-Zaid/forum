package models

import "time"

type Post struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Title     string    `json:"title"`
	GameTitle string    `json:"game_title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`

	Username     string     `json:"username,omitempty"`
	Categories   []Category `json:"categories,omitempty"`
	LikeCount    int        `json:"like_count"`
	DislikeCount int        `json:"dislike_count"`
	UserReaction string     `json:"user_reaction,omitempty"` 
	CommentCount int        `json:"comment_count,omitempty"`
}