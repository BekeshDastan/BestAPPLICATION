package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/bekesh/social/gateway/internal/client"
	"github.com/bekesh/social/gateway/internal/config"
	"github.com/bekesh/social/gateway/internal/handler"
	"github.com/bekesh/social/gateway/internal/middleware"
	"github.com/gin-gonic/gin"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	cfg := config.Load()

	clients, cleanup, err := client.New(
		cfg.GRPC.User,
		cfg.GRPC.Post,
		cfg.GRPC.Chat,
		cfg.GRPC.Story,
		cfg.GRPC.Notification,
	)
	if err != nil {
		slog.Error("connect grpc clients", "err", err)
		os.Exit(1)
	}
	defer cleanup()

	hub     := handler.NewHub()
	authH   := handler.NewAuthHandler(clients.User)
	userH   := handler.NewUserHandler(clients.User)
	postH   := handler.NewPostHandler(clients.Post, clients.User)
	chatH   := handler.NewChatHandler(clients.Chat, clients.User, hub)
	storyH  := handler.NewStoryHandler(clients.Story, clients.User)
	notifH  := handler.NewNotificationHandler(clients.Notification)
	adminH  := handler.NewAdminHandler(clients.User, clients.Post, clients.Chat, clients.Story, clients.Notification)
	mediaH  := handler.NewMediaHandler(
		cfg.MinIO.Endpoint,
		cfg.MinIO.AccessKey,
		cfg.MinIO.SecretKey,
		cfg.MinIO.Bucket,
		cfg.MinIO.PublicHost,
		cfg.MinIO.UseSSL,
	)
	wsH := handler.NewWsHandler(clients.User, hub)

	r := gin.New()
	r.Use(gin.Recovery())

	// CORS — concrete origin from config (supports credentials).
	allowedOrigin := cfg.HTTP.AllowedOrigin
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", allowedOrigin)
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
		c.Header("Vary", "Origin")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	v1 := r.Group("/api/v1")

	// ── WebSocket (public, auth via query param) ───────────────────────────
	v1.GET("/ws", wsH.Handle)

	// ── Auth (public) ──────────────────────────────────────────────────────
	auth := v1.Group("/auth")
	{
		auth.POST("/register",     authH.Register)
		auth.POST("/login",        authH.Login)
		auth.POST("/logout",       authH.Logout)
		auth.POST("/refresh",      authH.Refresh)
		auth.POST("/forgot-password",       authH.ForgotPassword)
		auth.POST("/reset-password",        authH.ResetPassword)
		auth.POST("/verify-email",          authH.VerifyEmail)
		auth.POST("/resend-verification",   authH.ResendVerification)
	}

	// ── Protected ──────────────────────────────────────────────────────────
	protected := v1.Group("")
	protected.Use(middleware.Auth(clients.User))

	// Users
	users := protected.Group("/users")
	{
		users.GET("/me",                  userH.GetMe)
		users.PUT("/me",                  userH.UpdateProfile)
		users.DELETE("/me",               userH.DeleteAccount)
		users.PUT("/avatar",              userH.UpdateAvatar)
		users.GET("/search",              userH.SearchUsers)
		users.GET("/suggestions",         userH.SuggestUsers)
		users.GET("/:id",                 userH.GetProfile)
		users.GET("/:id/is-following",    userH.IsFollowing)
		users.POST("/:id/follow",         userH.Follow)
		users.DELETE("/:id/follow",       userH.Unfollow)
		users.GET("/:id/followers",       userH.ListFollowers)
		users.GET("/:id/following",       userH.ListFollowing)
		users.GET("/:id/posts",           postH.ListUserPosts)
		users.POST("/:id/block",          userH.Block)
		users.DELETE("/:id/block",        userH.Unblock)
	}

	// Posts
	posts := protected.Group("/posts")
	{
		posts.POST("",                    postH.Create)
		posts.GET("/feed",                postH.Feed)
		posts.GET("/search",              postH.Search)
		posts.GET("/saved",               postH.GetSavedPosts)
		posts.GET("/:id",                 postH.Get)
		posts.PUT("/:id",                 postH.Update)
		posts.DELETE("/:id",              postH.Delete)
		posts.POST("/:id/like",           postH.Like)
		posts.DELETE("/:id/like",         postH.Unlike)
		posts.POST("/:id/save",           postH.SavePost)
		posts.DELETE("/:id/save",         postH.UnsavePost)
		posts.POST("/:id/report",         postH.ReportPost)
		posts.GET("/:id/comments",        postH.ListComments)
		posts.POST("/:id/comments",       postH.AddComment)
		posts.DELETE("/:id/comments/:comment_id", postH.DeleteComment)
	}

	// Chats / Conversations
	chats := protected.Group("/chats")
	{
		chats.POST("",                          chatH.CreateConversation)
		chats.GET("",                           chatH.ListConversations)
		chats.GET("/:id",                       chatH.GetConversation)
		chats.DELETE("/:id",                    chatH.DeleteConversation)
		chats.POST("/:id/read",                 chatH.MarkConversationRead)
		chats.PUT("/:id",                       chatH.UpdateGroup)
		chats.POST("/:id/participants",         chatH.AddParticipant)
		chats.DELETE("/:id/participants/:uid",  chatH.RemoveParticipant)
		chats.DELETE("/:id/participants/me",    chatH.LeaveGroup)
		chats.POST("/:id/messages",             chatH.SendMessage)
		chats.GET("/:id/messages",              chatH.ListMessages)
		chats.DELETE("/:id/messages/:msg_id",   chatH.DeleteMessageNested)
	}

	// Flat message routes (used by ConversationView)
	messages := protected.Group("/messages")
	{
		messages.POST("",          chatH.FlatSendMessage)
		messages.PUT("/:id",       chatH.EditMessage)
		messages.DELETE("/:id",    chatH.DeleteMessage)
		messages.POST("/:id/pin",  chatH.PinMessage)
	}

	// Stories
	stories := protected.Group("/stories")
	{
		stories.POST("",                storyH.Create)
		stories.GET("/following",       storyH.ListFollowing)
		stories.GET("/user/:user_id",   storyH.ListUser)
		stories.GET("/:id",             storyH.Get)
		stories.DELETE("/:id",          storyH.Delete)
		stories.POST("/:id/view",       storyH.MarkViewed)
		stories.GET("/:id/viewers",     storyH.ListViewers)
		stories.POST("/:id/reply",      storyH.Reply)
		stories.POST("/:id/reaction",   storyH.AddReaction)
		stories.DELETE("/:id/reaction", storyH.RemoveReaction)
		stories.GET("/:id/analytics",   storyH.Analytics)
	}

	// Highlights
	highlights := protected.Group("/highlights")
	{
		highlights.POST("",                         storyH.CreateHighlight)
		highlights.GET("/user/:user_id",            storyH.ListHighlights)
		highlights.DELETE("/:id",                   storyH.DeleteHighlight)
		highlights.POST("/:id/stories",             storyH.AddToHighlight)
		highlights.DELETE("/:id/stories/:story_id", storyH.RemoveFromHighlight)
	}

	// Notifications
	notifs := protected.Group("/notifications")
	{
		notifs.GET("",           notifH.List)
		notifs.GET("/count",     notifH.UnreadCount)
		notifs.PUT("/read-all",  notifH.MarkAllRead)
		notifs.PUT("/:id/read",  notifH.MarkRead)
		notifs.DELETE("/:id",    notifH.Delete)
	}

	// Auth (protected)
	protected.POST("/auth/change-password", authH.ChangePassword)

	// Notification settings
	protected.GET("/notification-settings",  notifH.GetSettings)
	protected.PUT("/notification-settings",  notifH.UpdateSetting)

	// Media upload
	protected.GET("/media/upload-url", mediaH.UploadURL)

	// Devices (stub – no backend implementation)
	protected.GET("/devices", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"devices": []any{}}) })
	protected.DELETE("/devices/:id", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	// Admin
	admin := protected.Group("/admin")
	admin.Use(middleware.Admin(clients.User, cfg.AdminEmails))
	{
		admin.GET("/me",                    adminH.Me)
		admin.GET("/stats",                 adminH.Stats)
		admin.GET("/stats/registrations",   adminH.StatsRegistrations)
		admin.GET("/stats/posts",           adminH.StatsPosts)
		admin.GET("/activity",              adminH.Activity)
		admin.GET("/users/top",             adminH.TopUsers)
		admin.GET("/users",                 adminH.ListUsers)
		admin.GET("/users/:id",             adminH.GetUser)
		admin.DELETE("/users/:id",          adminH.DeleteUser)
		admin.PUT("/users/:id/suspend",     adminH.BanUser)
		admin.GET("/posts",                 adminH.ListPosts)
		admin.DELETE("/posts/:id",          adminH.DeletePost)
		admin.GET("/stories",               adminH.ListStories)
		admin.DELETE("/stories/:id",        adminH.DeleteStory)
		admin.GET("/reports",               adminH.ListReports)
		admin.PUT("/reports/:id/resolve",   adminH.ResolveReport)
		admin.GET("/system/health",         adminH.SystemHealth)
	}

	slog.Info("gateway starting", "port", cfg.HTTP.Port)
	if err := r.Run(":" + cfg.HTTP.Port); err != nil {
		slog.Error("gateway stopped", "err", err)
		os.Exit(1)
	}
}
