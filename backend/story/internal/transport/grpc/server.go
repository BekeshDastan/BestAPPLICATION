package grpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os/signal"
	"syscall"

	"github.com/bekesh/social/backend/story/internal/domain"
	"github.com/bekesh/social/backend/story/internal/usecase"
	storyv1 "github.com/bekesh/social/gen/go/story/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// ── Error mapping ──────────────────────────────────────────────────────────

func invalidArg(msg string) error { return status.Error(codes.InvalidArgument, msg) }

func domainErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrStoryNotFound), errors.Is(err, domain.ErrHighlightNotFound), errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrAlreadyViewed), errors.Is(err, domain.ErrAlreadyReacted):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrStoryExpired),
		errors.Is(err, domain.ErrMediaURLRequired), errors.Is(err, domain.ErrInvalidMediaType),
		errors.Is(err, domain.ErrInvalidHighlightTitle), errors.Is(err, domain.ErrReplyTextEmpty):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrReactionNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}

// ── Proto converters ───────────────────────────────────────────────────────

func toProtoStory(s *domain.Story) *storyv1.StoryProto {
	return &storyv1.StoryProto{
		Id:         s.ID.String(),
		UserId:     s.UserID.String(),
		MediaUrl:   s.MediaURL,
		MediaType:  string(s.MediaType),
		Caption:    s.Caption,
		ExpiresAt:  s.ExpiresAt.Unix(),
		ViewsCount: int32(s.ViewsCount),
		CreatedAt:  s.CreatedAt.Unix(),
	}
}

func toProtoHighlight(h *domain.Highlight) *storyv1.HighlightProto {
	return &storyv1.HighlightProto{
		Id:        h.ID.String(),
		UserId:    h.UserID.String(),
		Title:     h.Title,
		CoverUrl:  h.CoverURL,
		CreatedAt: h.CreatedAt.Unix(),
	}
}

func toProtoView(v *domain.StoryView) *storyv1.StoryViewProto {
	return &storyv1.StoryViewProto{
		StoryId:  v.StoryID.String(),
		ViewerId: v.ViewerID.String(),
		ViewedAt: v.ViewedAt.Unix(),
	}
}

func toProtoReply(r *domain.StoryReply) *storyv1.StoryReplyProto {
	return &storyv1.StoryReplyProto{
		Id:        r.ID.String(),
		StoryId:   r.StoryID.String(),
		UserId:    r.UserID.String(),
		Text:      r.Text,
		CreatedAt: r.CreatedAt.Unix(),
	}
}

// ── StoryHandler ───────────────────────────────────────────────────────────

type StoryHandler struct {
	uc *usecase.StoryUseCase
}

func NewStoryHandler(uc *usecase.StoryUseCase) *StoryHandler {
	return &StoryHandler{uc: uc}
}

func (h *StoryHandler) CreateStory(ctx context.Context, req *storyv1.CreateStoryRequest) (*storyv1.CreateStoryResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	s, err := h.uc.CreateStory(ctx, userID, req.MediaUrl, req.MediaType, req.Caption)
	if err != nil {
		return nil, domainErr(err)
	}
	return &storyv1.CreateStoryResponse{Story: toProtoStory(s)}, nil
}

