package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/bekesh/social/backend/story/internal/domain"
	"github.com/google/uuid"
)

func clamp(limit, offset int) (int, int) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// ── StoryUseCase ──────────────────────────────────────────────────────────────

type StoryUseCase struct {
	stories   domain.StoryRepository
	views     domain.StoryViewRepository
	replies   domain.StoryReplyRepository
	reactions domain.StoryReactionRepository
	pub       domain.EventPublisher
	cache     domain.StoryCache
	tx        domain.Transactor
}

func NewStoryUseCase(
	stories domain.StoryRepository,
	views domain.StoryViewRepository,
	replies domain.StoryReplyRepository,
	reactions domain.StoryReactionRepository,
	pub domain.EventPublisher,
	cache domain.StoryCache,
	tx domain.Transactor,
) *StoryUseCase {
	return &StoryUseCase{
		stories:   stories,
		views:     views,
		replies:   replies,
		reactions: reactions,
		pub:       pub,
		cache:     cache,
		tx:        tx,
	}
}

func (uc *StoryUseCase) CreateStory(ctx context.Context, userID uuid.UUID, mediaURL, mediaType, caption string) (*domain.Story, error) {
	if strings.TrimSpace(mediaURL) == "" {
		return nil, domain.ErrMediaURLRequired
	}
	mt := domain.MediaType(mediaType)
	if mt != domain.MediaTypeImage && mt != domain.MediaTypeVideo {
		return nil, domain.ErrInvalidMediaType
	}

	now := time.Now()
	s := &domain.Story{
		ID:        uuid.New(),
		UserID:    userID,
		MediaURL:  mediaURL,
		MediaType: mt,
		Caption:   caption,
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	}

	if err := uc.stories.Create(ctx, s); err != nil {
		return nil, err
	}

	_ = uc.pub.Publish(ctx, domain.EventStoryCreated, map[string]string{
		"story_id": s.ID.String(),
		"user_id":  userID.String(),
	})

	return s, nil
}

func (uc *StoryUseCase) GetStory(ctx context.Context, storyID, userID uuid.UUID) (*domain.Story, error) {
	s, err := uc.stories.GetByID(ctx, storyID)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (uc *StoryUseCase) DeleteStory(ctx context.Context, storyID, userID uuid.UUID) error {
	s, err := uc.stories.GetByID(ctx, storyID)
	if err != nil {
		return err
	}
	if s.UserID != userID {
		return domain.ErrForbidden
	}
	return uc.stories.SoftDelete(ctx, storyID)
}

func (uc *StoryUseCase) ListUserStories(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Story, error) {
	limit, offset = clamp(limit, offset)
	return uc.stories.ListByUser(ctx, userID, limit, offset)
}

func (uc *StoryUseCase) ListFollowingStories(ctx context.Context, followingUserIDs []uuid.UUID, limit, offset int) ([]*domain.Story, error) {
	if len(followingUserIDs) == 0 {
		return []*domain.Story{}, nil
	}
	limit, offset = clamp(limit, offset)
	return uc.stories.ListByUserIDs(ctx, followingUserIDs, limit, offset)
}

func (uc *StoryUseCase) MarkStoryViewed(ctx context.Context, storyID, viewerID uuid.UUID) error {
	s, err := uc.stories.GetByID(ctx, storyID)
	if err != nil {
		return err
	}
	if s.IsExpired() || s.IsDeleted() {
		return domain.ErrStoryExpired
	}

	already, err := uc.views.IsViewed(ctx, storyID, viewerID)
	if err != nil {
		return err
	}
	if already {
		return domain.ErrAlreadyViewed
	}

	err = uc.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.views.Add(txCtx, &domain.StoryView{
			StoryID:  storyID,
			ViewerID: viewerID,
			ViewedAt: time.Now(),
		}); err != nil {
			return err
		}
		return uc.stories.IncrViewsCount(txCtx, storyID)
	})
	if err != nil {
		return err
	}

	_ = uc.cache.IncrViews(ctx, storyID)
	_ = uc.pub.Publish(ctx, domain.EventStoryViewed, map[string]string{
		"story_id":  storyID.String(),
		"viewer_id": viewerID.String(),
		"owner_id":  s.UserID.String(),
	})

	return nil
}

func (uc *StoryUseCase) ListStoryViewers(ctx context.Context, storyID, userID uuid.UUID, limit, offset int) ([]*domain.StoryView, error) {
	s, err := uc.stories.GetByID(ctx, storyID)
	if err != nil {
		return nil, err
	}
	if s.UserID != userID {
		return nil, domain.ErrForbidden
	}
	limit, offset = clamp(limit, offset)
	return uc.views.ListViewers(ctx, storyID, limit, offset)
}

