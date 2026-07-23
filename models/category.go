package models

type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type PostCategory struct {
	PostID     int `json:"post_id"`
	CategoryID int `json:"category_id"`
}