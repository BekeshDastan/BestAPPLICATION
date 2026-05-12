// Composition root: config → postgres → redis → nats → repos → usecases → gRPC.
package main

import (
	"log/slog"
	"os"

	"github.com/bekesh/social/backend/user/internal/config"
	"github.com/bekesh/social/backend/user/internal/infrastructure/messaging"
	pginfra "github.com/bekesh/social/backend/user/internal/infrastructure/postgres"
	redisinfra "github.com/bekesh/social/backend/user/internal/infrastructure/redis"
	grpctransport "github.com/bekesh/social/backend/user/internal/transport/grpc"
	"github.com/bekesh/social/backend/user/internal/usecase"
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
	userRepo := pginfra.NewUserRepo(db)
	tokenRepo := pginfra.NewTokenRepo(db)
	followRepo := pginfra.NewFollowRepo(db)

	// ── Redis ──────────────────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
	})
	cache := redisinfra.NewUserCache(rdb)

	// ── NATS JetStream ─────────────────────────────────────────────────────
	nc, err := nats.Connect(cfg.NATS.URL)
	must(err, "connect nats")
	defer nc.Drain()

	publisher, err := messaging.NewNATSPublisher(nc, cfg.NATS.Stream)
	must(err, "create nats publisher")

	// ── Use-cases ──────────────────────────────────────────────────────────
	authUC := usecase.NewAuthUseCase(
		userRepo, tokenRepo, cache, publisher, txDB,
		cfg.JWT.PrivateKey, cfg.JWT.PublicKey,
		cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL,
	)
	profileUC := usecase.NewProfileUseCase(userRepo, cache, publisher)
	followUC := usecase.NewFollowUseCase(userRepo, followRepo, publisher)

	// ── gRPC transport ─────────────────────────────────────────────────────
	authH := grpctransport.NewAuthHandler(authUC)
	profileH := grpctransport.NewProfileHandler(profileUC)
	followH := grpctransport.NewFollowHandler(followUC)
	srv := grpctransport.NewUserServer(authH, profileH, followH)

	must(grpctransport.Run(cfg.GRPC.Port, srv), "grpc server")
}

func must(err error, msg string) {
	if err != nil {
		slog.Error(msg, "err", err)
		os.Exit(1)
	}
}
