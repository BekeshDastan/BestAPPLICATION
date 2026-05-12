package grpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os/signal"
	"syscall"

	"github.com/bekesh/social/backend/post/internal/domain"
	"github.com/bekesh/social/backend/post/internal/usecase"
	postv1 "github.com/bekesh/social/gen/go/post/v1"
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
	case errors.Is(err, domain.ErrPostNotFound), errors.Is(err, domain.ErrCommentNotFound), errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrAlreadyLiked):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrNotLiked):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrCaptionTooLong), errors.Is(err, domain.ErrEmptyMedia),
		errors.Is(err, domain.ErrTooManyMedia), errors.Is(err, domain.ErrCommentEmpty),
		errors.Is(err, domain.ErrCommentTooLong):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}

// ── Proto converters ───────────────────────────────────────────────────────

func toProtoPost(p *domain.Post) *postv1.PostProto {
	return &postv1.PostProto{
		Id:            p.ID.String(),
		AuthorId:      p.AuthorID.String(),
		Caption:       p.Caption,
		MediaUrls:     p.MediaURLs,
		Tags:          p.Tags,
		LikesCount:    int32(p.LikesCount),
		CommentsCount: int32(p.CommentsCount),
		CreatedAt:     p.CreatedAt.Unix(),
		UpdatedAt:     p.UpdatedAt.Unix(),
	}
}

func toProtoPosts(posts []*domain.Post) []*postv1.PostProto {
	out := make([]*postv1.PostProto, len(posts))
	for i, p := range posts {
		out[i] = toProtoPost(p)
	}
	return out
}

func toProtoComment(c *domain.Comment) *postv1.CommentProto {
	return &postv1.CommentProto{
		Id:        c.ID.String(),
		PostId:    c.PostID.String(),
		AuthorId:  c.AuthorID.String(),
		Body:      c.Body,
		CreatedAt: c.CreatedAt.Unix(),
	}
}

// ── PostHandler ────────────────────────────────────────────────────────────

type PostHandler struct{ uc *usecase.PostUseCase }

func NewPostHandler(uc *usecase.PostUseCase) *PostHandler { return &PostHandler{uc: uc} }

func (h *PostHandler) CreatePost(ctx context.Context, req *postv1.CreatePostRequest) (*postv1.CreatePostResponse, error) {
	authorID, err := uuid.Parse(req.AuthorId)
	if err != nil {
		return nil, invalidArg("invalid author_id")
	}
	p, err := h.uc.CreatePost(ctx, usecase.CreatePostInput{
		AuthorID:  authorID,
		Caption:   req.Caption,
		MediaURLs: req.MediaUrls,
		Tags:      req.Tags,
	})
	if err != nil {
		return nil, domainErr(err)
	}
	return &postv1.CreatePostResponse{Post: toProtoPost(p)}, nil
}

func (h *PostHandler) GetPost(ctx context.Context, req *postv1.GetPostRequest) (*postv1.GetPostResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, invalidArg("invalid id")
	}
	p, err := h.uc.GetPost(ctx, id)
	if err != nil {
		return nil, domainErr(err)
	}
	return &postv1.GetPostResponse{Post: toProtoPost(p)}, nil
}

func (h *PostHandler) UpdatePost(ctx context.Context, req *postv1.UpdatePostRequest) (*postv1.UpdatePostResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, invalidArg("invalid id")
	}
	authorID, err := uuid.Parse(req.AuthorId)
	if err != nil {
		return nil, invalidArg("invalid author_id")
	}
	p, err := h.uc.UpdatePost(ctx, id, authorID, usecase.UpdatePostInput{
		Caption: req.Caption,
		Tags:    req.Tags,
	})
	if err != nil {
		return nil, domainErr(err)
	}
	return &postv1.UpdatePostResponse{Post: toProtoPost(p)}, nil
}

func (h *PostHandler) DeletePost(ctx context.Context, req *postv1.DeletePostRequest) (*postv1.DeletePostResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, invalidArg("invalid id")
	}
	authorID, err := uuid.Parse(req.AuthorId)
	if err != nil {
		return nil, invalidArg("invalid author_id")
	}
	if err = h.uc.DeletePost(ctx, id, authorID); err != nil {
		return nil, domainErr(err)
	}
	return &postv1.DeletePostResponse{}, nil
}

