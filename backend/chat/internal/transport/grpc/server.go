package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os/signal"
	"syscall"
	"time"

	"github.com/bekesh/social/backend/chat/internal/domain"
	"github.com/bekesh/social/backend/chat/internal/usecase"
	chatv1 "github.com/bekesh/social/gen/go/chat/v1"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// ── Error mapping ──────────────────────────────────────────────────────────

func invalidArg(msg string) error { return status.Error(codes.InvalidArgument, msg) }

func domainErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrConversationNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrMessageNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrNotParticipant):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrAlreadyParticipant):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrDuplicateReaction):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrMessageEmpty), errors.Is(err, domain.ErrMessageTooLong),
		errors.Is(err, domain.ErrInvalidGroupName), errors.Is(err, domain.ErrGroupRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}

// ── Proto converters ───────────────────────────────────────────────────────

func toProtoConversation(c *domain.Conversation) *chatv1.ConversationProto {
	p := &chatv1.ConversationProto{
		Id:        c.ID.String(),
		Type:      string(c.Type),
		Name:      c.Name,
		AvatarUrl: c.AvatarURL,
		CreatedBy: c.CreatedBy.String(),
		CreatedAt: c.CreatedAt.Unix(),
	}
	if c.LastMessageAt != nil {
		p.LastMessageAt = c.LastMessageAt.Unix()
	}
	return p
}

func toProtoMessage(m *domain.Message) *chatv1.MessageProto {
	p := &chatv1.MessageProto{
		Id:             m.ID.String(),
		ConversationId: m.ConversationID.String(),
		SenderId:       m.SenderID.String(),
		Text:           m.Text,
		MediaUrl:       m.MediaURL,
		IsPinned:       m.IsPinned,
		CreatedAt:      m.CreatedAt.Unix(),
	}
	if m.ReplyToID != nil {
		p.ReplyToId = m.ReplyToID.String()
	}
	if m.EditedAt != nil {
		p.EditedAt = m.EditedAt.Unix()
	}
	return p
}

// ── ConversationHandler ────────────────────────────────────────────────────

type ConversationHandler struct {
	uc *usecase.ConversationUseCase
}

func NewConversationHandler(uc *usecase.ConversationUseCase) *ConversationHandler {
	return &ConversationHandler{uc: uc}
}

func (h *ConversationHandler) CreateConversation(ctx context.Context, req *chatv1.CreateConversationRequest) (*chatv1.CreateConversationResponse, error) {
	creatorID, err := uuid.Parse(req.CreatorId)
	if err != nil {
		return nil, invalidArg("invalid creator_id")
	}
	memberIDs := make([]uuid.UUID, 0, len(req.MemberIds))
	for _, s := range req.MemberIds {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, invalidArg("invalid member_id: " + s)
		}
		memberIDs = append(memberIDs, id)
	}
	conv, err := h.uc.CreateConversation(ctx, creatorID, memberIDs)
	if err != nil {
		return nil, domainErr(err)
	}
	return &chatv1.CreateConversationResponse{Conversation: toProtoConversation(conv)}, nil
}

