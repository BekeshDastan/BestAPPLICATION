package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type PostRepository interface {
	Create(ctx context.Context, p *Post) error
	GetByID(ctx context.Context, id uuid.UUID) (*Post, error)
	Update(ctx context.Context, p *Post) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	ListByAuthor(ctx context.Context, authorID uuid.UUID, limit, offset int) ([]*Post, error)
	ListByAuthors(ctx context.Context, authorIDs []uuid.UUID, limit, offset int) ([]*Post, error)
	Search(ctx context.Context, query string, limit, offset int) ([]*Post, error)
	IncrementLikes(ctx context.Context, postID uuid.UUID) error
	DecrementLikes(ctx context.Context, postID uuid.UUID) error
	IncrementComments(ctx context.Context, postID uuid.UUID) error
	DecrementComments(ctx context.Context, postID uuid.UUID) error
}

type LikeRepository interface {
	Like(ctx context.Context, postID, userID uuid.UUID) error
	Unlike(ctx context.Context, postID, userID uuid.UUID) error
	IsLiked(ctx context.Context, postID, userID uuid.UUID) (bool, error)
	ListLikers(ctx context.Context, postID uuid.UUID, limit, offset int) ([]uuid.UUID, error)
}

type CommentRepository interface {
	Create(ctx context.Context, c *Comment) error
	GetByID(ctx context.Context, id uuid.UUID) (*Comment, error)
	ListByPost(ctx context.Context, postID uuid.UUID, limit, offset int) ([]*Comment, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

type SaveRepository interface {
	Save(ctx context.Context, postID, userID uuid.UUID) error
	Unsave(ctx context.Context, postID, userID uuid.UUID) error
	IsSaved(ctx context.Context, postID, userID uuid.UUID) (bool, error)
	ListSavedPosts(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Post, error)
}

type PostCache interface {
	GetPost(ctx context.Context, id uuid.UUID) (*Post, error)
	SetPost(ctx context.Context, p *Post, ttl time.Duration) error
	InvalidatePost(ctx context.Context, id uuid.UUID) error
}
