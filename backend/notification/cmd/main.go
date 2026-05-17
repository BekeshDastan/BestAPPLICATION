package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/bekesh/social/backend/notification/internal/config"
	"github.com/bekesh/social/backend/notification/internal/infrastructure/email"
	"github.com/bekesh/social/backend/notification/internal/infrastructure/messaging"
	pginfra "github.com/bekesh/social/backend/notification/internal/infrastructure/postgres"
	redisinfra "github.com/bekesh/social/backend/notification/internal/infrastructure/redis"
	grpctransport "github.com/bekesh/social/backend/notification/internal/transport/grpc"
	"github.com/bekesh/social/backend/notification/internal/usecase"
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

	notifRepo := pginfra.NewNotificationRepo(db)
	prefRepo := pginfra.NewPreferenceRepo(db)

	// ── Redis ──────────────────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
	})
	cache := redisinfra.NewNotificationCache(rdb)

	// ── NATS ───────────────────────────────────────────────────────────────
	nc, err := nats.Connect(cfg.NATS.URL)
	must(err, "connect nats")
	defer nc.Drain()

	publisher, err := messaging.NewNATSPublisher(nc, cfg.NATS.Stream)
	must(err, "create nats publisher")

	// ── Email ──────────────────────────────────────────────────────────────
	emailSender := email.NewSMTPSender(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password)

	// ── Use-cases ──────────────────────────────────────────────────────────
	notifUC := usecase.NewNotificationUseCase(notifRepo, cache, publisher)
	prefUC := usecase.NewPreferenceUseCase(prefRepo)

	// ── NATS Consumer ──────────────────────────────────────────────────────
	appURL := cfg.AppURL
	consumer, err := messaging.NewNATSConsumer(nc, notifUC, emailSender, appURL)
	must(err, "create nats consumer")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	consumer.Start(ctx)

	// ── gRPC transport ─────────────────────────────────────────────────────
	handler := grpctransport.NewNotificationHandler(notifUC, prefUC, emailSender)
	srv := grpctransport.NewServer(handler)

	must(grpctransport.Run(cfg.GRPC.Port, srv), "grpc server")
}

func must(err error, msg string) {
	if err != nil {
		slog.Error(msg, "err", err)
		os.Exit(1)
	}
}
