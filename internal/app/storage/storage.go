package storage

import "graphql-posts/internal/app/models"

// Storage - интерфейс для всех типов хранилищ (in-memory и PostgreSQL)
type Storage interface {
	GetAllPosts() []models.Post
	GetPostByID(id string) *models.Post
	AddPost(title, content string, allowComments bool) models.Post
	AddComment(postID string, parentID *string, content string) (*models.Comment, error)
	GetCommentsByPostID(postID string, limit, offset int) ([]*models.Comment, error)
	SubscribeToComments(postID string) (<-chan *models.Comment, error)
}