func (h *ConversationHandler) GetConversation(ctx context.Context, req *chatv1.GetConversationRequest) (*chatv1.GetConversationResponse, error) {
	convID, err := uuid.Parse(req.ConversationId)
	if err != nil {
		return nil, invalidArg("invalid conversation_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	conv, err := h.uc.GetConversation(ctx, convID, userID)
	if err != nil {
		return nil, domainErr(err)
	}
	return &chatv1.GetConversationResponse{Conversation: toProtoConversation(conv)}, nil
}

func (h *ConversationHandler) ListConversations(ctx context.Context, req *chatv1.ListConversationsRequest) (*chatv1.ListConversationsResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	convs, err := h.uc.ListConversations(ctx, userID, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, domainErr(err)
	}
	out := make([]*chatv1.ConversationProto, len(convs))
	for i, c := range convs {
		out[i] = toProtoConversation(c)
	}
	return &chatv1.ListConversationsResponse{Conversations: out}, nil
}

func (h *ConversationHandler) DeleteConversation(ctx context.Context, req *chatv1.DeleteConversationRequest) (*chatv1.DeleteConversationResponse, error) {
	convID, err := uuid.Parse(req.ConversationId)
	if err != nil {
		return nil, invalidArg("invalid conversation_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	if err = h.uc.DeleteConversation(ctx, convID, userID); err != nil {
		return nil, domainErr(err)
	}
	return &chatv1.DeleteConversationResponse{}, nil
}

func (h *ConversationHandler) MarkConversationRead(ctx context.Context, req *chatv1.MarkConvReadRequest) (*chatv1.MarkConvReadResponse, error) {
	convID, err := uuid.Parse(req.ConversationId)
	if err != nil {
		return nil, invalidArg("invalid conversation_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	if err = h.uc.MarkConversationRead(ctx, convID, userID); err != nil {
		return nil, domainErr(err)
	}
	return &chatv1.MarkConvReadResponse{}, nil
}

func (h *ConversationHandler) CreateGroupChat(ctx context.Context, req *chatv1.CreateGroupChatRequest) (*chatv1.CreateGroupChatResponse, error) {
	creatorID, err := uuid.Parse(req.CreatorId)
	if err != nil {
		return nil, invalidArg("invalid creator_id")
	}
	memberIDs := make([]uuid.UUID, 0, len(req.MemberIds))
	for _, s := range req.MemberIds {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, invalidArg("invalid member_id: " + s)
		}
		memberIDs = append(memberIDs, id)
	}
	conv, err := h.uc.CreateGroupChat(ctx, creatorID, req.Name, memberIDs)
	if err != nil {
		return nil, domainErr(err)
	}
	return &chatv1.CreateGroupChatResponse{Conversation: toProtoConversation(conv)}, nil
}

func (h *ConversationHandler) AddParticipant(ctx context.Context, req *chatv1.AddParticipantRequest) (*chatv1.AddParticipantResponse, error) {
	convID, err := uuid.Parse(req.ConversationId)
	if err != nil {
		return nil, invalidArg("invalid conversation_id")
	}
	requesterID, err := uuid.Parse(req.RequesterId)
	if err != nil {
		return nil, invalidArg("invalid requester_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	if err = h.uc.AddParticipant(ctx, convID, requesterID, userID); err != nil {
		return nil, domainErr(err)
	}
	return &chatv1.AddParticipantResponse{}, nil
}

func (h *ConversationHandler) RemoveParticipant(ctx context.Context, req *chatv1.RemoveParticipantRequest) (*chatv1.RemoveParticipantResponse, error) {
	convID, err := uuid.Parse(req.ConversationId)
	if err != nil {
		return nil, invalidArg("invalid conversation_id")
	}
	requesterID, err := uuid.Parse(req.RequesterId)
	if err != nil {
		return nil, invalidArg("invalid requester_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	if err = h.uc.RemoveParticipant(ctx, convID, requesterID, userID); err != nil {
		return nil, domainErr(err)
	}
	return &chatv1.RemoveParticipantResponse{}, nil
}

func (h *ConversationHandler) LeaveGroup(ctx context.Context, req *chatv1.LeaveGroupRequest) (*chatv1.LeaveGroupResponse, error) {
	convID, err := uuid.Parse(req.ConversationId)
	if err != nil {
		return nil, invalidArg("invalid conversation_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	if err = h.uc.LeaveGroup(ctx, convID, userID); err != nil {
		return nil, domainErr(err)
	}
	return &chatv1.LeaveGroupResponse{}, nil
}

func (h *ConversationHandler) UpdateGroupInfo(ctx context.Context, req *chatv1.UpdateGroupInfoRequest) (*chatv1.UpdateGroupInfoResponse, error) {
	convID, err := uuid.Parse(req.ConversationId)
	if err != nil {
		return nil, invalidArg("invalid conversation_id")
	}
	requesterID, err := uuid.Parse(req.RequesterId)
	if err != nil {
		return nil, invalidArg("invalid requester_id")
	}
	if err = h.uc.UpdateGroupInfo(ctx, convID, requesterID, req.Name, req.AvatarUrl); err != nil {
		return nil, domainErr(err)
	}
	return &chatv1.UpdateGroupInfoResponse{}, nil
}

// ── MessageHandler ─────────────────────────────────────────────────────────

type MessageHandler struct {
	uc *usecase.MessageUseCase
}

func NewMessageHandler(uc *usecase.MessageUseCase) *MessageHandler {
	return &MessageHandler{uc: uc}
}

func (h *MessageHandler) SendMessage(ctx context.Context, req *chatv1.SendMessageRequest) (*chatv1.SendMessageResponse, error) {
	convID, err := uuid.Parse(req.ConversationId)
	if err != nil {
		return nil, invalidArg("invalid conversation_id")
	}
	senderID, err := uuid.Parse(req.SenderId)
	if err != nil {
		return nil, invalidArg("invalid sender_id")
	}
	in := usecase.SendMessageInput{
		ConvID:   convID,
		SenderID: senderID,
		Text:     req.Text,
		MediaURL: req.MediaUrl,
	}
	if req.ReplyToId != "" {
		rid, err := uuid.Parse(req.ReplyToId)
		if err != nil {
			return nil, invalidArg("invalid reply_to_id")
		}
		in.ReplyToID = &rid
	}
	m, err := h.uc.SendMessage(ctx, in)
	if err != nil {
		return nil, domainErr(err)
	}
	return &chatv1.SendMessageResponse{Message: toProtoMessage(m)}, nil
}

func (h *MessageHandler) EditMessage(ctx context.Context, req *chatv1.EditMessageRequest) (*chatv1.EditMessageResponse, error) {
	msgID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, invalidArg("invalid message_id")
	}
	senderID, err := uuid.Parse(req.SenderId)
	if err != nil {
		return nil, invalidArg("invalid sender_id")
	}
	m, err := h.uc.EditMessage(ctx, msgID, senderID, req.Text)
	if err != nil {
		return nil, domainErr(err)
	}
	return &chatv1.EditMessageResponse{Message: toProtoMessage(m)}, nil
}

func (h *MessageHandler) DeleteMessage(ctx context.Context, req *chatv1.DeleteMessageRequest) (*chatv1.DeleteMessageResponse, error) {
	msgID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, invalidArg("invalid message_id")
	}
	senderID, err := uuid.Parse(req.SenderId)
	if err != nil {
		return nil, invalidArg("invalid sender_id")
	}
	if err = h.uc.DeleteMessage(ctx, msgID, senderID); err != nil {
		return nil, domainErr(err)
	}
	return &chatv1.DeleteMessageResponse{}, nil
}

func (h *MessageHandler) ListMessages(ctx context.Context, req *chatv1.ListMessagesRequest) (*chatv1.ListMessagesResponse, error) {
	convID, err := uuid.Parse(req.ConversationId)
	if err != nil {
		return nil, invalidArg("invalid conversation_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	msgs, err := h.uc.ListMessages(ctx, convID, userID, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, domainErr(err)
	}
	out := make([]*chatv1.MessageProto, len(msgs))
	for i, m := range msgs {
		out[i] = toProtoMessage(m)
	}
	return &chatv1.ListMessagesResponse{Messages: out}, nil
}

func (h *MessageHandler) SearchMessages(ctx context.Context, req *chatv1.SearchMessagesRequest) (*chatv1.SearchMessagesResponse, error) {
	convID, err := uuid.Parse(req.ConversationId)
	if err != nil {
		return nil, invalidArg("invalid conversation_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	msgs, err := h.uc.SearchMessages(ctx, convID, userID, req.Query, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, domainErr(err)
	}
	out := make([]*chatv1.MessageProto, len(msgs))
	for i, m := range msgs {
		out[i] = toProtoMessage(m)
	}
	return &chatv1.SearchMessagesResponse{Messages: out}, nil
}

func (h *MessageHandler) PinMessage(ctx context.Context, req *chatv1.PinMessageRequest) (*chatv1.PinMessageResponse, error) {
	msgID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, invalidArg("invalid message_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	if err = h.uc.PinMessage(ctx, msgID, userID, req.Pinned); err != nil {
		return nil, domainErr(err)
	}
	return &chatv1.PinMessageResponse{}, nil
}

func (h *MessageHandler) AddReaction(ctx context.Context, req *chatv1.AddReactionRequest) (*chatv1.AddReactionResponse, error) {
	msgID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, invalidArg("invalid message_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	if err = h.uc.AddReaction(ctx, msgID, userID, req.Emoji); err != nil {
		return nil, domainErr(err)
	}
	return &chatv1.AddReactionResponse{}, nil
}

func (h *MessageHandler) RemoveReaction(ctx context.Context, req *chatv1.RemoveReactionRequest) (*chatv1.RemoveReactionResponse, error) {
	msgID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, invalidArg("invalid message_id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	if err = h.uc.RemoveReaction(ctx, msgID, userID, req.Emoji); err != nil {
		return nil, domainErr(err)
	}
	return &chatv1.RemoveReactionResponse{}, nil
}

// ── StreamHandler ──────────────────────────────────────────────────────────

type StreamHandler struct {
	nc    *nats.Conn
	cache domain.ChatCache
}

func NewStreamHandler(nc *nats.Conn, cache domain.ChatCache) *StreamHandler {
	return &StreamHandler{nc: nc, cache: cache}
}

func (h *StreamHandler) SubscribeMessages(req *chatv1.SubscribeRequest, stream chatv1.ChatService_SubscribeMessagesServer) error {
	convID, err := uuid.Parse(req.ConversationId)
	if err != nil {
		return invalidArg("invalid conversation_id")
	}

	subject := domain.EventChatMessageSent + "." + convID.String()
	msgCh := make(chan *nats.Msg, 64)
	sub, err := h.nc.ChanSubscribe(subject, msgCh)
	if err != nil {
		return status.Error(codes.Internal, "subscribe failed")
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case msg, ok := <-msgCh:
			if !ok {
				return nil
			}
			var payload map[string]string
			if err := json.Unmarshal(msg.Data, &payload); err != nil {
				continue
			}
			event := &chatv1.MessageEvent{
				MessageId:      payload["message_id"],
				ConversationId: payload["conversation_id"],
				SenderId:       payload["sender_id"],
				Text:           payload["text"],
				MediaUrl:       payload["media_url"],
				CreatedAt:      time.Now().Unix(),
			}
			if err := stream.Send(event); err != nil {
				return err
			}
		}
	}
}

func (h *StreamHandler) TypingIndicator(stream chatv1.ChatService_TypingIndicatorServer) error {
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		convID, err := uuid.Parse(event.ConversationId)
		if err != nil {
			continue
		}
		userID, err := uuid.Parse(event.UserId)
		if err != nil {
			continue
		}

		if event.IsTyping {
			_ = h.cache.SetTyping(stream.Context(), convID, userID, 5*time.Second)
		}

		if err := stream.Send(event); err != nil {
			return err
		}
	}
}

// ── ChatServer ─────────────────────────────────────────────────────────────

type ChatServer struct {
	chatv1.UnimplementedChatServiceServer
	conv   *ConversationHandler
	msg    *MessageHandler
	stream *StreamHandler
}

func NewChatServer(conv *ConversationHandler, msg *MessageHandler, stream *StreamHandler) *ChatServer {
	return &ChatServer{conv: conv, msg: msg, stream: stream}
}

// Conversation delegation
func (s *ChatServer) CreateConversation(ctx context.Context, r *chatv1.CreateConversationRequest) (*chatv1.CreateConversationResponse, error) {
	return s.conv.CreateConversation(ctx, r)
}
func (s *ChatServer) GetConversation(ctx context.Context, r *chatv1.GetConversationRequest) (*chatv1.GetConversationResponse, error) {
	return s.conv.GetConversation(ctx, r)
}
func (s *ChatServer) ListConversations(ctx context.Context, r *chatv1.ListConversationsRequest) (*chatv1.ListConversationsResponse, error) {
	return s.conv.ListConversations(ctx, r)
}
func (s *ChatServer) DeleteConversation(ctx context.Context, r *chatv1.DeleteConversationRequest) (*chatv1.DeleteConversationResponse, error) {
	return s.conv.DeleteConversation(ctx, r)
}
func (s *ChatServer) MarkConversationRead(ctx context.Context, r *chatv1.MarkConvReadRequest) (*chatv1.MarkConvReadResponse, error) {
	return s.conv.MarkConversationRead(ctx, r)
}
func (s *ChatServer) CreateGroupChat(ctx context.Context, r *chatv1.CreateGroupChatRequest) (*chatv1.CreateGroupChatResponse, error) {
	return s.conv.CreateGroupChat(ctx, r)
}
func (s *ChatServer) AddParticipant(ctx context.Context, r *chatv1.AddParticipantRequest) (*chatv1.AddParticipantResponse, error) {
	return s.conv.AddParticipant(ctx, r)
}
func (s *ChatServer) RemoveParticipant(ctx context.Context, r *chatv1.RemoveParticipantRequest) (*chatv1.RemoveParticipantResponse, error) {
	return s.conv.RemoveParticipant(ctx, r)
}
func (s *ChatServer) LeaveGroup(ctx context.Context, r *chatv1.LeaveGroupRequest) (*chatv1.LeaveGroupResponse, error) {
	return s.conv.LeaveGroup(ctx, r)
}
func (s *ChatServer) UpdateGroupInfo(ctx context.Context, r *chatv1.UpdateGroupInfoRequest) (*chatv1.UpdateGroupInfoResponse, error) {
	return s.conv.UpdateGroupInfo(ctx, r)
}

// Message delegation
func (s *ChatServer) SendMessage(ctx context.Context, r *chatv1.SendMessageRequest) (*chatv1.SendMessageResponse, error) {
	return s.msg.SendMessage(ctx, r)
}
func (s *ChatServer) EditMessage(ctx context.Context, r *chatv1.EditMessageRequest) (*chatv1.EditMessageResponse, error) {
	return s.msg.EditMessage(ctx, r)
}
func (s *ChatServer) DeleteMessage(ctx context.Context, r *chatv1.DeleteMessageRequest) (*chatv1.DeleteMessageResponse, error) {
	return s.msg.DeleteMessage(ctx, r)
}
func (s *ChatServer) ListMessages(ctx context.Context, r *chatv1.ListMessagesRequest) (*chatv1.ListMessagesResponse, error) {
	return s.msg.ListMessages(ctx, r)
}
func (s *ChatServer) SearchMessages(ctx context.Context, r *chatv1.SearchMessagesRequest) (*chatv1.SearchMessagesResponse, error) {
	return s.msg.SearchMessages(ctx, r)
}
func (s *ChatServer) PinMessage(ctx context.Context, r *chatv1.PinMessageRequest) (*chatv1.PinMessageResponse, error) {
	return s.msg.PinMessage(ctx, r)
}
func (s *ChatServer) AddReaction(ctx context.Context, r *chatv1.AddReactionRequest) (*chatv1.AddReactionResponse, error) {
	return s.msg.AddReaction(ctx, r)
}
func (s *ChatServer) RemoveReaction(ctx context.Context, r *chatv1.RemoveReactionRequest) (*chatv1.RemoveReactionResponse, error) {
	return s.msg.RemoveReaction(ctx, r)
}

// Stream delegation
func (s *ChatServer) SubscribeMessages(r *chatv1.SubscribeRequest, stream chatv1.ChatService_SubscribeMessagesServer) error {
	return s.stream.SubscribeMessages(r, stream)
}
func (s *ChatServer) TypingIndicator(stream chatv1.ChatService_TypingIndicatorServer) error {
	return s.stream.TypingIndicator(stream)
}

// ── Run ────────────────────────────────────────────────────────────────────

func Run(port string, srv *ChatServer) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("listen :%s: %w", port, err)
	}

	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(recoveryInterceptor),
	)
	chatv1.RegisterChatServiceServer(grpcSrv, srv)
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