func (uc *StoryUseCase) ReplyToStory(ctx context.Context, storyID, userID uuid.UUID, text string) (*domain.StoryReply, error) {
	if strings.TrimSpace(text) == "" {
		return nil, domain.ErrReplyTextEmpty
	}
	s, err := uc.stories.GetByID(ctx, storyID)
	if err != nil {
		return nil, err
	}
	if s.IsExpired() || s.IsDeleted() {
		return nil, domain.ErrStoryExpired
	}

	r := &domain.StoryReply{
		ID:        uuid.New(),
		StoryID:   storyID,
		UserID:    userID,
		Text:      text,
		CreatedAt: time.Now(),
	}
	if err := uc.replies.Create(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

func (uc *StoryUseCase) AddReaction(ctx context.Context, storyID, userID uuid.UUID, emoji string) error {
	if strings.TrimSpace(emoji) == "" {
		return domain.ErrInvalidMediaType
	}
	s, err := uc.stories.GetByID(ctx, storyID)
	if err != nil {
		return err
	}
	if s.IsExpired() || s.IsDeleted() {
		return domain.ErrStoryExpired
	}
	return uc.reactions.Add(ctx, &domain.StoryReaction{
		StoryID: storyID,
		UserID:  userID,
		Emoji:   emoji,
	})
}

func (uc *StoryUseCase) RemoveReaction(ctx context.Context, storyID, userID uuid.UUID) error {
	return uc.reactions.Remove(ctx, storyID, userID)
}

func (uc *StoryUseCase) GetStoryAnalytics(ctx context.Context, storyID, userID uuid.UUID) (*domain.StoryAnalytics, error) {
	s, err := uc.stories.GetByID(ctx, storyID)
	if err != nil {
		return nil, err
	}
	if s.UserID != userID {
		return nil, domain.ErrForbidden
	}

	counts, err := uc.reactions.GetReactionCounts(ctx, storyID)
	if err != nil {
		return nil, err
	}

	return &domain.StoryAnalytics{
		StoryID:    storyID,
		ViewsCount: s.ViewsCount,
		Reactions:  counts,
	}, nil
}

// ── HighlightUseCase ──────────────────────────────────────────────────────────

type HighlightUseCase struct {
	highlights domain.HighlightRepository
	stories    domain.StoryRepository
}

func NewHighlightUseCase(
	highlights domain.HighlightRepository,
	stories domain.StoryRepository,
) *HighlightUseCase {
	return &HighlightUseCase{highlights: highlights, stories: stories}
}

func (uc *HighlightUseCase) CreateHighlight(ctx context.Context, userID uuid.UUID, title, coverURL string) (*domain.Highlight, error) {
	if strings.TrimSpace(title) == "" {
		return nil, domain.ErrInvalidHighlightTitle
	}
	h := &domain.Highlight{
		ID:        uuid.New(),
		UserID:    userID,
		Title:     title,
		CoverURL:  coverURL,
		CreatedAt: time.Now(),
	}
	if err := uc.highlights.Create(ctx, h); err != nil {
		return nil, err
	}
	return h, nil
}

func (uc *HighlightUseCase) AddToHighlight(ctx context.Context, highlightID, storyID, userID uuid.UUID) error {
	h, err := uc.highlights.GetByID(ctx, highlightID)
	if err != nil {
		return err
	}
	if h.UserID != userID {
		return domain.ErrForbidden
	}
	s, err := uc.stories.GetByID(ctx, storyID)
	if err != nil {
		return err
	}
	if s.UserID != userID {
		return domain.ErrForbidden
	}
	return uc.highlights.AddStory(ctx, &domain.HighlightStory{
		HighlightID: highlightID,
		StoryID:     storyID,
	})
}

func (uc *HighlightUseCase) RemoveFromHighlight(ctx context.Context, highlightID, storyID, userID uuid.UUID) error {
	h, err := uc.highlights.GetByID(ctx, highlightID)
	if err != nil {
		return err
	}
	if h.UserID != userID {
		return domain.ErrForbidden
	}
	return uc.highlights.RemoveStory(ctx, highlightID, storyID)
}

func (uc *HighlightUseCase) ListHighlights(ctx context.Context, userID uuid.UUID) ([]*domain.Highlight, error) {
	return uc.highlights.ListByUser(ctx, userID)
}

func (uc *HighlightUseCase) DeleteHighlight(ctx context.Context, highlightID, userID uuid.UUID) error {
	h, err := uc.highlights.GetByID(ctx, highlightID)
	if err != nil {
		return err
	}
	if h.UserID != userID {
		return domain.ErrForbidden
	}
	return uc.highlights.Delete(ctx, highlightID)
}
