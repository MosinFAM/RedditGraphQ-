package storage

import (
	"github.com/MosinFAM/graphql-posts/internal/models"
	"github.com/stretchr/testify/mock"
)

type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) AddPost(title, content string, allowComments bool) (models.Post, error) {
	args := m.Called(title, content, allowComments)
	return args.Get(0).(models.Post), args.Error(1)
}

func (m *MockStorage) GetAllPosts() ([]models.Post, error) {
	args := m.Called()
	return args.Get(0).([]models.Post), args.Error(1)
}

func (m *MockStorage) GetPostByID(id string) (*models.Post, error) {
	args := m.Called(id)
	return args.Get(0).(*models.Post), args.Error(1)
}

func (m *MockStorage) AddComment(postID string, parentID *string, content string) (*models.Comment, error) {
	args := m.Called(postID, parentID, content)
	return args.Get(0).(*models.Comment), args.Error(1)
}

func (m *MockStorage) GetCommentsByPostID(postID string, limit, offset int) ([]*models.Comment, error) {
	args := m.Called(postID, limit, offset)
	return args.Get(0).([]*models.Comment), args.Error(1)
}

func (m *MockStorage) SubscribeToComments(postID string) (<-chan *models.Comment, error) {
	args := m.Called(postID)
	return args.Get(0).(chan *models.Comment), args.Error(1)
}
