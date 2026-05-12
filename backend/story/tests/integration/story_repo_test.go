package integration_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/bekesh/social/backend/story/internal/domain"
	pginfra "github.com/bekesh/social/backend/story/internal/infrastructure/postgres"
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
			"POSTGRES_DB":       "story_test",
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

	dsn := "host=" + host + " port=" + port.Port() + " user=test password=test dbname=story_test sslmode=disable"
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

// ── Tests ──────────────────────────────────────────────────────────────────

func TestStoryRepo_CreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewStoryRepo(db)

	userID := uuid.New()
	s := &domain.Story{
		ID:        uuid.New(),
		UserID:    userID,
		MediaURL:  "https://cdn.example.com/story.jpg",
		MediaType: domain.MediaTypeImage,
		Caption:   "test caption",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
	}

	err := repo.Create(context.Background(), s)
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), s.ID)
	require.NoError(t, err)
	assert.Equal(t, s.ID, got.ID)
	assert.Equal(t, s.UserID, got.UserID)
	assert.Equal(t, domain.MediaTypeImage, got.MediaType)
}

func TestStoryRepo_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewStoryRepo(db)

	_, err := repo.GetByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrStoryNotFound)
}

func TestStoryRepo_SoftDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewStoryRepo(db)

	s := &domain.Story{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MediaURL:  "https://cdn.example.com/s.jpg",
		MediaType: domain.MediaTypeImage,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	require.NoError(t, repo.Create(context.Background(), s))
	require.NoError(t, repo.SoftDelete(context.Background(), s.ID))

	got, err := repo.GetByID(context.Background(), s.ID)
	require.NoError(t, err)
	assert.NotNil(t, got.DeletedAt)
}

func TestStoryRepo_ListByUser(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewStoryRepo(db)

	userID := uuid.New()
	for i := 0; i < 3; i++ {
		s := &domain.Story{
			ID:        uuid.New(),
			UserID:    userID,
			MediaURL:  "https://cdn.example.com/s.jpg",
			MediaType: domain.MediaTypeImage,
			ExpiresAt: time.Now().Add(24 * time.Hour),
			CreatedAt: time.Now(),
		}
		require.NoError(t, repo.Create(context.Background(), s))
	}

	// other user
	other := &domain.Story{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MediaURL:  "https://cdn.example.com/s.jpg",
		MediaType: domain.MediaTypeImage,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	require.NoError(t, repo.Create(context.Background(), other))

	stories, err := repo.ListByUser(context.Background(), userID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, stories, 3)
}

func TestStoryViewRepo_AddAndIsViewed(t *testing.T) {
	db := setupTestDB(t)
	storyRepo := pginfra.NewStoryRepo(db)
	viewRepo := pginfra.NewStoryViewRepo(db)

	s := &domain.Story{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MediaURL:  "https://cdn.example.com/s.jpg",
		MediaType: domain.MediaTypeImage,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	require.NoError(t, storyRepo.Create(context.Background(), s))

	viewerID := uuid.New()
	v := &domain.StoryView{StoryID: s.ID, ViewerID: viewerID, ViewedAt: time.Now()}
	require.NoError(t, viewRepo.Add(context.Background(), v))

	ok, err := viewRepo.IsViewed(context.Background(), s.ID, viewerID)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = viewRepo.IsViewed(context.Background(), s.ID, uuid.New())
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestHighlightRepo_CreateAndList(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewHighlightRepo(db)

	userID := uuid.New()
	h := &domain.Highlight{
		ID:        uuid.New(),
		UserID:    userID,
		Title:     "My Stories",
		CreatedAt: time.Now(),
	}
	require.NoError(t, repo.Create(context.Background(), h))

	got, err := repo.GetByID(context.Background(), h.ID)
	require.NoError(t, err)
	assert.Equal(t, h.Title, got.Title)

	list, err := repo.ListByUser(context.Background(), userID)
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestHighlightRepo_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewHighlightRepo(db)

	_, err := repo.GetByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrHighlightNotFound)
}

func TestTransactor_RollbackOnError(t *testing.T) {
	db := setupTestDB(t)
	txDB := pginfra.NewDB(db)
	storyRepo := pginfra.NewStoryRepo(db)

	s := &domain.Story{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MediaURL:  "https://cdn.example.com/s.jpg",
		MediaType: domain.MediaTypeImage,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	require.NoError(t, storyRepo.Create(context.Background(), s))

	err := txDB.WithinTransaction(context.Background(), func(ctx context.Context) error {
		if err := storyRepo.IncrViewsCount(ctx, s.ID); err != nil {
			return err
		}
		return assert.AnError
	})
	assert.Error(t, err)

	got, err := storyRepo.GetByID(context.Background(), s.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, got.ViewsCount) // rolled back
}