func (h *StoryHandler) GetStory(ctx context.Context, req *storyv1.GetStoryRequest) (*storyv1.GetStoryResponse, error) {
	storyID, err := uuid.Parse(req.StoryId)
	if err != nil {
		return nil, invalidArg("invalid story_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	s, err := h.uc.GetStory(ctx, storyID, userID)
	if err != nil {
		return nil, domainErr(err)
	}
	return &storyv1.GetStoryResponse{Story: toProtoStory(s)}, nil
}

func (h *StoryHandler) DeleteStory(ctx context.Context, req *storyv1.DeleteStoryRequest) (*storyv1.DeleteStoryResponse, error) {
	storyID, err := uuid.Parse(req.StoryId)
	if err != nil {
		return nil, invalidArg("invalid story_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	if err = h.uc.DeleteStory(ctx, storyID, userID); err != nil {
		return nil, domainErr(err)
	}
	return &storyv1.DeleteStoryResponse{}, nil
}

func (h *StoryHandler) ListUserStories(ctx context.Context, req *storyv1.ListUserStoriesRequest) (*storyv1.ListUserStoriesResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	stories, err := h.uc.ListUserStories(ctx, userID, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, domainErr(err)
	}
	out := make([]*storyv1.StoryProto, len(stories))
	for i, s := range stories {
		out[i] = toProtoStory(s)
	}
	return &storyv1.ListUserStoriesResponse{Stories: out}, nil
}

func (h *StoryHandler) ListFollowingStories(ctx context.Context, req *storyv1.ListFollowingRequest) (*storyv1.ListFollowingResponse, error) {
	followingIDs := make([]uuid.UUID, 0, len(req.FollowingUserIds))
	for _, s := range req.FollowingUserIds {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, invalidArg("invalid following_user_id: " + s)
		}
		followingIDs = append(followingIDs, id)
	}
	stories, err := h.uc.ListFollowingStories(ctx, followingIDs, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, domainErr(err)
	}
	out := make([]*storyv1.StoryProto, len(stories))
	for i, s := range stories {
		out[i] = toProtoStory(s)
	}
	return &storyv1.ListFollowingResponse{Stories: out}, nil
}

func (h *StoryHandler) MarkStoryViewed(ctx context.Context, req *storyv1.MarkViewedRequest) (*storyv1.MarkViewedResponse, error) {
	storyID, err := uuid.Parse(req.StoryId)
	if err != nil {
		return nil, invalidArg("invalid story_id")
	}
	viewerID, err := uuid.Parse(req.ViewerId)
	if err != nil {
		return nil, invalidArg("invalid viewer_id")
	}
	if err = h.uc.MarkStoryViewed(ctx, storyID, viewerID); err != nil {
		return nil, domainErr(err)
	}
	return &storyv1.MarkViewedResponse{}, nil
}

func (h *StoryHandler) ListStoryViewers(ctx context.Context, req *storyv1.ListViewersRequest) (*storyv1.ListViewersResponse, error) {
	storyID, err := uuid.Parse(req.StoryId)
	if err != nil {
		return nil, invalidArg("invalid story_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	views, err := h.uc.ListStoryViewers(ctx, storyID, userID, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, domainErr(err)
	}
	out := make([]*storyv1.StoryViewProto, len(views))
	for i, v := range views {
		out[i] = toProtoView(v)
	}
	return &storyv1.ListViewersResponse{Viewers: out}, nil
}

func (h *StoryHandler) ReplyToStory(ctx context.Context, req *storyv1.ReplyToStoryRequest) (*storyv1.ReplyToStoryResponse, error) {
	storyID, err := uuid.Parse(req.StoryId)
	if err != nil {
		return nil, invalidArg("invalid story_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	r, err := h.uc.ReplyToStory(ctx, storyID, userID, req.Text)
	if err != nil {
		return nil, domainErr(err)
	}
	return &storyv1.ReplyToStoryResponse{Reply: toProtoReply(r)}, nil
}

func (h *StoryHandler) AddReaction(ctx context.Context, req *storyv1.AddReactionRequest) (*storyv1.AddReactionResponse, error) {
	storyID, err := uuid.Parse(req.StoryId)
	if err != nil {
		return nil, invalidArg("invalid story_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	if err = h.uc.AddReaction(ctx, storyID, userID, req.Emoji); err != nil {
		return nil, domainErr(err)
	}
	return &storyv1.AddReactionResponse{}, nil
}

func (h *StoryHandler) RemoveReaction(ctx context.Context, req *storyv1.RemoveReactionRequest) (*storyv1.RemoveReactionResponse, error) {
	storyID, err := uuid.Parse(req.StoryId)
	if err != nil {
		return nil, invalidArg("invalid story_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	if err = h.uc.RemoveReaction(ctx, storyID, userID); err != nil {
		return nil, domainErr(err)
	}
	return &storyv1.RemoveReactionResponse{}, nil
}

func (h *StoryHandler) GetStoryAnalytics(ctx context.Context, req *storyv1.GetStoryAnalyticsRequest) (*storyv1.GetStoryAnalyticsResponse, error) {
	storyID, err := uuid.Parse(req.StoryId)
	if err != nil {
		return nil, invalidArg("invalid story_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	analytics, err := h.uc.GetStoryAnalytics(ctx, storyID, userID)
	if err != nil {
		return nil, domainErr(err)
	}
	reactionCounts := make([]*storyv1.ReactionCount, 0, len(analytics.Reactions))
	for emoji, count := range analytics.Reactions {
		reactionCounts = append(reactionCounts, &storyv1.ReactionCount{Emoji: emoji, Count: int32(count)})
	}
	return &storyv1.GetStoryAnalyticsResponse{
		StoryId:    analytics.StoryID.String(),
		ViewsCount: int32(analytics.ViewsCount),
		Reactions:  reactionCounts,
	}, nil
}

// ── HighlightHandler ───────────────────────────────────────────────────────

type HighlightHandler struct {
	uc *usecase.HighlightUseCase
}

func NewHighlightHandler(uc *usecase.HighlightUseCase) *HighlightHandler {
	return &HighlightHandler{uc: uc}
}

func (h *HighlightHandler) CreateHighlight(ctx context.Context, req *storyv1.CreateHighlightRequest) (*storyv1.CreateHighlightResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	hl, err := h.uc.CreateHighlight(ctx, userID, req.Title, req.CoverUrl)
	if err != nil {
		return nil, domainErr(err)
	}
	return &storyv1.CreateHighlightResponse{Highlight: toProtoHighlight(hl)}, nil
}

func (h *HighlightHandler) AddToHighlight(ctx context.Context, req *storyv1.AddToHighlightRequest) (*storyv1.AddToHighlightResponse, error) {
	highlightID, err := uuid.Parse(req.HighlightId)
	if err != nil {
		return nil, invalidArg("invalid highlight_id")
	}
	storyID, err := uuid.Parse(req.StoryId)
	if err != nil {
		return nil, invalidArg("invalid story_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	if err = h.uc.AddToHighlight(ctx, highlightID, storyID, userID); err != nil {
		return nil, domainErr(err)
	}
	return &storyv1.AddToHighlightResponse{}, nil
}

func (h *HighlightHandler) RemoveFromHighlight(ctx context.Context, req *storyv1.RemoveFromHighlightRequest) (*storyv1.RemoveFromHighlightResponse, error) {
	highlightID, err := uuid.Parse(req.HighlightId)
	if err != nil {
		return nil, invalidArg("invalid highlight_id")
	}
	storyID, err := uuid.Parse(req.StoryId)
	if err != nil {
		return nil, invalidArg("invalid story_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	if err = h.uc.RemoveFromHighlight(ctx, highlightID, storyID, userID); err != nil {
		return nil, domainErr(err)
	}
	return &storyv1.RemoveFromHighlightResponse{}, nil
}

func (h *HighlightHandler) ListHighlights(ctx context.Context, req *storyv1.ListHighlightsRequest) (*storyv1.ListHighlightsResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	highlights, err := h.uc.ListHighlights(ctx, userID)
	if err != nil {
		return nil, domainErr(err)
	}
	out := make([]*storyv1.HighlightProto, len(highlights))
	for i, hl := range highlights {
		out[i] = toProtoHighlight(hl)
	}
	return &storyv1.ListHighlightsResponse{Highlights: out}, nil
}

func (h *HighlightHandler) DeleteHighlight(ctx context.Context, req *storyv1.DeleteHighlightRequest) (*storyv1.DeleteHighlightResponse, error) {
	highlightID, err := uuid.Parse(req.HighlightId)
	if err != nil {
		return nil, invalidArg("invalid highlight_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	if err = h.uc.DeleteHighlight(ctx, highlightID, userID); err != nil {
		return nil, domainErr(err)
	}
	return &storyv1.DeleteHighlightResponse{}, nil
}

// ── StoryServer ────────────────────────────────────────────────────────────

type StoryServer struct {
	storyv1.UnimplementedStoryServiceServer
	story     *StoryHandler
	highlight *HighlightHandler
}

func NewStoryServer(story *StoryHandler, highlight *HighlightHandler) *StoryServer {
	return &StoryServer{story: story, highlight: highlight}
}

// Story delegation
func (s *StoryServer) CreateStory(ctx context.Context, r *storyv1.CreateStoryRequest) (*storyv1.CreateStoryResponse, error) {
	return s.story.CreateStory(ctx, r)
}
func (s *StoryServer) GetStory(ctx context.Context, r *storyv1.GetStoryRequest) (*storyv1.GetStoryResponse, error) {
	return s.story.GetStory(ctx, r)
}
func (s *StoryServer) DeleteStory(ctx context.Context, r *storyv1.DeleteStoryRequest) (*storyv1.DeleteStoryResponse, error) {
	return s.story.DeleteStory(ctx, r)
}
func (s *StoryServer) ListUserStories(ctx context.Context, r *storyv1.ListUserStoriesRequest) (*storyv1.ListUserStoriesResponse, error) {
	return s.story.ListUserStories(ctx, r)
}
func (s *StoryServer) ListFollowingStories(ctx context.Context, r *storyv1.ListFollowingRequest) (*storyv1.ListFollowingResponse, error) {
	return s.story.ListFollowingStories(ctx, r)
}
func (s *StoryServer) MarkStoryViewed(ctx context.Context, r *storyv1.MarkViewedRequest) (*storyv1.MarkViewedResponse, error) {
	return s.story.MarkStoryViewed(ctx, r)
}
func (s *StoryServer) ListStoryViewers(ctx context.Context, r *storyv1.ListViewersRequest) (*storyv1.ListViewersResponse, error) {
	return s.story.ListStoryViewers(ctx, r)
}
func (s *StoryServer) ReplyToStory(ctx context.Context, r *storyv1.ReplyToStoryRequest) (*storyv1.ReplyToStoryResponse, error) {
	return s.story.ReplyToStory(ctx, r)
}
func (s *StoryServer) AddReaction(ctx context.Context, r *storyv1.AddReactionRequest) (*storyv1.AddReactionResponse, error) {
	return s.story.AddReaction(ctx, r)
}
func (s *StoryServer) RemoveReaction(ctx context.Context, r *storyv1.RemoveReactionRequest) (*storyv1.RemoveReactionResponse, error) {
	return s.story.RemoveReaction(ctx, r)
}
func (s *StoryServer) GetStoryAnalytics(ctx context.Context, r *storyv1.GetStoryAnalyticsRequest) (*storyv1.GetStoryAnalyticsResponse, error) {
	return s.story.GetStoryAnalytics(ctx, r)
}

// Highlight delegation
func (s *StoryServer) CreateHighlight(ctx context.Context, r *storyv1.CreateHighlightRequest) (*storyv1.CreateHighlightResponse, error) {
	return s.highlight.CreateHighlight(ctx, r)
}
func (s *StoryServer) AddToHighlight(ctx context.Context, r *storyv1.AddToHighlightRequest) (*storyv1.AddToHighlightResponse, error) {
	return s.highlight.AddToHighlight(ctx, r)
}
func (s *StoryServer) RemoveFromHighlight(ctx context.Context, r *storyv1.RemoveFromHighlightRequest) (*storyv1.RemoveFromHighlightResponse, error) {
	return s.highlight.RemoveFromHighlight(ctx, r)
}
func (s *StoryServer) ListHighlights(ctx context.Context, r *storyv1.ListHighlightsRequest) (*storyv1.ListHighlightsResponse, error) {
	return s.highlight.ListHighlights(ctx, r)
}
func (s *StoryServer) DeleteHighlight(ctx context.Context, r *storyv1.DeleteHighlightRequest) (*storyv1.DeleteHighlightResponse, error) {
	return s.highlight.DeleteHighlight(ctx, r)
}

// ── Run ────────────────────────────────────────────────────────────────────

func Run(port string, srv *StoryServer) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("listen :%s: %w", port, err)
	}

	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(recoveryInterceptor),
	)
	storyv1.RegisterStoryServiceServer(grpcSrv, srv)
	reflection.Register(grpcSrv)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("gRPC server started", "port", port)
		if err := grpcSrv.Serve(lis); err != nil {
			slog.Error("gRPC serve error", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down gRPC server")
	grpcSrv.GracefulStop()
	return nil
}

func recoveryInterceptor(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	defer func() {
		if p := recover(); p != nil {
			slog.Error("panic in gRPC handler", "panic", p)
			err = fmt.Errorf("internal server error")
		}
	}()
	return handler(ctx, req)
}
