package models

import "time"

type Post struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Title     string    `json:"title"`
	GameTitle string    `json:"game_title"`
	Content   string    `json:"content"`
	ImageURL  string    `json:"image_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`

	Username     string     `json:"username,omitempty"`
	Categories   []Category `json:"categories,omitempty"`
	LikeCount    int        `json:"like_count"`
	DislikeCount int        `json:"dislike_count"`
	UserReaction string     `json:"user_reaction,omitempty"`
	CommentCount int        `json:"comment_count,omitempty"`
}


func (p *Post) IsEdited() bool {
	return p.UpdatedAt != nil
}


func (p *Post) HasCategory(id int) bool {
	for _, c := range p.Categories {
		if c.ID == id {
			return true
		}
	}
	return false
}