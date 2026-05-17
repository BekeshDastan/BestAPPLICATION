package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/bekesh/social/backend/story/internal/config"
	"github.com/bekesh/social/backend/story/internal/infrastructure/jobs"
	"github.com/bekesh/social/backend/story/internal/infrastructure/messaging"
	pginfra "github.com/bekesh/social/backend/story/internal/infrastructure/postgres"
	redisinfra "github.com/bekesh/social/backend/story/internal/infrastructure/redis"
	grpctransport "github.com/bekesh/social/backend/story/internal/transport/grpc"
	"github.com/bekesh/social/backend/story/internal/usecase"
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
	storyRepo := pginfra.NewStoryRepo(db)
	viewRepo := pginfra.NewStoryViewRepo(db)
	replyRepo := pginfra.NewStoryReplyRepo(db)
	reactionRepo := pginfra.NewStoryReactionRepo(db)
	highlightRepo := pginfra.NewHighlightRepo(db)

	// ── Redis ──────────────────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
	})
	cache := redisinfra.NewStoryCache(rdb)

	// ── NATS ───────────────────────────────────────────────────────────────
	nc, err := nats.Connect(cfg.NATS.URL)
	must(err, "connect nats")
	defer nc.Drain()

	publisher, err := messaging.NewNATSPublisher(nc, cfg.NATS.Stream)
	must(err, "create nats publisher")

	// ── Use-cases ──────────────────────────────────────────────────────────
	storyUC := usecase.NewStoryUseCase(storyRepo, viewRepo, replyRepo, reactionRepo, publisher, cache, txDB)
	highlightUC := usecase.NewHighlightUseCase(highlightRepo, storyRepo)

	// ── Cleanup job ────────────────────────────────────────────────────────
	cleanupJob := jobs.NewStoryCleanupJob(storyRepo)
	go cleanupJob.Start(context.Background(), 5*time.Minute)

	// ── gRPC transport ─────────────────────────────────────────────────────
	storyH := grpctransport.NewStoryHandler(storyUC)
	highlightH := grpctransport.NewHighlightHandler(highlightUC)
	srv := grpctransport.NewStoryServer(storyH, highlightH)

	must(grpctransport.Run(cfg.GRPC.Port, srv), "grpc server")
}

func must(err error, msg string) {
	if err != nil {
		slog.Error(msg, "err", err)
		os.Exit(1)
	}
}
