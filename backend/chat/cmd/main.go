package main

import (
	"log/slog"
	"os"

	"github.com/bekesh/social/backend/chat/internal/config"
	"github.com/bekesh/social/backend/chat/internal/infrastructure/messaging"
	pginfra "github.com/bekesh/social/backend/chat/internal/infrastructure/postgres"
	redisinfra "github.com/bekesh/social/backend/chat/internal/infrastructure/redis"
	grpctransport "github.com/bekesh/social/backend/chat/internal/transport/grpc"
	"github.com/bekesh/social/backend/chat/internal/usecase"
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
	convRepo := pginfra.NewConversationRepo(db)
	partRepo := pginfra.NewParticipantRepo(db)
	msgRepo := pginfra.NewMessageRepo(db)
	reactRepo := pginfra.NewReactionRepo(db)

	// ── Redis ──────────────────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
	})
	cache := redisinfra.NewChatCache(rdb)

	// ── NATS ───────────────────────────────────────────────────────────────
	nc, err := nats.Connect(cfg.NATS.URL)
	must(err, "connect nats")
	defer nc.Drain()

	publisher, err := messaging.NewNATSPublisher(nc, cfg.NATS.Stream)
	must(err, "create nats publisher")

	// ── Use-cases ──────────────────────────────────────────────────────────
	convUC := usecase.NewConversationUseCase(convRepo, partRepo, publisher, cache, txDB)
	msgUC := usecase.NewMessageUseCase(convRepo, partRepo, msgRepo, reactRepo, publisher, cache, txDB)

	// ── gRPC transport ─────────────────────────────────────────────────────
	convH := grpctransport.NewConversationHandler(convUC)
	msgH := grpctransport.NewMessageHandler(msgUC)
	streamH := grpctransport.NewStreamHandler(nc, cache)
	srv := grpctransport.NewChatServer(convH, msgH, streamH)

	must(grpctransport.Run(cfg.GRPC.Port, srv), "grpc server")
}

func must(err error, msg string) {
	if err != nil {
		slog.Error(msg, "err", err)
		os.Exit(1)
	}
}
