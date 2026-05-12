package integration_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/bekesh/social/backend/user/internal/domain"
	pginfra "github.com/bekesh/social/backend/user/internal/infrastructure/postgres"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ── Container helpers ──────────────────────────────────────────────────────

func setupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	requireDocker(t)
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "test",
				"POSTGRES_PASSWORD": "test",
				"POSTGRES_DB":       "test",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	dsn := fmt.Sprintf(
		"host=%s port=%s user=test password=test dbname=test sslmode=disable",
		host, port.Port(),
	)
	db, err := sqlx.Connect("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	runMigrations(t, db)
	return db
}

func runMigrations(t *testing.T, db *sqlx.DB) {
	t.Helper()
	_, callerFile, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Join(filepath.Dir(callerFile), "..", "..", "migrations")
	require.NoError(t, goose.SetDialect("postgres"))
	require.NoError(t, goose.Up(db.DB, migrationsDir))
}

// buildTestUser returns a valid User ready to insert.
func buildTestUser() *domain.User {
	return &domain.User{
		ID:           uuid.New(),
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "$2a$04$placeholder",
		FullName:     "Alice Smith",
		Bio:          "Hello world",
		IsVerified:   true,
		IsPrivate:    false,
		CreatedAt:    time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt:    time.Now().UTC().Truncate(time.Millisecond),
	}
}

// ── UserRepo tests ─────────────────────────────────────────────────────────

func TestUserRepo_CreateAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewUserRepo(db)
	ctx := context.Background()

	u := buildTestUser()
	require.NoError(t, repo.Create(ctx, u))

	got, err := repo.GetByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)
	assert.Equal(t, u.Username, got.Username)
	assert.Equal(t, u.Email, got.Email)
	assert.Equal(t, u.FullName, got.FullName)
	assert.True(t, got.IsVerified)
}

func TestUserRepo_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewUserRepo(db)

	_, err := repo.GetByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserRepo_GetByEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewUserRepo(db)
	ctx := context.Background()

	u := buildTestUser()
	require.NoError(t, repo.Create(ctx, u))

	got, err := repo.GetByEmail(ctx, string(u.Email))
	require.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)
}

func TestUserRepo_GetByEmail_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewUserRepo(db)

	_, err := repo.GetByEmail(context.Background(), "nobody@example.com")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserRepo_GetByUsername(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewUserRepo(db)
	ctx := context.Background()

	u := buildTestUser()
	require.NoError(t, repo.Create(ctx, u))

	got, err := repo.GetByUsername(ctx, string(u.Username))
	require.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)
}

func TestUserRepo_Create_DuplicateEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewUserRepo(db)
	ctx := context.Background()

	u1 := buildTestUser()
	require.NoError(t, repo.Create(ctx, u1))

	u2 := buildTestUser()
	u2.ID = uuid.New()
	u2.Username = "bob" // different username, same email
	err := repo.Create(ctx, u2)
	assert.ErrorIs(t, err, domain.ErrEmailTaken)
}

func TestUserRepo_Create_DuplicateUsername(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewUserRepo(db)
	ctx := context.Background()

	u1 := buildTestUser()
	require.NoError(t, repo.Create(ctx, u1))

	u2 := buildTestUser()
	u2.ID = uuid.New()
	u2.Email = "bob@example.com" // different email, same username
	err := repo.Create(ctx, u2)
	assert.ErrorIs(t, err, domain.ErrUsernameTaken)
}

func TestUserRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewUserRepo(db)
	ctx := context.Background()

	u := buildTestUser()
	require.NoError(t, repo.Create(ctx, u))

	u.FullName = "Alice Updated"
	u.Bio = "Updated bio"
	u.IsPrivate = true
	u.UpdatedAt = time.Now().UTC()
	require.NoError(t, repo.Update(ctx, u))

	got, err := repo.GetByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "Alice Updated", got.FullName)
	assert.True(t, got.IsPrivate)
}

func TestUserRepo_SoftDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewUserRepo(db)
	ctx := context.Background()

	u := buildTestUser()
	require.NoError(t, repo.Create(ctx, u))

	require.NoError(t, repo.SoftDelete(ctx, u.ID))

	// GetByEmail and GetByUsername filter out deleted users
	_, err := repo.GetByEmail(ctx, string(u.Email))
	assert.ErrorIs(t, err, domain.ErrNotFound)

	_, err = repo.GetByUsername(ctx, string(u.Username))
	assert.ErrorIs(t, err, domain.ErrNotFound)

	// GetByID still returns the row (with DeletedAt set)
	got, err := repo.GetByID(ctx, u.ID)
	require.NoError(t, err)
	assert.True(t, got.IsDeleted())
}

func TestUserRepo_SoftDelete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewUserRepo(db)

	err := repo.SoftDelete(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserRepo_Search(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewUserRepo(db)
	ctx := context.Background()

	u1 := buildTestUser()
	require.NoError(t, repo.Create(ctx, u1))

	u2 := buildTestUser()
	u2.ID = uuid.New()
	u2.Username = "bob_smith"
	u2.Email = "bob@example.com"
	u2.FullName = "Bob Smith"
	require.NoError(t, repo.Create(ctx, u2))

	results, err := repo.Search(ctx, "alice", 10, 0)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, u1.ID, results[0].ID)
}

func TestUserRepo_CountFollowers(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewUserRepo(db)
	ctx := context.Background()

	follower := buildTestUser()
	followee := buildTestUser()
	followee.ID = uuid.New()
	followee.Username = "followee"
	followee.Email = "followee@example.com"

	require.NoError(t, repo.Create(ctx, follower))
	require.NoError(t, repo.Create(ctx, followee))

	// Insert a follow relationship directly
	_, err := db.ExecContext(ctx,
		`INSERT INTO follows (follower_id, followee_id) VALUES ($1, $2)`,
		follower.ID.String(), followee.ID.String(),
	)
	require.NoError(t, err)

	count, err := repo.CountFollowers(ctx, followee.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	countF, err := repo.CountFollowing(ctx, follower.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, countF)
}

// ── Transaction tests ──────────────────────────────────────────────────────

func TestTransactor_RollbackOnError(t *testing.T) {
	db := setupTestDB(t)
	tx := pginfra.NewDB(db)
	repo := pginfra.NewUserRepo(db)
	ctx := context.Background()

	u := buildTestUser()
	expectedErr := fmt.Errorf("intentional error")

	err := tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := repo.Create(txCtx, u); err != nil {
			return err
		}
		return expectedErr
	})
	assert.ErrorIs(t, err, expectedErr)

	// User must NOT be in the DB after rollback
	_, err = repo.GetByID(ctx, u.ID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
