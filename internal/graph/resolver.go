package graph

// THIS CODE WILL BE UPDATED WITH SCHEMA CHANGES. PREVIOUS IMPLEMENTATION FOR SCHEMA CHANGES WILL BE KEPT IN THE COMMENT SECTION. IMPLEMENTATION FOR UNCHANGED SCHEMA WILL BE KEPT.

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/MosinFAM/graphql-posts/internal/storage"
)

type Resolver struct {
	Storage       storage.Storage
	mu            sync.Mutex
	subscriptions map[string][]chan *Comment
}

// AddPost is the resolver for the addPost field.
func (r *mutationResolver) AddPost(ctx context.Context, title string, content string, allowComments bool) (*Post, error) {
	log.Printf("Adding post: title=%s", title)
	modelPost, err := r.Storage.AddPost(title, content, allowComments)
	if err != nil {
		log.Printf("Failed to create post: %v", err)
		return nil, err
	}
	if modelPost.ID == "" {
		log.Println("Post ID is empty, creation failed")
		return nil, errors.New("failed to create post")
	}

	post := &Post{
		ID:            modelPost.ID,
		Title:         modelPost.Title,
		Content:       modelPost.Content,
		AllowComments: modelPost.AllowComments,
	}

	log.Printf("Post created successfully: ID=%s", post.ID)
	return post, nil
}

// Posts is the resolver for the posts field.
func (r *queryResolver) Posts(ctx context.Context) ([]*Post, error) {
	log.Println("Fetching all posts")
	modelPosts, err := r.Storage.GetAllPosts()
	if err != nil {
		log.Printf("Failed to fetch posts: %v", err)
		return nil, err
	}

	var posts []*Post

	for _, modelPost := range modelPosts {
		post := &Post{
			ID:            modelPost.ID,
			Title:         modelPost.Title,
			AllowComments: modelPost.AllowComments,
		}
		posts = append(posts, post)
	}

	return posts, nil
}

// Post is the resolver for the post field.
func (r *queryResolver) Post(ctx context.Context, id string) (*Post, error) {
	log.Printf("Fetching post with ID: %s", id)
	modelPost, err := r.Storage.GetPostByID(id)
	if err != nil {
		log.Printf("Failed to fetch post: %v", err)
		return nil, err
	}

	post := &Post{
		ID:            modelPost.ID,
		Title:         modelPost.Title,
		Content:       modelPost.Content,
		AllowComments: modelPost.AllowComments,
	}
	return post, nil

}

func (r *mutationResolver) AddComment(ctx context.Context, postID string, parentID *string, content string) (*Comment, error) {
	log.Printf("Adding comment to post ID: %s", postID)
	post, err := r.Storage.GetPostByID(postID)
	if err != nil {
		log.Printf("Failed to fetch post: %v", err)
		return nil, err
	}

	if !post.AllowComments {
		log.Println("Comments are disabled for this post")
		return nil, errors.New("comments are disabled for this post")
	}

	modelComment, err := r.Storage.AddComment(postID, parentID, content)
	if err != nil {
		log.Printf("Failed to add comment: %v", err)
		return nil, err
	}

	comment := &Comment{
		ID:        modelComment.ID,
		PostID:    modelComment.PostID,
		ParentID:  modelComment.ParentID,
		Content:   modelComment.Content,
		CreatedAt: modelComment.CreatedAt.String(),
	}

	log.Printf("Comment added successfully: ID=%s", comment.ID)
	// Отправка комментария в подписки
	go func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		if subscribers, ok := r.subscriptions[postID]; ok {
			for _, ch := range subscribers {
				ch <- comment
			}
		}
	}()

	return comment, nil
}

func (r *queryResolver) Comments(ctx context.Context, postID string, limit, offset int) ([]*Comment, error) {
	log.Printf("Fetching comments for post ID: %s", postID)
	modelComments, err := r.Storage.GetCommentsByPostID(postID, limit, offset)
	if err != nil {
		log.Printf("Failed to fetch comments: %v", err)
		return nil, err
	}

	var comments []*Comment

	for _, modelComment := range modelComments {
		comment := &Comment{
			ID:        modelComment.ID,
			PostID:    modelComment.PostID,
			ParentID:  modelComment.ParentID,
			Content:   modelComment.Content,
			CreatedAt: modelComment.CreatedAt.String(),
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

// Subscription resolver: подписка на новые комментарии
func (r *subscriptionResolver) CommentAdded(ctx context.Context, postID string) (<-chan *Comment, error) {
	log.Printf("Subscribing to comments for post ID: %s", postID)
	modelCh, err := r.Storage.SubscribeToComments(postID)
	if err != nil {
		log.Printf("Failed to subscribe: %v", err)
		return nil, err
	}

	ch := make(chan *Comment, 1)

	// Горутина для преобразования значений
	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				log.Println("Subscription cancelled")
				return // Контекст отменён — просто выходим из горутины
			case comment, ok := <-modelCh:
				if !ok {
					log.Println("Subscription channel closed")
					return // Если modelCh закрыт, выходим из горутины
				}
				convertedComment := &Comment{
					ID:        comment.ID,
					PostID:    comment.PostID,
					ParentID:  comment.ParentID,
					Content:   comment.Content,
					CreatedAt: comment.CreatedAt.String(),
				}
				select {
				case ch <- convertedComment:
				case <-ctx.Done():
					return // Контекст отменён, выходим
				}
			}
		}
	}()

	r.mu.Lock()
	if r.subscriptions == nil {
		r.subscriptions = make(map[string][]chan *Comment)
	}
	r.subscriptions[postID] = append(r.subscriptions[postID], ch)
	r.mu.Unlock()

	// Отписка при завершении контекста
	go func() {
		<-ctx.Done()
		r.mu.Lock()
		defer r.mu.Unlock()

		// Удаляем подписчика
		for i, sub := range r.subscriptions[postID] {
			if sub == ch {
				r.subscriptions[postID] = append(r.subscriptions[postID][:i], r.subscriptions[postID][i+1:]...)
				break
			}
		}
	}()

	return ch, nil
}

// Mutation returns MutationResolver implementation.
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

// Query returns SubscriptionResolver implementation.
func (r *Resolver) Subscription() SubscriptionResolver { return &subscriptionResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type subscriptionResolver struct{ *Resolver }

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//  - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//    it when you're done.
//  - You have helper methods in this file. Move them out to keep these resolver files clean.
/*
	type Resolver struct {
	Storage interface {
		GetAllPosts() []models.Post
		GetPostByID(id string) *models.Post
		AddPost(title, content string) models.Post
	}
}
*/
