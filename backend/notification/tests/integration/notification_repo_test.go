package integration_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/bekesh/social/backend/notification/internal/domain"
	pginfra "github.com/bekesh/social/backend/notification/internal/infrastructure/postgres"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	requireDocker(t)

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "notification_test",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	dsn := "host=" + host + " port=" + port.Port() + " user=test password=test dbname=notification_test sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	runMigrations(t, db)
	return db
}

func runMigrations(t *testing.T, db *sqlx.DB) {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Join(filepath.Dir(filename), "..", "..", "migrations")
	err := goose.Up(db.DB, migrationsDir)
	require.NoError(t, err)
}

func sampleNotif(userID uuid.UUID) *domain.Notification {
	return &domain.Notification{
		ID:            uuid.New(),
		UserID:        userID,
		ActorID:       uuid.New(),
		Type:          domain.NotificationTypeLike,
		ReferenceID:   uuid.New(),
		ReferenceType: "post",
		Message:       "liked your post",
		IsRead:        false,
		CreatedAt:     time.Now().UTC().Truncate(time.Millisecond),
	}
}

// ── Tests ──────────────────────────────────────────────────────────────────

func TestNotifRepo_CreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewNotificationRepo(db)
	ctx := context.Background()

	userID := uuid.New()
	n := sampleNotif(userID)

	require.NoError(t, repo.Create(ctx, n))

	got, err := repo.GetByID(ctx, n.ID)
	require.NoError(t, err)
	assert.Equal(t, n.ID, got.ID)
	assert.Equal(t, n.UserID, got.UserID)
	assert.Equal(t, n.Message, got.Message)
	assert.False(t, got.IsRead)
}

func TestNotifRepo_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewNotificationRepo(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
}

func TestNotifRepo_ListByUser(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewNotificationRepo(db)
	ctx := context.Background()

	userID := uuid.New()
	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Create(ctx, sampleNotif(userID)))
	}
	// different user — should not appear
	require.NoError(t, repo.Create(ctx, sampleNotif(uuid.New())))

	list, err := repo.ListByUser(ctx, userID, 20, 0)
	require.NoError(t, err)
	assert.Len(t, list, 3)
}

func TestNotifRepo_MarkAsRead(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewNotificationRepo(db)
	ctx := context.Background()

	userID := uuid.New()
	n := sampleNotif(userID)
	require.NoError(t, repo.Create(ctx, n))

	require.NoError(t, repo.MarkAsRead(ctx, n.ID, userID))

	got, err := repo.GetByID(ctx, n.ID)
	require.NoError(t, err)
	assert.True(t, got.IsRead)
}

func TestNotifRepo_MarkAllAsRead(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewNotificationRepo(db)
	ctx := context.Background()

	userID := uuid.New()
	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Create(ctx, sampleNotif(userID)))
	}

	require.NoError(t, repo.MarkAllAsRead(ctx, userID))

	list, err := repo.ListUnreadByUser(ctx, userID, 20, 0)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestNotifRepo_CountUnread(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewNotificationRepo(db)
	ctx := context.Background()

	userID := uuid.New()
	for i := 0; i < 4; i++ {
		require.NoError(t, repo.Create(ctx, sampleNotif(userID)))
	}

	count, err := repo.CountUnread(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, 4, count)

	require.NoError(t, repo.MarkAllAsRead(ctx, userID))

	count, err = repo.CountUnread(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestNotifRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewNotificationRepo(db)
	ctx := context.Background()

	userID := uuid.New()
	n := sampleNotif(userID)
	require.NoError(t, repo.Create(ctx, n))

	require.NoError(t, repo.Delete(ctx, n.ID, userID))

	_, err := repo.GetByID(ctx, n.ID)
	assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
}

func TestNotifRepo_DeleteAllRead(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewNotificationRepo(db)
	ctx := context.Background()

	userID := uuid.New()
	n1 := sampleNotif(userID)
	n2 := sampleNotif(userID)
	require.NoError(t, repo.Create(ctx, n1))
	require.NoError(t, repo.Create(ctx, n2))

	require.NoError(t, repo.MarkAsRead(ctx, n1.ID, userID))
	require.NoError(t, repo.DeleteAllRead(ctx, userID))

	list, err := repo.ListByUser(ctx, userID, 20, 0)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, n2.ID, list[0].ID)
}