func (h *PostHandler) ListUserPosts(ctx context.Context, req *postv1.ListUserPostsRequest) (*postv1.ListUserPostsResponse, error) {
	authorID, err := uuid.Parse(req.AuthorId)
	if err != nil {
		return nil, invalidArg("invalid author_id")
	}
	posts, err := h.uc.ListUserPosts(ctx, authorID, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, domainErr(err)
	}
	return &postv1.ListUserPostsResponse{Posts: toProtoPosts(posts)}, nil
}

func (h *PostHandler) GetFeed(ctx context.Context, req *postv1.GetFeedRequest) (*postv1.GetFeedResponse, error) {
	ids := make([]uuid.UUID, 0, len(req.FollowingIds))
	for _, s := range req.FollowingIds {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, invalidArg("invalid following_id: " + s)
		}
		ids = append(ids, id)
	}
	posts, err := h.uc.GetFeed(ctx, ids, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, domainErr(err)
	}
	return &postv1.GetFeedResponse{Posts: toProtoPosts(posts)}, nil
}

func (h *PostHandler) SearchPosts(ctx context.Context, req *postv1.SearchPostsRequest) (*postv1.SearchPostsResponse, error) {
	posts, err := h.uc.SearchPosts(ctx, req.Query, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, domainErr(err)
	}
	return &postv1.SearchPostsResponse{Posts: toProtoPosts(posts)}, nil
}

// ── LikeHandler ────────────────────────────────────────────────────────────

type LikeHandler struct{ uc *usecase.LikeUseCase }

func NewLikeHandler(uc *usecase.LikeUseCase) *LikeHandler { return &LikeHandler{uc: uc} }

