package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/MosinFAM/graphql-posts/internal/models"

	"github.com/google/uuid"
	"github.com/lib/pq"
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

// GetAllPosts возвращает все посты
func (s *PostgresStorage) GetAllPosts() ([]models.Post, error) {
	log.Println("Fetching all posts from database")
	rows, err := s.DB.Query("SELECT id, title, content, allow_comments FROM posts")
	if err != nil {
		log.Println("Error fetching posts:", err)
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var post models.Post
		if err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.AllowComments); err != nil {
			log.Println("Error scanning post row:", err)
			return nil, err
		}
		posts = append(posts, post)
	}
	log.Println("Successfully fetched posts")
	return posts, nil
}

// GetPostByID возвращает пост по ID
func (s *PostgresStorage) GetPostByID(id string) (*models.Post, error) {
	log.Printf("Fetching post with ID: %s", id)
	var post models.Post
	err := s.DB.QueryRow("SELECT id, title, content, allow_comments FROM posts WHERE id=$1", id).
		Scan(&post.ID, &post.Title, &post.Content, &post.AllowComments)
	if err != nil {
		log.Println("Error fetching post:", err)
		return nil, err
	}
	return &post, nil
}

// AddPost добавляет новый пост в БД
func (s *PostgresStorage) AddPost(title, content string, allowComments bool) (models.Post, error) {
	post := models.Post{
		ID:            uuid.New().String(),
		Title:         title,
		Content:       content,
		AllowComments: allowComments,
	}
	log.Printf("Adding new post: %+v", post)
	_, err := s.DB.Exec("INSERT INTO posts (id, title, content, allow_comments) VALUES ($1, $2, $3, $4)",
		post.ID, post.Title, post.Content, post.AllowComments)
	if err != nil {
		log.Println("DB Insert Error:", err)
		return models.Post{}, err
	}
	return post, nil
}

func (s *PostgresStorage) AddComment(postID string, parentID *string, content string) (*models.Comment, error) {
	log.Printf("Adding comment to post %s", postID)
	var allowComments bool
	err := s.DB.QueryRow("SELECT allow_comments FROM posts WHERE id=$1", postID).Scan(&allowComments)
	if err != nil {
		log.Println("Post not found:", err)
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
		log.Println("Notification error:", err)
		return nil, err
	}

	log.Printf("Comment added: %+v", comment)
	return &comment, nil
}

func (s *PostgresStorage) GetCommentsByPostID(postID string, limit, offset int) ([]*models.Comment, error) {
	log.Printf("Getting comment by post id %s", postID)
	rows, err := s.DB.Query("SELECT id, post_id, parent_id, content, created_at FROM comments WHERE post_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3",
		postID, limit, offset)
	if err != nil {
		log.Println("Post not found")
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
	log.Printf("Subscribing to comments for post %s", postID)
	ch := make(chan *models.Comment)

	// Подключаемся к LISTEN через pq.Listener
	listener := pq.NewListener(s.DataSource, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Println("Postgres Listener error:", err)
		}
	})

	err := listener.Listen("comments_channel")
	if err != nil {
		log.Println("Failed to listen on comments_channel:", err)
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
				if _, err := fmt.Sscanf(notification.Extra, "%s|%s", &notifPostID, &content); err != nil {
					log.Printf("Error parsing notification.Extra: %v\n", err)
				}

				// fmt.Sscanf(notification.Extra, "%s|%s", &notifPostID, &content)

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

	log.Println("Listening for comments on comments_channel")
	return ch, nil
}
