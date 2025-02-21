package graph

import (
	"context"
	"errors"
	"testing"

	"github.com/MosinFAM/graphql-posts/internal/models"
	"github.com/MosinFAM/graphql-posts/internal/storage"

	"github.com/stretchr/testify/assert"
)

func TestAddPost(t *testing.T) {
	mockStorage := new(storage.MockStorage)
	resolver := &mutationResolver{&Resolver{Storage: mockStorage}}

	expectedPost := models.Post{ID: "1", Title: "Test Post", Content: "Test Content", AllowComments: true}
	mockStorage.On("AddPost", "Test Post", "Test Content", true).Return(expectedPost, nil)

	post, err := resolver.AddPost(context.Background(), "Test Post", "Test Content", true)
	assert.NoError(t, err)
	assert.NotNil(t, post)
	assert.Equal(t, "Test Post", post.Title)

	mockStorage.AssertExpectations(t)
}

func TestAddPost_Failure(t *testing.T) {
	mockStorage := new(storage.MockStorage)
	resolver := &mutationResolver{&Resolver{Storage: mockStorage}}

	mockStorage.On("AddPost", "Test Post", "Test Content", true).Return(models.Post{}, errors.New("failed to create post"))

	post, err := resolver.AddPost(context.Background(), "Test Post", "Test Content", true)
	assert.Error(t, err)
	assert.Nil(t, post)

	mockStorage.AssertExpectations(t)
}

func TestPosts(t *testing.T) {
	mockStorage := new(storage.MockStorage)
	resolver := &queryResolver{&Resolver{Storage: mockStorage}}

	expectedPosts := []models.Post{
		{ID: "1", Title: "Test Post 1"},
		{ID: "2", Title: "Test Post 2"},
	}
	mockStorage.On("GetAllPosts").Return(expectedPosts, nil)

	posts, err := resolver.Posts(context.Background())
	assert.NoError(t, err)
	assert.Len(t, posts, 2)
	assert.Equal(t, "Test Post 1", posts[0].Title)
	assert.Equal(t, "Test Post 2", posts[1].Title)

	mockStorage.AssertExpectations(t)
}

func TestPost(t *testing.T) {
	mockStorage := new(storage.MockStorage)
	resolver := &queryResolver{&Resolver{Storage: mockStorage}}

	expectedPost := &models.Post{ID: "1", Title: "Test Post"}
	mockStorage.On("GetPostByID", "1").Return(expectedPost, nil)

	post, err := resolver.Post(context.Background(), "1")
	assert.NoError(t, err)
	assert.NotNil(t, post)
	assert.Equal(t, "Test Post", post.Title)

	mockStorage.AssertExpectations(t)
}

func TestAddComment(t *testing.T) {
	mockStorage := new(storage.MockStorage)
	resolver := &mutationResolver{&Resolver{Storage: mockStorage}}

	expectedComment := &models.Comment{ID: "1", PostID: "1", Content: "Test Comment"}
	mockStorage.On("GetPostByID", "1").Return(&models.Post{ID: "1", AllowComments: true}, nil)
	mockStorage.On("AddComment", "1", (*string)(nil), "Test Comment").Return(expectedComment, nil)

	comment, err := resolver.AddComment(context.Background(), "1", nil, "Test Comment")
	assert.NoError(t, err)
	assert.NotNil(t, comment)
	assert.Equal(t, "Test Comment", comment.Content)

	mockStorage.AssertExpectations(t)
}

func TestComments(t *testing.T) {
	mockStorage := new(storage.MockStorage)
	resolver := &queryResolver{&Resolver{Storage: mockStorage}}

	expectedComments := []*models.Comment{
		{ID: "1", PostID: "1", Content: "Test Comment 1"},
		{ID: "2", PostID: "1", Content: "Test Comment 2"},
	}
	mockStorage.On("GetCommentsByPostID", "1", 10, 0).Return(expectedComments, nil)

	comments, err := resolver.Comments(context.Background(), "1", 10, 0)
	assert.NoError(t, err)
	assert.Len(t, comments, 2)
	assert.Equal(t, "Test Comment 1", comments[0].Content)

	mockStorage.AssertExpectations(t)
}

func TestCommentAdded(t *testing.T) {
	mockStorage := new(storage.MockStorage)
	resolver := &subscriptionResolver{&Resolver{Storage: mockStorage}}

	commentCh := make(chan *models.Comment, 1)
	mockStorage.On("SubscribeToComments", "1").Return(commentCh, nil)

	subCh, err := resolver.CommentAdded(context.Background(), "1")
	assert.NoError(t, err)
	assert.NotNil(t, subCh)

	expectedComment := &models.Comment{ID: "1", PostID: "1", Content: "New Comment"}
	commentCh <- expectedComment

	receivedComment := <-subCh
	assert.Equal(t, "New Comment", receivedComment.Content)

	mockStorage.AssertExpectations(t)
}
