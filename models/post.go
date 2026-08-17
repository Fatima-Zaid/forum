package models

import "time"

type Post struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id"`
	Title     string     `json:"title"`
	GameTitle string     `json:"game_title"`
	Content   string     `json:"content"`
	Images    []string   `json:"images,omitempty"` // ordered, replaces ImageURL
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`

	Username     string     `json:"username,omitempty"`
	Categories   []Category `json:"categories,omitempty"`
	LikeCount    int        `json:"like_count"`
	DislikeCount int        `json:"dislike_count"`
	UserReaction string     `json:"user_reaction,omitempty"`
	CommentCount int        `json:"comment_count,omitempty"`
}

// IsEdited reports whether the post has been edited since creation.
// Templates call this directly: {{if .IsEdited}}.
func (p Post) IsEdited() bool {
	return p.UpdatedAt != nil
}

// HasCategory reports whether the post is tagged with the given category id.
// Used by the edit form to pre-check the right boxes: {{if .HasCategory 3}}.
func (p Post) HasCategory(id int) bool {
	for _, c := range p.Categories {
		if c.ID == id {
			return true
		}
	}
	return false
}