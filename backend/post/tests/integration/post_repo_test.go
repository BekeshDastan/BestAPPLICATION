package integration_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/bekesh/social/backend/post/internal/domain"
	pginfra "github.com/bekesh/social/backend/post/internal/infrastructure/postgres"
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
			"POSTGRES_DB":       "post_test",
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

	dsn := "host=" + host + " port=" + port.Port() + " user=test password=test dbname=post_test sslmode=disable"
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

func buildTestPost(authorID uuid.UUID) *domain.Post {
	return &domain.Post{
		ID:        uuid.New(),
		AuthorID:  authorID,
		Caption:   "test caption",
		MediaURLs: []string{"https://example.com/img.jpg"},
		Tags:      []string{"go", "test"},
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
	}
}

// ── Tests ──────────────────────────────────────────────────────────────────

func TestPostRepo_CreateAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewPostRepo(db)

	authorID := uuid.New()
	p := buildTestPost(authorID)

	err := repo.Create(context.Background(), p)
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), p.ID)
	require.NoError(t, err)
	assert.Equal(t, p.ID, got.ID)
	assert.Equal(t, p.Caption, got.Caption)
	assert.Equal(t, p.MediaURLs, got.MediaURLs)
	assert.Equal(t, p.Tags, got.Tags)
}

func TestPostRepo_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewPostRepo(db)

	_, err := repo.GetByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrPostNotFound)
}

func TestPostRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewPostRepo(db)

	p := buildTestPost(uuid.New())
	require.NoError(t, repo.Create(context.Background(), p))

	p.Caption = "updated caption"
	p.Tags = []string{"updated"}
	p.UpdatedAt = time.Now().UTC().Truncate(time.Millisecond)

	require.NoError(t, repo.Update(context.Background(), p))

	got, err := repo.GetByID(context.Background(), p.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated caption", got.Caption)
	assert.Equal(t, []string{"updated"}, got.Tags)
}

func TestPostRepo_SoftDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewPostRepo(db)

	p := buildTestPost(uuid.New())
	require.NoError(t, repo.Create(context.Background(), p))
	require.NoError(t, repo.SoftDelete(context.Background(), p.ID))

	got, err := repo.GetByID(context.Background(), p.ID)
	require.NoError(t, err)
	assert.NotNil(t, got.DeletedAt)
}

func TestPostRepo_SoftDelete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewPostRepo(db)
	err := repo.SoftDelete(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrPostNotFound)
}

func TestPostRepo_ListByAuthor(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewPostRepo(db)

	authorID := uuid.New()
	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Create(context.Background(), buildTestPost(authorID)))
	}
	// Other author's post — should not appear
	require.NoError(t, repo.Create(context.Background(), buildTestPost(uuid.New())))

	posts, err := repo.ListByAuthor(context.Background(), authorID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, posts, 3)
}

func TestPostRepo_ListByAuthors_Feed(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewPostRepo(db)

	a1, a2 := uuid.New(), uuid.New()
	require.NoError(t, repo.Create(context.Background(), buildTestPost(a1)))
	require.NoError(t, repo.Create(context.Background(), buildTestPost(a2)))
	require.NoError(t, repo.Create(context.Background(), buildTestPost(uuid.New()))) // unrelated

	feed, err := repo.ListByAuthors(context.Background(), []uuid.UUID{a1, a2}, 10, 0)
	require.NoError(t, err)
	assert.Len(t, feed, 2)
}

func TestPostRepo_ListByAuthors_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewPostRepo(db)

	feed, err := repo.ListByAuthors(context.Background(), []uuid.UUID{}, 10, 0)
	require.NoError(t, err)
	assert.Empty(t, feed)
}

func TestPostRepo_Search(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewPostRepo(db)

	authorID := uuid.New()
	p := buildTestPost(authorID)
	p.Caption = "beautiful sunset photo"
	require.NoError(t, repo.Create(context.Background(), p))

	other := buildTestPost(authorID)
	other.Caption = "morning coffee"
	require.NoError(t, repo.Create(context.Background(), other))

	results, err := repo.Search(context.Background(), "sunset", 10, 0)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, p.ID, results[0].ID)
}

func TestPostRepo_IncrementDecrementLikes(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewPostRepo(db)

	p := buildTestPost(uuid.New())
	require.NoError(t, repo.Create(context.Background(), p))

	require.NoError(t, repo.IncrementLikes(context.Background(), p.ID))
	require.NoError(t, repo.IncrementLikes(context.Background(), p.ID))

	got, err := repo.GetByID(context.Background(), p.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, got.LikesCount)

	require.NoError(t, repo.DecrementLikes(context.Background(), p.ID))
	got, err = repo.GetByID(context.Background(), p.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, got.LikesCount)
}

func TestPostRepo_Transactor_RollbackOnError(t *testing.T) {
	db := setupTestDB(t)
	txDB := pginfra.NewDB(db)
	repo := pginfra.NewPostRepo(db)

	p := buildTestPost(uuid.New())
	require.NoError(t, repo.Create(context.Background(), p))

	err := txDB.WithinTransaction(context.Background(), func(ctx context.Context) error {
		if err := repo.IncrementLikes(ctx, p.ID); err != nil {
			return err
		}
		return assert.AnError // trigger rollback
	})
	assert.Error(t, err)

	got, err := repo.GetByID(context.Background(), p.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, got.LikesCount) // rolled back
}

func TestCommentRepo_CreateAndList(t *testing.T) {
	db := setupTestDB(t)
	postRepo := pginfra.NewPostRepo(db)
	commentRepo := pginfra.NewCommentRepo(db)

	p := buildTestPost(uuid.New())
	require.NoError(t, postRepo.Create(context.Background(), p))

	authorID := uuid.New()
	for i := 0; i < 3; i++ {
		c := &domain.Comment{
			ID:        uuid.New(),
			PostID:    p.ID,
			AuthorID:  authorID,
			Body:      "comment",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		require.NoError(t, commentRepo.Create(context.Background(), c))
	}

	comments, err := commentRepo.ListByPost(context.Background(), p.ID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, comments, 3)
}

func TestLikeRepo_LikeUnlike(t *testing.T) {
	db := setupTestDB(t)
	postRepo := pginfra.NewPostRepo(db)
	likeRepo := pginfra.NewLikeRepo(db)

	p := buildTestPost(uuid.New())
	require.NoError(t, postRepo.Create(context.Background(), p))

	userID := uuid.New()

	ok, err := likeRepo.IsLiked(context.Background(), p.ID, userID)
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, likeRepo.Like(context.Background(), p.ID, userID))

	ok, err = likeRepo.IsLiked(context.Background(), p.ID, userID)
	require.NoError(t, err)
	assert.True(t, ok)

	require.NoError(t, likeRepo.Unlike(context.Background(), p.ID, userID))

	ok, err = likeRepo.IsLiked(context.Background(), p.ID, userID)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestLikeRepo_DuplicateLike(t *testing.T) {
	db := setupTestDB(t)
	postRepo := pginfra.NewPostRepo(db)
	likeRepo := pginfra.NewLikeRepo(db)

	p := buildTestPost(uuid.New())
	require.NoError(t, postRepo.Create(context.Background(), p))

	userID := uuid.New()
	require.NoError(t, likeRepo.Like(context.Background(), p.ID, userID))
	err := likeRepo.Like(context.Background(), p.ID, userID)
	assert.ErrorIs(t, err, domain.ErrAlreadyLiked)
}
