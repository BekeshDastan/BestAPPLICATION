package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bekesh/social/backend/post/internal/domain"
	"github.com/google/uuid"
)

// ── helpers ────────────────────────────────────────────────────────────────

func clamp(limit, offset int) (int, int) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func normalizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(t, "#")))
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func filterDeleted(posts []*domain.Post) []*domain.Post {
	out := make([]*domain.Post, 0, len(posts))
	for _, p := range posts {
		if !p.IsDeleted() {
			out = append(out, p)
		}
	}
	return out
}

// ── PostUseCase ────────────────────────────────────────────────────────────

type CreatePostInput struct {
	AuthorID  uuid.UUID
	Caption   string
	MediaURLs []string
	Tags      []string
}

type UpdatePostInput struct {
	Caption string
	Tags    []string
}

type PostUseCase struct {
	posts     domain.PostRepository
	cache     domain.PostCache
	publisher domain.EventPublisher
}

func NewPostUseCase(posts domain.PostRepository, cache domain.PostCache, publisher domain.EventPublisher) *PostUseCase {
	return &PostUseCase{posts: posts, cache: cache, publisher: publisher}
}

func (uc *PostUseCase) CreatePost(ctx context.Context, in CreatePostInput) (*domain.Post, error) {
	if len(in.Caption) > domain.MaxCaptionLen {
		return nil, domain.ErrCaptionTooLong
	}
	if len(in.MediaURLs) == 0 {
		return nil, domain.ErrEmptyMedia
	}
	if len(in.MediaURLs) > domain.MaxMediaCount {
		return nil, domain.ErrTooManyMedia
	}
	now := time.Now()
	p := &domain.Post{
		ID:        uuid.New(),
		AuthorID:  in.AuthorID,
		Caption:   strings.TrimSpace(in.Caption),
		MediaURLs: in.MediaURLs,
		Tags:      normalizeTags(in.Tags),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := uc.posts.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("create post: %w", err)
	}
	_ = uc.publisher.Publish(ctx, domain.EventPostCreated, map[string]string{
		"post_id":   p.ID.String(),
		"author_id": p.AuthorID.String(),
	})
	return p, nil
}

func (uc *PostUseCase) GetPost(ctx context.Context, id uuid.UUID) (*domain.Post, error) {
	if cached, err := uc.cache.GetPost(ctx, id); err == nil {
		return cached, nil
	}
	p, err := uc.posts.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.IsDeleted() {
		return nil, domain.ErrPostNotFound
	}
	_ = uc.cache.SetPost(ctx, p, 5*time.Minute)
	return p, nil
}

func (uc *PostUseCase) UpdatePost(ctx context.Context, id, authorID uuid.UUID, in UpdatePostInput) (*domain.Post, error) {
	p, err := uc.posts.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.IsDeleted() {
		return nil, domain.ErrPostNotFound
	}
	if p.AuthorID != authorID {
		return nil, domain.ErrForbidden
	}
	if len(in.Caption) > domain.MaxCaptionLen {
		return nil, domain.ErrCaptionTooLong
	}
	p.Caption = strings.TrimSpace(in.Caption)
	p.Tags = normalizeTags(in.Tags)
	p.UpdatedAt = time.Now()
	if err = uc.posts.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("update post: %w", err)
	}
	_ = uc.cache.InvalidatePost(ctx, id)
	return p, nil
}

func (uc *PostUseCase) DeletePost(ctx context.Context, id, authorID uuid.UUID) error {
	p, err := uc.posts.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if p.IsDeleted() {
		return domain.ErrPostNotFound
	}
	if p.AuthorID != authorID {
		return domain.ErrForbidden
	}
	if err = uc.posts.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("delete post: %w", err)
	}
	_ = uc.cache.InvalidatePost(ctx, id)
	_ = uc.publisher.Publish(ctx, domain.EventPostDeleted, map[string]string{
		"post_id":   id.String(),
		"author_id": authorID.String(),
	})
	return nil
}

func (uc *PostUseCase) ListUserPosts(ctx context.Context, authorID uuid.UUID, limit, offset int) ([]*domain.Post, error) {
	limit, offset = clamp(limit, offset)
	posts, err := uc.posts.ListByAuthor(ctx, authorID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list posts: %w", err)
	}
	return filterDeleted(posts), nil
}

func (uc *PostUseCase) GetFeed(ctx context.Context, followingIDs []uuid.UUID, limit, offset int) ([]*domain.Post, error) {
	if len(followingIDs) == 0 {
		return []*domain.Post{}, nil
	}
	limit, offset = clamp(limit, offset)
	posts, err := uc.posts.ListByAuthors(ctx, followingIDs, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get feed: %w", err)
	}
	return filterDeleted(posts), nil
}

func (uc *PostUseCase) SearchPosts(ctx context.Context, query string, limit, offset int) ([]*domain.Post, error) {
	limit, offset = clamp(limit, offset)
	posts, err := uc.posts.Search(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("search posts: %w", err)
	}
	return filterDeleted(posts), nil
}

// ── LikeUseCase ────────────────────────────────────────────────────────────

type LikeUseCase struct {
	posts     domain.PostRepository
	likes     domain.LikeRepository
	cache     domain.PostCache
	publisher domain.EventPublisher
	tx        domain.Transactor
}

