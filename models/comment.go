package models

import "time"

type Comment struct {
	ID              int       `json:"id"`
	PostID          int       `json:"post_id"`
	UserID          int       `json:"user_id"`
	ParentCommentID *int      `json:"parent_comment_id,omitempty"`
	Content         string    `json:"content"`
	MediaURL        string    `json:"media_url,omitempty"`
	MediaType       string    `json:"media_type,omitempty"`
	CreatedAt       time.Time `json:"created_at"`

	Username     string    `json:"username,omitempty"`
	Replies      []Comment `json:"replies,omitempty"`
	LikeCount    int       `json:"like_count"`
	DislikeCount int       `json:"dislike_count"`
	UserReaction string    `json:"user_reaction,omitempty"`
}