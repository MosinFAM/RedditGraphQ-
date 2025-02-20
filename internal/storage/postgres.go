package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"graphql-posts/internal/models"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

// PostgresStorage - хранилище в PostgreSQL
type PostgresStorage struct {
	DB         *sql.DB
	DataSource string
}

// NewPostgresStorage создаёт экземпляр PostgreSQL-хранилища
func NewPostgresStorage(db *sql.DB, dataSource string) *PostgresStorage {
	return &PostgresStorage{DB: db, DataSource: dataSource}
}

// InitDB инициализирует таблицы в БД
func (s *PostgresStorage) InitDB() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS posts (
            id UUID PRIMARY KEY,
            title TEXT NOT NULL,
            content TEXT NOT NULL,
            allow_comments BOOLEAN DEFAULT FALSE
        )`,
		`CREATE TABLE IF NOT EXISTS comments (
            id UUID PRIMARY KEY,
            post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
            parent_id UUID NULL REFERENCES comments(id) ON DELETE CASCADE,
            content TEXT NOT NULL CHECK (LENGTH(content) <= 2000),
            created_at TIMESTAMP DEFAULT NOW()
        )`,
	}

	for _, query := range queries {
		_, err := s.DB.Exec(query)
		if err != nil {
			return err
		}
	}
	log.Println("Database initialized")
	return nil
}

// GetAllPosts возвращает все посты
func (s *PostgresStorage) GetAllPosts() []models.Post {
	rows, err := s.DB.Query("SELECT id, title, content, allow_comments FROM posts")
	if err != nil {
		log.Println(err)
		return nil
	}
	defer rows.Close()
	log.Println("Работаем с бд")
	var posts []models.Post
	for rows.Next() {
		var post models.Post
		if err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.AllowComments); err != nil {
			log.Println(err)
			return nil
		}
		posts = append(posts, post)
	}
	return posts
}

// GetPostByID возвращает пост по ID
func (s *PostgresStorage) GetPostByID(id string) *models.Post {
	var post models.Post
	err := s.DB.QueryRow("SELECT id, title, content, allow_comments FROM posts WHERE id=$1", id).
		Scan(&post.ID, &post.Title, &post.Content, &post.AllowComments)
	if err != nil {
		log.Println(err)
		return nil
	}
	return &post
}

// AddPost добавляет новый пост в БД
func (s *PostgresStorage) AddPost(title, content string, allowComments bool) models.Post {
	post := models.Post{
		ID:            uuid.New().String(),
		Title:         title,
		Content:       content,
		AllowComments: allowComments,
	}
	_, err := s.DB.Exec("INSERT INTO posts (id, title, content, allow_comments) VALUES ($1, $2, $3, $4)",
		post.ID, post.Title, post.Content, post.AllowComments)
	if err != nil {
		log.Println("DB Insert Error:", err)
		return models.Post{}
	}
	return post
}

func (s *PostgresStorage) AddComment(postID string, parentID *string, content string) (*models.Comment, error) {
	var allowComments bool
	err := s.DB.QueryRow("SELECT allow_comments FROM posts WHERE id=$1", postID).Scan(&allowComments)
	if err != nil {
		return nil, errors.New("post not found")
	}
	if !allowComments {
		return nil, errors.New("comments are disabled for this post")
	}
	if len(content) > 2000 {
		return nil, errors.New("comment is too long")
	}

	comment := models.Comment{
		ID:        uuid.New().String(),
		PostID:    postID,
		ParentID:  parentID,
		Content:   content,
		CreatedAt: time.Now(),
	}

	_, ok := s.DB.Exec("INSERT INTO comments (id, post_id, parent_id, content, created_at) VALUES ($1, $2, $3, $4, $5)",
		comment.ID, comment.PostID, comment.ParentID, comment.Content, comment.CreatedAt)
	if ok != nil {
		log.Println("DB Insert Error:", err)
		return nil, err
	}

	// Отправляем уведомление в PostgreSQL NOTIFY
	notifyQuery := fmt.Sprintf("NOTIFY comments_channel, '%s|%s'", comment.PostID, comment.Content)
	_, err = s.DB.Exec(notifyQuery)
	if err != nil {
		return nil, err
	}

	return &comment, nil
}

func (s *PostgresStorage) GetCommentsByPostID(postID string, limit, offset int) ([]*models.Comment, error) {
	rows, err := s.DB.Query("SELECT id, post_id, parent_id, content, created_at FROM comments WHERE post_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3",
		postID, limit, offset)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer rows.Close()

	var comments []*models.Comment
	for rows.Next() {
		var comment models.Comment
		err := rows.Scan(&comment.ID, &comment.PostID, &comment.ParentID, &comment.Content, &comment.CreatedAt)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		comment.PostID = postID
		comments = append(comments, &comment)
	}
	return comments, nil
}

func (s *PostgresStorage) SubscribeToComments(postID string) (<-chan *models.Comment, error) {
	ch := make(chan *models.Comment)

	// Подключаемся к LISTEN через pq.Listener
	listener := pq.NewListener(s.DataSource, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Println("Postgres Listener error:", err)
		}
	})

	err := listener.Listen("comments_channel")
	if err != nil {
		return nil, fmt.Errorf("failed to listen on comments_channel: %w", err)
	}

	// Горутина для получения уведомлений
	go func() {
		defer close(ch)
		defer listener.Close()

		for {
			select {
			case <-time.After(90 * time.Second):
				// Проверяем соединение каждые 90 секунд
				err := listener.Ping()
				if err != nil {
					log.Println("Postgres Listener ping error:", err)
					return
				}

			case notification := <-listener.Notify:
				if notification == nil {
					continue
				}

				// Разбираем сообщение "postID|content"
				var notifPostID, content string
				fmt.Sscanf(notification.Extra, "%s|%s", &notifPostID, &content)

				// Если подписка на нужный пост, отправляем в канал
				if notifPostID == postID {
					ch <- &models.Comment{
						PostID:    notifPostID,
						Content:   content,
						CreatedAt: time.Now(),
					}
				}
			}
		}
	}()

	return ch, nil
}
