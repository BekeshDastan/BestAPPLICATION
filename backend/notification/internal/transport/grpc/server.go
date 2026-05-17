package grpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bekesh/social/backend/notification/internal/domain"
	"github.com/bekesh/social/backend/notification/internal/usecase"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	notifv1 "github.com/bekesh/social/gen/go/notification/v1"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// ── Error mapping ──────────────────────────────────────────────────────────

func invalidArg(msg string) error { return status.Error(codes.InvalidArgument, msg) }

func domainErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrNotificationNotFound), errors.Is(err, domain.ErrPreferenceNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrUserIDRequired), errors.Is(err, domain.ErrMessageRequired),
		errors.Is(err, domain.ErrInvalidType):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}

// ── Proto converters ───────────────────────────────────────────────────────

func toProtoNotif(n *domain.Notification) *notifv1.NotificationProto {
	return &notifv1.NotificationProto{
		Id:            n.ID.String(),
		UserId:        n.UserID.String(),
		ActorId:       n.ActorID.String(),
		Type:          string(n.Type),
		ReferenceId:   n.ReferenceID.String(),
		ReferenceType: n.ReferenceType,
		Message:       n.Message,
		IsRead:        n.IsRead,
		CreatedAt:     n.CreatedAt.Unix(),
	}
}

func toProtoPref(p *domain.NotificationPreference) *notifv1.PreferenceProto {
	return &notifv1.PreferenceProto{
		UserId:       p.UserID.String(),
		Type:         string(p.Type),
		EmailEnabled: p.EmailEnabled,
		PushEnabled:  p.PushEnabled,
	}
}

// ── NotificationHandler ────────────────────────────────────────────────────

type NotificationHandler struct {
	notifv1.UnimplementedNotificationServiceServer
	notifUC *usecase.NotificationUseCase
	prefUC  *usecase.PreferenceUseCase
	email   domain.EmailSender
}

func NewNotificationHandler(
	notifUC *usecase.NotificationUseCase,
	prefUC *usecase.PreferenceUseCase,
	email domain.EmailSender,
) *NotificationHandler {
	return &NotificationHandler{notifUC: notifUC, prefUC: prefUC, email: email}
}

func (h *NotificationHandler) CreateNotification(ctx context.Context, req *notifv1.CreateNotificationRequest) (*notifv1.CreateNotificationResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	actorID, err := uuid.Parse(req.ActorId)
	if err != nil {
		return nil, invalidArg("invalid actor_id")
	}
	refID, _ := uuid.Parse(req.ReferenceId)

	n := &domain.Notification{
		UserID:        userID,
		ActorID:       actorID,
		Type:          domain.NotificationType(req.Type),
		ReferenceID:   refID,
		ReferenceType: req.ReferenceType,
		Message:       req.Message,
		CreatedAt:     time.Now(),
	}
	created, err := h.notifUC.Create(ctx, n)
	if err != nil {
		return nil, domainErr(err)
	}
	return &notifv1.CreateNotificationResponse{Notification: toProtoNotif(created)}, nil
}

func (h *NotificationHandler) GetNotification(ctx context.Context, req *notifv1.GetNotificationRequest) (*notifv1.GetNotificationResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, invalidArg("invalid id")
	}
	callerID, err := uuid.Parse(req.CallerId)
	if err != nil {
		return nil, invalidArg("invalid caller_id")
	}
	n, err := h.notifUC.GetByID(ctx, id, callerID)
	if err != nil {
		return nil, domainErr(err)
	}
	return &notifv1.GetNotificationResponse{Notification: toProtoNotif(n)}, nil
}

func (h *NotificationHandler) ListNotifications(ctx context.Context, req *notifv1.ListNotificationsRequest) (*notifv1.ListNotificationsResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	list, err := h.notifUC.ListByUser(ctx, userID, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, domainErr(err)
	}
	out := make([]*notifv1.NotificationProto, len(list))
	for i, n := range list {
		out[i] = toProtoNotif(n)
	}
	return &notifv1.ListNotificationsResponse{Notifications: out}, nil
}

func (h *NotificationHandler) ListUnread(ctx context.Context, req *notifv1.ListUnreadRequest) (*notifv1.ListUnreadResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	list, err := h.notifUC.ListUnread(ctx, userID, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, domainErr(err)
	}
	out := make([]*notifv1.NotificationProto, len(list))
	for i, n := range list {
		out[i] = toProtoNotif(n)
	}
	return &notifv1.ListUnreadResponse{Notifications: out}, nil
}