func (h *LikeHandler) LikePost(ctx context.Context, req *postv1.LikePostRequest) (*postv1.LikePostResponse, error) {
	postID, err := uuid.Parse(req.PostId)
	if err != nil {
		return nil, invalidArg("invalid post_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	if err = h.uc.LikePost(ctx, postID, userID); err != nil {
		return nil, domainErr(err)
	}
	return &postv1.LikePostResponse{}, nil
}

func (h *LikeHandler) UnlikePost(ctx context.Context, req *postv1.UnlikePostRequest) (*postv1.UnlikePostResponse, error) {
	postID, err := uuid.Parse(req.PostId)
	if err != nil {
		return nil, invalidArg("invalid post_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	if err = h.uc.UnlikePost(ctx, postID, userID); err != nil {
		return nil, domainErr(err)
	}
	return &postv1.UnlikePostResponse{}, nil
}

func (h *LikeHandler) IsLiked(ctx context.Context, req *postv1.IsLikedRequest) (*postv1.IsLikedResponse, error) {
	postID, err := uuid.Parse(req.PostId)
	if err != nil {
		return nil, invalidArg("invalid post_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	ok, err := h.uc.IsLiked(ctx, postID, userID)
	if err != nil {
		return nil, domainErr(err)
	}
	return &postv1.IsLikedResponse{IsLiked: ok}, nil
}

func (h *LikeHandler) ListLikers(ctx context.Context, req *postv1.ListLikersRequest) (*postv1.ListLikersResponse, error) {
	postID, err := uuid.Parse(req.PostId)
	if err != nil {
		return nil, invalidArg("invalid post_id")
	}
	uids, err := h.uc.ListLikers(ctx, postID, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, domainErr(err)
	}
	ids := make([]string, len(uids))
	for i, id := range uids {
		ids[i] = id.String()
	}
	return &postv1.ListLikersResponse{UserIds: ids}, nil
}

// ── CommentHandler ─────────────────────────────────────────────────────────

type CommentHandler struct{ uc *usecase.CommentUseCase }

func NewCommentHandler(uc *usecase.CommentUseCase) *CommentHandler { return &CommentHandler{uc: uc} }

func (h *CommentHandler) AddComment(ctx context.Context, req *postv1.AddCommentRequest) (*postv1.AddCommentResponse, error) {
	postID, err := uuid.Parse(req.PostId)
	if err != nil {
		return nil, invalidArg("invalid post_id")
	}
	authorID, err := uuid.Parse(req.AuthorId)
	if err != nil {
		return nil, invalidArg("invalid author_id")
	}
	c, err := h.uc.AddComment(ctx, postID, authorID, req.Body)
	if err != nil {
		return nil, domainErr(err)
	}
	return &postv1.AddCommentResponse{Comment: toProtoComment(c)}, nil
}

func (h *CommentHandler) DeleteComment(ctx context.Context, req *postv1.DeleteCommentRequest) (*postv1.DeleteCommentResponse, error) {
	commentID, err := uuid.Parse(req.CommentId)
	if err != nil {
		return nil, invalidArg("invalid comment_id")
	}
	requesterID, err := uuid.Parse(req.RequesterId)
	if err != nil {
		return nil, invalidArg("invalid requester_id")
	}
	if err = h.uc.DeleteComment(ctx, commentID, requesterID); err != nil {
		return nil, domainErr(err)
	}
	return &postv1.DeleteCommentResponse{}, nil
}

func (h *CommentHandler) ListComments(ctx context.Context, req *postv1.ListCommentsRequest) (*postv1.ListCommentsResponse, error) {
	postID, err := uuid.Parse(req.PostId)
	if err != nil {
		return nil, invalidArg("invalid post_id")
	}
	comments, err := h.uc.ListComments(ctx, postID, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, domainErr(err)
	}
	out := make([]*postv1.CommentProto, len(comments))
	for i, c := range comments {
		out[i] = toProtoComment(c)
	}
	return &postv1.ListCommentsResponse{Comments: out}, nil
}

// ── PostServer ─────────────────────────────────────────────────────────────

type PostServer struct {
	postv1.UnimplementedPostServiceServer
	post    *PostHandler
	like    *LikeHandler
	comment *CommentHandler
}

func NewPostServer(post *PostHandler, like *LikeHandler, comment *CommentHandler) *PostServer {
	return &PostServer{post: post, like: like, comment: comment}
}

// Post delegation
func (s *PostServer) CreatePost(ctx context.Context, r *postv1.CreatePostRequest) (*postv1.CreatePostResponse, error) {
	return s.post.CreatePost(ctx, r)
}
func (s *PostServer) GetPost(ctx context.Context, r *postv1.GetPostRequest) (*postv1.GetPostResponse, error) {
	return s.post.GetPost(ctx, r)
}
func (s *PostServer) UpdatePost(ctx context.Context, r *postv1.UpdatePostRequest) (*postv1.UpdatePostResponse, error) {
	return s.post.UpdatePost(ctx, r)
}
func (s *PostServer) DeletePost(ctx context.Context, r *postv1.DeletePostRequest) (*postv1.DeletePostResponse, error) {
	return s.post.DeletePost(ctx, r)
}
func (s *PostServer) ListUserPosts(ctx context.Context, r *postv1.ListUserPostsRequest) (*postv1.ListUserPostsResponse, error) {
	return s.post.ListUserPosts(ctx, r)
}
func (s *PostServer) GetFeed(ctx context.Context, r *postv1.GetFeedRequest) (*postv1.GetFeedResponse, error) {
	return s.post.GetFeed(ctx, r)
}
func (s *PostServer) SearchPosts(ctx context.Context, r *postv1.SearchPostsRequest) (*postv1.SearchPostsResponse, error) {
	return s.post.SearchPosts(ctx, r)
}

// Like delegation
func (s *PostServer) LikePost(ctx context.Context, r *postv1.LikePostRequest) (*postv1.LikePostResponse, error) {
	return s.like.LikePost(ctx, r)
}
func (s *PostServer) UnlikePost(ctx context.Context, r *postv1.UnlikePostRequest) (*postv1.UnlikePostResponse, error) {
	return s.like.UnlikePost(ctx, r)
}
func (s *PostServer) IsLiked(ctx context.Context, r *postv1.IsLikedRequest) (*postv1.IsLikedResponse, error) {
	return s.like.IsLiked(ctx, r)
}
func (s *PostServer) ListLikers(ctx context.Context, r *postv1.ListLikersRequest) (*postv1.ListLikersResponse, error) {
	return s.like.ListLikers(ctx, r)
}

// Comment delegation
func (s *PostServer) AddComment(ctx context.Context, r *postv1.AddCommentRequest) (*postv1.AddCommentResponse, error) {
	return s.comment.AddComment(ctx, r)
}
func (s *PostServer) DeleteComment(ctx context.Context, r *postv1.DeleteCommentRequest) (*postv1.DeleteCommentResponse, error) {
	return s.comment.DeleteComment(ctx, r)
}
func (s *PostServer) ListComments(ctx context.Context, r *postv1.ListCommentsRequest) (*postv1.ListCommentsResponse, error) {
	return s.comment.ListComments(ctx, r)
}

// ── Run ────────────────────────────────────────────────────────────────────

func Run(port string, srv *PostServer) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("listen :%s: %w", port, err)
	}

	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(recoveryInterceptor),
	)
	postv1.RegisterPostServiceServer(grpcSrv, srv)
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
