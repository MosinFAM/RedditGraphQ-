package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetAllPosts_Empty(t *testing.T) {
	storage := NewMemoryStorage()

	posts, err := storage.GetAllPosts()

	// Assert что сообщения не найдены и возвращается ошибка
	assert.Error(t, err)
	assert.Nil(t, posts)
}

func TestGetAllPosts_ExistingPosts(t *testing.T) {
	storage := NewMemoryStorage()

	_, err := storage.AddPost("Post 1", "Content", true)
	assert.NoError(t, err)

	posts, err := storage.GetAllPosts()

	// Assert что сообщения найдены и нет ошибок
	assert.NoError(t, err)
	assert.Len(t, posts, 1)
	assert.Equal(t, "Post 1", posts[0].Title)
}

func TestGetPostByID_NotFound(t *testing.T) {
	storage := NewMemoryStorage()

	post, err := storage.GetPostByID("nonexistent-id")

	// Assert что сообщение не найдено и возвращается ошибка
	assert.Error(t, err)
	assert.Nil(t, post)
}

func TestGetPostByID_Found(t *testing.T) {
	storage := NewMemoryStorage()

	post, err := storage.AddPost("Post 1", "Content", true)
	assert.NoError(t, err)

	fetchedPost, err := storage.GetPostByID(post.ID)

	// Assert что сообщение найдено и возвращено правильно
	assert.NoError(t, err)
	assert.Equal(t, post.ID, fetchedPost.ID)
	assert.Equal(t, post.Title, fetchedPost.Title)
}

func TestAddPost(t *testing.T) {
	storage := NewMemoryStorage()

	post, err := storage.AddPost("Post 1", "Content", true)

	// Assert no error и пост правильно добавлен
	assert.NoError(t, err)
	assert.NotEmpty(t, post.ID)
	assert.Equal(t, "Post 1", post.Title)
	assert.True(t, post.AllowComments)
}

func TestAddComment_NoPost(t *testing.T) {
	storage := NewMemoryStorage()

	comment, err := storage.AddComment("nonexistent-post-id", nil, "Test comment")

	// Assert an error and nil комментарий
	assert.Error(t, err)
	assert.Nil(t, comment)
}

func TestAddComment_CommentsDisabled(t *testing.T) {
	storage := NewMemoryStorage()

	post, err := storage.AddPost("Post 1", "Content", false)
	assert.NoError(t, err)

	comment, err := storage.AddComment(post.ID, nil, "Test comment")

	// Assert an error and nil комментарий
	assert.Error(t, err)
	assert.Nil(t, comment)
}

func TestAddComment_Success(t *testing.T) {
	storage := NewMemoryStorage()

	post, err := storage.AddPost("Post 1", "Content", true)
	assert.NoError(t, err)

	comment, err := storage.AddComment(post.ID, nil, "Test comment")

	// Assert no error и комментарий добавлен правильно
	assert.NoError(t, err)
	assert.NotEmpty(t, comment.ID)
	assert.Equal(t, "Test comment", comment.Content)
	assert.Equal(t, post.ID, comment.PostID)
}

func TestAddComment_LongContent(t *testing.T) {
	storage := NewMemoryStorage()

	post, err := storage.AddPost("Post 1", "Content", true)
	assert.NoError(t, err)

	longContent := string(make([]byte, 2001)) // Exceeding 2000 chars
	comment, err := storage.AddComment(post.ID, nil, longContent)

	// Assert error, комментарий слишком длинный
	assert.Error(t, err)
	assert.Nil(t, comment)
}

func TestGetCommentsByPostID_NotFound(t *testing.T) {
	storage := NewMemoryStorage()

	comments, err := storage.GetCommentsByPostID("nonexistent-post-id", 10, 0)

	// Assert error, пост не существует
	assert.Error(t, err)
	assert.Nil(t, comments)
}

func TestGetCommentsByPostID_Success(t *testing.T) {
	storage := NewMemoryStorage()

	post, err := storage.AddPost("Post 1", "Content", true)
	assert.NoError(t, err)

	_, err = storage.AddComment(post.ID, nil, "Test comment")
	assert.NoError(t, err)

	comments, err := storage.GetCommentsByPostID(post.ID, 10, 0)

	// Assert no error и комментарий возвращается правильно
	assert.NoError(t, err)
	assert.Len(t, comments, 1)
	assert.Equal(t, "Test comment", comments[0].Content)
}

func TestSubscribeToComments_Success(t *testing.T) {
	storage := NewMemoryStorage()

	post, err := storage.AddPost("Post 1", "Content", true)
	assert.NoError(t, err)

	ch, err := storage.SubscribeToComments(post.ID)
	assert.NoError(t, err)
	assert.NotNil(t, ch)

	_, err = storage.AddComment(post.ID, nil, "Test comment")
	assert.NoError(t, err)

	// Получение комментария с канала
	select {
	case comment := <-ch:
		assert.Equal(t, "Test comment", comment.Content)
	case <-time.After(1 * time.Second):
		assert.Fail(t, "Failed to receive comment")
	}
}