func NewLikeUseCase(
	posts domain.PostRepository,
	likes domain.LikeRepository,
	cache domain.PostCache,
	publisher domain.EventPublisher,
	tx domain.Transactor,
) *LikeUseCase {
	return &LikeUseCase{posts: posts, likes: likes, cache: cache, publisher: publisher, tx: tx}
}

func (uc *LikeUseCase) LikePost(ctx context.Context, postID, userID uuid.UUID) error {
	p, err := uc.posts.GetByID(ctx, postID)
	if err != nil {
		return err
	}
	if p.IsDeleted() {
		return domain.ErrPostNotFound
	}
	already, err := uc.likes.IsLiked(ctx, postID, userID)
	if err != nil {
		return fmt.Errorf("check like: %w", err)
	}
	if already {
		return domain.ErrAlreadyLiked
	}
	if err = uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := uc.likes.Like(ctx, postID, userID); err != nil {
			return fmt.Errorf("like: %w", err)
		}
		return uc.posts.IncrementLikes(ctx, postID)
	}); err != nil {
		return err
	}
	_ = uc.cache.InvalidatePost(ctx, postID)
	_ = uc.publisher.Publish(ctx, domain.EventPostLiked, map[string]string{
		"post_id": postID.String(),
		"user_id": userID.String(),
	})
	return nil
}

func (uc *LikeUseCase) UnlikePost(ctx context.Context, postID, userID uuid.UUID) error {
	p, err := uc.posts.GetByID(ctx, postID)
	if err != nil {
		return err
	}
	if p.IsDeleted() {
		return domain.ErrPostNotFound
	}
	liked, err := uc.likes.IsLiked(ctx, postID, userID)
	if err != nil {
		return fmt.Errorf("check like: %w", err)
	}
	if !liked {
		return domain.ErrNotLiked
	}
	if err = uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := uc.likes.Unlike(ctx, postID, userID); err != nil {
			return fmt.Errorf("unlike: %w", err)
		}
		return uc.posts.DecrementLikes(ctx, postID)
	}); err != nil {
		return err
	}
	_ = uc.cache.InvalidatePost(ctx, postID)
	_ = uc.publisher.Publish(ctx, domain.EventPostUnliked, map[string]string{
		"post_id": postID.String(),
		"user_id": userID.String(),
	})
	return nil
}

func (uc *LikeUseCase) IsLiked(ctx context.Context, postID, userID uuid.UUID) (bool, error) {
	return uc.likes.IsLiked(ctx, postID, userID)
}

func (uc *LikeUseCase) ListLikers(ctx context.Context, postID uuid.UUID, limit, offset int) ([]uuid.UUID, error) {
	limit, offset = clamp(limit, offset)
	return uc.likes.ListLikers(ctx, postID, limit, offset)
}

// ── CommentUseCase ─────────────────────────────────────────────────────────

type CommentUseCase struct {
	posts     domain.PostRepository
	comments  domain.CommentRepository
	publisher domain.EventPublisher
	tx        domain.Transactor
}

func NewCommentUseCase(
	posts domain.PostRepository,
	comments domain.CommentRepository,
	publisher domain.EventPublisher,
	tx domain.Transactor,
) *CommentUseCase {
	return &CommentUseCase{posts: posts, comments: comments, publisher: publisher, tx: tx}
}

func (uc *CommentUseCase) AddComment(ctx context.Context, postID, authorID uuid.UUID, body string) (*domain.Comment, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, domain.ErrCommentEmpty
	}
	if len(body) > domain.MaxCommentLen {
		return nil, domain.ErrCommentTooLong
	}
	p, err := uc.posts.GetByID(ctx, postID)
	if err != nil {
		return nil, err
	}
	if p.IsDeleted() {
		return nil, domain.ErrPostNotFound
	}
	now := time.Now()
	c := &domain.Comment{
		ID:        uuid.New(),
		PostID:    postID,
		AuthorID:  authorID,
		Body:      body,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err = uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := uc.comments.Create(ctx, c); err != nil {
			return fmt.Errorf("create comment: %w", err)
		}
		return uc.posts.IncrementComments(ctx, postID)
	}); err != nil {
		return nil, err
	}
	_ = uc.publisher.Publish(ctx, domain.EventPostCommented, map[string]string{
		"post_id":    postID.String(),
		"comment_id": c.ID.String(),
		"author_id":  authorID.String(),
	})
	return c, nil
}

func (uc *CommentUseCase) DeleteComment(ctx context.Context, commentID, requesterID uuid.UUID) error {
	c, err := uc.comments.GetByID(ctx, commentID)
	if err != nil {
		return err
	}
	if c.IsDeleted() {
		return domain.ErrCommentNotFound
	}
	if c.AuthorID != requesterID {
		return domain.ErrForbidden
	}
	return uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := uc.comments.SoftDelete(ctx, commentID); err != nil {
			return fmt.Errorf("delete comment: %w", err)
		}
		return uc.posts.DecrementComments(ctx, c.PostID)
	})
}

func (uc *CommentUseCase) ListComments(ctx context.Context, postID uuid.UUID, limit, offset int) ([]*domain.Comment, error) {
	limit, offset = clamp(limit, offset)
	comments, err := uc.comments.ListByPost(ctx, postID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	out := make([]*domain.Comment, 0, len(comments))
	for _, c := range comments {
		if !c.IsDeleted() {
			out = append(out, c)
		}
	}
	return out, nil
}
