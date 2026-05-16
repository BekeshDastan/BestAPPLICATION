package main

import (
	"log/slog"
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

	clients, cleanup, err := client.New(cfg.GRPC.User, cfg.GRPC.Post, cfg.GRPC.Chat, cfg.GRPC.Story)
	if err != nil {
		slog.Error("connect grpc clients", "err", err)
		os.Exit(1)
	}
	defer cleanup()

	authH := handler.NewAuthHandler(clients.User)
	userH := handler.NewUserHandler(clients.User)
	postH := handler.NewPostHandler(clients.Post)
	chatH := handler.NewChatHandler(clients.Chat)
	storyH := handler.NewStoryHandler(clients.Story, clients.User)

	r := gin.New()
	r.Use(gin.Recovery())

	// CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	v1 := r.Group("/api/v1")

	// ── Auth (public) ──────────────────────────────────────────────────────
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authH.Register)
		auth.POST("/login", authH.Login)
		auth.POST("/logout", authH.Logout)
		auth.POST("/refresh", authH.Refresh)
		auth.POST("/forgot-password", authH.ForgotPassword)
		auth.POST("/reset-password", authH.ResetPassword)
	}

	// ── Protected routes ───────────────────────────────────────────────────
	protected := v1.Group("")
	protected.Use(middleware.Auth(clients.User))

	// Users
	users := protected.Group("/users")
	{
		users.GET("/me", userH.GetMe)
		users.PUT("/me", userH.UpdateProfile)
		users.GET("/search", userH.SearchUsers)
		users.GET("/:id", userH.GetProfile)
		users.GET("/:id/is-following", userH.IsFollowing)
		users.POST("/:id/follow", userH.Follow)
		users.DELETE("/:id/follow", userH.Unfollow)
		users.GET("/:id/followers", userH.ListFollowers)
		users.GET("/:id/following", userH.ListFollowing)
		users.GET("/:id/posts", postH.ListUserPosts)
	}

	// Posts
	posts := protected.Group("/posts")
	{
		posts.POST("", postH.Create)
		posts.GET("/feed", postH.Feed)
		posts.GET("/search", postH.Search)
		posts.GET("/:id", postH.Get)
		posts.PUT("/:id", postH.Update)
		posts.DELETE("/:id", postH.Delete)
		posts.POST("/:id/like", postH.Like)
		posts.DELETE("/:id/like", postH.Unlike)
		posts.GET("/:id/comments", postH.ListComments)
		posts.POST("/:id/comments", postH.AddComment)
		posts.DELETE("/:id/comments/:comment_id", postH.DeleteComment)
	}

	// Chats
	chats := protected.Group("/chats")
	{
		chats.POST("", chatH.CreateConversation)
		chats.GET("", chatH.ListConversations)
		chats.GET("/:id", chatH.GetConversation)
		chats.DELETE("/:id", chatH.DeleteConversation)
		chats.POST("/:id/messages", chatH.SendMessage)
		chats.GET("/:id/messages", chatH.ListMessages)
		chats.DELETE("/:id/messages/:msg_id", chatH.DeleteMessage)
	}

	// Stories
	stories := protected.Group("/stories")
	{
		stories.POST("", storyH.Create)
		stories.GET("/following", storyH.ListFollowing)
		stories.GET("/:id", storyH.Get)
		stories.DELETE("/:id", storyH.Delete)
		stories.POST("/:id/view", storyH.MarkViewed)
		stories.GET("/:id/viewers", storyH.ListViewers)
		stories.POST("/:id/reply", storyH.Reply)
		stories.POST("/:id/reaction", storyH.AddReaction)
		stories.DELETE("/:id/reaction", storyH.RemoveReaction)
		stories.GET("/:id/analytics", storyH.Analytics)
	}

	stories.GET("/user/:user_id", storyH.ListUser)

	// Highlights
	highlights := protected.Group("/highlights")
	{
		highlights.POST("", storyH.CreateHighlight)
		highlights.GET("/user/:user_id", storyH.ListHighlights)
		highlights.DELETE("/:id", storyH.DeleteHighlight)
		highlights.POST("/:id/stories", storyH.AddToHighlight)
		highlights.DELETE("/:id/stories/:story_id", storyH.RemoveFromHighlight)
	}

	slog.Info("gateway starting", "port", cfg.HTTP.Port)
	if err := r.Run(":" + cfg.HTTP.Port); err != nil {
		slog.Error("gateway stopped", "err", err)
		os.Exit(1)
	}
}
