package storage

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/MosinFAM/graphql-posts/internal/models"

	"github.com/google/uuid"
)

// MemoryStorage - хранилище в памяти
type MemoryStorage struct {
	posts         map[string]models.Post
	comments      map[string][]models.Comment
	subscriptions map[string][]chan *models.Comment
	mu            sync.RWMutex
}

// NewMemoryStorage создает новое in-memory хранилище
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		posts:         make(map[string]models.Post),
		comments:      make(map[string][]models.Comment),
		subscriptions: make(map[string][]chan *models.Comment),
	}
}

// GetAllPosts возвращает все посты
func (s *MemoryStorage) GetAllPosts() ([]models.Post, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	log.Println("Fetching all posts from memory")
	if len(s.posts) == 0 {
		log.Println("No posts found")
		return nil, errors.New("no posts found")
	}
	var result []models.Post
	for _, post := range s.posts {
		result = append(result, post)
	}

	log.Println("Successfully fetched posts")
	return result, nil
}

// GetPostByID возвращает пост по ID
func (s *MemoryStorage) GetPostByID(id string) (*models.Post, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	log.Printf("Fetching post with ID: %s", id)
	post, exists := s.posts[id]
	if !exists {
		log.Println("Post not found")
		return nil, errors.New("post not found")
	}
	return &post, nil
}

// AddPost добавляет новый пост
func (s *MemoryStorage) AddPost(title, content string, allowComments bool) (models.Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	post := models.Post{
		ID:            uuid.New().String(),
		Title:         title,
		Content:       content,
		AllowComments: allowComments,
	}
	log.Printf("Adding new post: %+v", post)
	s.posts[post.ID] = post
	return post, nil
}

// AddComment добавляет комментарий в память
func (s *MemoryStorage) AddComment(postID string, parentID *string, content string) (*models.Comment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Adding comment to post %s", postID)
	post, exists := s.posts[postID]
	if !exists {
		log.Println("Post not found")
		return nil, errors.New("post not found")
	}
	if !post.AllowComments {
		return nil, errors.New("comments are disabled for this post")
	}
	if len(content) > 2000 {
		return nil, errors.New("comment is too long")
	}

	comment := models.Comment{
		ID:        uuid.New().String(),
		PostID:    postID,
		ParentID:  nil,
		Content:   content,
		CreatedAt: time.Now(),
	}
	if parentID != nil {
		comment.ParentID = parentID
	}

	s.comments[postID] = append(s.comments[postID], comment)

	log.Println("Notificating...")
	// Уведомляем подписчиков
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if subscribers, ok := s.subscriptions[postID]; ok {
			for i := 0; i < len(subscribers); {
				select {
				case subscribers[i] <- &comment:
					i++
				default: // Если канал закрыт или клиент отключился — удаляем подписку
					subscribers = append(subscribers[:i], subscribers[i+1:]...)
				}
			}
			s.subscriptions[postID] = subscribers
		}
	}()

	log.Printf("Comment added: %+v", comment)
	return &comment, nil
}

// GetCommentsByPostID возвращает комментарии к посту
func (s *MemoryStorage) GetCommentsByPostID(postID string, limit, offset int) ([]*models.Comment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	log.Printf("Getting comment by post id %s", postID)
	// Проверка на существование поста
	if _, exists := s.posts[postID]; !exists {
		log.Println("Post not found")
		return nil, errors.New("post not found")
	}

	comments, exists := s.comments[postID]
	if !exists {
		log.Println("No comments found for this post")
		return nil, errors.New("no comments found for this post")
	}

	// Пагинация
	start := offset
	end := offset + limit
	if start > len(comments) {
		return []*models.Comment{}, nil
	}
	if end > len(comments) {
		end = len(comments)
	}

	var result []*models.Comment
	for i := start; i < end; i++ {
		result = append(result, &comments[i])
	}
	return result, nil
}

// SubscribeToComments подписка на комментарии для поста
func (s *MemoryStorage) SubscribeToComments(postID string) (<-chan *models.Comment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Subscribing to comments for post %s", postID)
	ch := make(chan *models.Comment, 1)

	// Добавляем подписчика в список
	s.subscriptions[postID] = append(s.subscriptions[postID], ch)

	log.Println("Listening for comments on comments_channel")
	return ch, nil
}
