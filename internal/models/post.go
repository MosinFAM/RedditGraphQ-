package models

// Модель поста
type Post struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Content       string `json:"content"`
	AllowComments bool   `json:"allowComments"`
}