func (h *NotificationHandler) MarkAsRead(ctx context.Context, req *notifv1.MarkAsReadRequest) (*notifv1.MarkAsReadResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, invalidArg("invalid id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	if err := h.notifUC.MarkAsRead(ctx, id, userID); err != nil {
		return nil, domainErr(err)
	}
	return &notifv1.MarkAsReadResponse{}, nil
}

func (h *NotificationHandler) MarkAllAsRead(ctx context.Context, req *notifv1.MarkAllAsReadRequest) (*notifv1.MarkAllAsReadResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	if err := h.notifUC.MarkAllAsRead(ctx, userID); err != nil {
		return nil, domainErr(err)
	}
	return &notifv1.MarkAllAsReadResponse{}, nil
}

func (h *NotificationHandler) DeleteNotification(ctx context.Context, req *notifv1.DeleteNotificationRequest) (*notifv1.DeleteNotificationResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, invalidArg("invalid id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	if err := h.notifUC.Delete(ctx, id, userID); err != nil {
		return nil, domainErr(err)
	}
	return &notifv1.DeleteNotificationResponse{}, nil
}

func (h *NotificationHandler) DeleteAllRead(ctx context.Context, req *notifv1.DeleteAllReadRequest) (*notifv1.DeleteAllReadResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	if err := h.notifUC.DeleteAllRead(ctx, userID); err != nil {
		return nil, domainErr(err)
	}
	return &notifv1.DeleteAllReadResponse{}, nil
}

func (h *NotificationHandler) GetUnreadCount(ctx context.Context, req *notifv1.GetUnreadCountRequest) (*notifv1.GetUnreadCountResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	count, err := h.notifUC.GetUnreadCount(ctx, userID)
	if err != nil {
		return nil, domainErr(err)
	}
	return &notifv1.GetUnreadCountResponse{Count: count}, nil
}

func (h *NotificationHandler) GetPreferences(ctx context.Context, req *notifv1.GetPreferencesRequest) (*notifv1.GetPreferencesResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	prefs, err := h.prefUC.GetPreferences(ctx, userID)
	if err != nil {
		return nil, domainErr(err)
	}
	out := make([]*notifv1.PreferenceProto, len(prefs))
	for i, p := range prefs {
		out[i] = toProtoPref(p)
	}
	return &notifv1.GetPreferencesResponse{Preferences: out}, nil
}

func (h *NotificationHandler) UpdatePreference(ctx context.Context, req *notifv1.UpdatePreferenceRequest) (*notifv1.UpdatePreferenceResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	p := &domain.NotificationPreference{
		UserID:       userID,
		Type:         domain.NotificationType(req.Type),
		EmailEnabled: req.EmailEnabled,
		PushEnabled:  req.PushEnabled,
	}
	if err := h.prefUC.UpdatePreference(ctx, p); err != nil {
		return nil, domainErr(err)
	}
	return &notifv1.UpdatePreferenceResponse{}, nil
}

func (h *NotificationHandler) SendEmail(ctx context.Context, req *notifv1.SendEmailRequest) (*notifv1.SendEmailResponse, error) {
	if req.To == "" {
		return nil, invalidArg("to is required")
	}
	if req.Subject == "" {
		return nil, invalidArg("subject is required")
	}
	if err := h.email.Send(ctx, req.To, req.Subject, req.Body); err != nil {
		return nil, status.Error(codes.Internal, "failed to send email")
	}
	return &notifv1.SendEmailResponse{}, nil
}

// ── Server ─────────────────────────────────────────────────────────────────

func NewServer(h *NotificationHandler) *grpc.Server {
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
		grpc.ChainStreamInterceptor(grpc_prometheus.StreamServerInterceptor),
	)
	notifv1.RegisterNotificationServiceServer(srv, h)
	reflection.Register(srv)
	grpc_prometheus.Register(srv)
	return srv
}

func Run(port string, srv *grpc.Server) error {
	startMetricsServer(port)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(lis) }()

	slog.Info("notification gRPC server listening", "port", port)

	select {
	case <-ctx.Done():
		slog.Info("shutting down notification gRPC server")
		srv.GracefulStop()
		return nil
	case err := <-errCh:
		return err
	}
}

func startMetricsServer(grpcPort string) {
	portNum, _ := strconv.Atoi(grpcPort)
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		addr := fmt.Sprintf(":%d", portNum+1000)
		slog.Info("metrics server started", "addr", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			slog.Error("metrics server failed", "err", err)
		}
	}()
}
