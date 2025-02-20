package models

import "time"

// Модель комментария к посту
type Comment struct {
	ID        string    `json:"id"`
	PostID    string    `json:"postId"`   // ID поста, к которому прикреплён комментарий
	ParentID  *string   `json:"parentId"` // ID родительского комментария (null, если корневой)
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}
