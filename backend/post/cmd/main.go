package main

import (
	"log/slog"
	"os"

	"github.com/bekesh/social/backend/post/internal/config"
	"github.com/bekesh/social/backend/post/internal/infrastructure/messaging"
	pginfra "github.com/bekesh/social/backend/post/internal/infrastructure/postgres"
	redisinfra "github.com/bekesh/social/backend/post/internal/infrastructure/redis"
	grpctransport "github.com/bekesh/social/backend/post/internal/transport/grpc"
	"github.com/bekesh/social/backend/post/internal/usecase"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	cfg, err := config.Load()
	must(err, "load config")

	// ── PostgreSQL ─────────────────────────────────────────────────────────
	db, err := sqlx.Connect("postgres", cfg.Postgres.DSN)
	must(err, "connect postgres")
	defer db.Close()

	if err = goose.Up(db.DB, "migrations"); err != nil {
		slog.Warn("goose migration warning", "err", err)
	}

	txDB := pginfra.NewDB(db)
	postRepo := pginfra.NewPostRepo(db)
	likeRepo := pginfra.NewLikeRepo(db)
	commentRepo := pginfra.NewCommentRepo(db)
	saveRepo := pginfra.NewSaveRepo(db)

	// ── Redis ──────────────────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
	})
	cache := redisinfra.NewPostCache(rdb)

	// ── NATS JetStream ─────────────────────────────────────────────────────
	nc, err := nats.Connect(cfg.NATS.URL)
	must(err, "connect nats")
	defer nc.Drain()

	publisher, err := messaging.NewNATSPublisher(nc, cfg.NATS.Stream)
	must(err, "create nats publisher")

	// ── Use-cases ──────────────────────────────────────────────────────────
	postUC := usecase.NewPostUseCase(postRepo, cache, publisher)
	likeUC := usecase.NewLikeUseCase(postRepo, likeRepo, cache, publisher, txDB)
	commentUC := usecase.NewCommentUseCase(postRepo, commentRepo, publisher, txDB)
	saveUC := usecase.NewSaveUseCase(postRepo, saveRepo)

	// ── gRPC transport ─────────────────────────────────────────────────────
	postH := grpctransport.NewPostHandler(postUC)
	likeH := grpctransport.NewLikeHandler(likeUC)
	commentH := grpctransport.NewCommentHandler(commentUC)
	saveH := grpctransport.NewSaveHandler(saveUC)
	srv := grpctransport.NewPostServer(postH, likeH, commentH, saveH)

	must(grpctransport.Run(cfg.GRPC.Port, srv), "grpc server")
}

func must(err error, msg string) {
	if err != nil {
		slog.Error(msg, "err", err)
		os.Exit(1)
	}
}
