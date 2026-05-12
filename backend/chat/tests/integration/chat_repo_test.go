package integration_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/bekesh/social/backend/chat/internal/domain"
	pginfra "github.com/bekesh/social/backend/chat/internal/infrastructure/postgres"
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
			"POSTGRES_DB":       "chat_test",
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

	dsn := "host=" + host + " port=" + port.Port() + " user=test password=test dbname=chat_test sslmode=disable"
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

func TestConversationRepo_CreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewConversationRepo(db)

	creatorID := uuid.New()
	conv := &domain.Conversation{
		ID:        uuid.New(),
		Type:      domain.ConvTypeDirect,
		CreatedBy: creatorID,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
	}

	err := repo.Create(context.Background(), conv)
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), conv.ID)
	require.NoError(t, err)
	assert.Equal(t, conv.ID, got.ID)
	assert.Equal(t, conv.Type, got.Type)
	assert.Equal(t, conv.CreatedBy, got.CreatedBy)
}

func TestConversationRepo_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewConversationRepo(db)

	_, err := repo.GetByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrConversationNotFound)
}

func TestConversationRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := pginfra.NewConversationRepo(db)

	conv := &domain.Conversation{
		ID:        uuid.New(),
		Type:      domain.ConvTypeDirect,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
	}
	require.NoError(t, repo.Create(context.Background(), conv))
	require.NoError(t, repo.Delete(context.Background(), conv.ID))

	_, err := repo.GetByID(context.Background(), conv.ID)
	assert.ErrorIs(t, err, domain.ErrConversationNotFound)
}

func TestConversationRepo_ListByUser(t *testing.T) {
	db := setupTestDB(t)
	convRepo := pginfra.NewConversationRepo(db)
	partRepo := pginfra.NewParticipantRepo(db)

	userID := uuid.New()
	now := time.Now()

	for i := 0; i < 3; i++ {
		conv := &domain.Conversation{
			ID:        uuid.New(),
			Type:      domain.ConvTypeDirect,
			CreatedBy: userID,
			CreatedAt: now,
		}
		require.NoError(t, convRepo.Create(context.Background(), conv))
		require.NoError(t, partRepo.Add(context.Background(), &domain.Participant{
			ConversationID: conv.ID,
			UserID:         userID,
			Role:           domain.RoleOwner,
			JoinedAt:       now,
		}))
	}

	// unrelated conv
	otherConv := &domain.Conversation{ID: uuid.New(), Type: domain.ConvTypeDirect, CreatedBy: uuid.New(), CreatedAt: now}
	require.NoError(t, convRepo.Create(context.Background(), otherConv))

	convs, err := convRepo.ListByUser(context.Background(), userID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, convs, 3)
}

func TestParticipantRepo_AddIsRemove(t *testing.T) {
	db := setupTestDB(t)
	convRepo := pginfra.NewConversationRepo(db)
	partRepo := pginfra.NewParticipantRepo(db)

	conv := &domain.Conversation{ID: uuid.New(), Type: domain.ConvTypeDirect, CreatedBy: uuid.New(), CreatedAt: time.Now()}
	require.NoError(t, convRepo.Create(context.Background(), conv))

	userID := uuid.New()
	p := &domain.Participant{ConversationID: conv.ID, UserID: userID, Role: domain.RoleMember, JoinedAt: time.Now()}

	require.NoError(t, partRepo.Add(context.Background(), p))

	ok, err := partRepo.IsParticipant(context.Background(), conv.ID, userID)
	require.NoError(t, err)
	assert.True(t, ok)

	require.NoError(t, partRepo.Remove(context.Background(), conv.ID, userID))

	ok, err = partRepo.IsParticipant(context.Background(), conv.ID, userID)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestMessageRepo_CreateAndList(t *testing.T) {
	db := setupTestDB(t)
	convRepo := pginfra.NewConversationRepo(db)
	msgRepo := pginfra.NewMessageRepo(db)

	conv := &domain.Conversation{ID: uuid.New(), Type: domain.ConvTypeDirect, CreatedBy: uuid.New(), CreatedAt: time.Now()}
	require.NoError(t, convRepo.Create(context.Background(), conv))

	senderID := uuid.New()
	for i := 0; i < 3; i++ {
		m := &domain.Message{
			ID:             uuid.New(),
			ConversationID: conv.ID,
			SenderID:       senderID,
			Text:           "hello",
			CreatedAt:      time.Now(),
		}
		require.NoError(t, msgRepo.Create(context.Background(), m))
	}

	msgs, err := msgRepo.ListByConversation(context.Background(), conv.ID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, msgs, 3)
}

func TestMessageRepo_SoftDelete(t *testing.T) {
	db := setupTestDB(t)
	convRepo := pginfra.NewConversationRepo(db)
	msgRepo := pginfra.NewMessageRepo(db)

	conv := &domain.Conversation{ID: uuid.New(), Type: domain.ConvTypeDirect, CreatedBy: uuid.New(), CreatedAt: time.Now()}
	require.NoError(t, convRepo.Create(context.Background(), conv))

	m := &domain.Message{ID: uuid.New(), ConversationID: conv.ID, SenderID: uuid.New(), Text: "hi", CreatedAt: time.Now()}
	require.NoError(t, msgRepo.Create(context.Background(), m))
	require.NoError(t, msgRepo.SoftDelete(context.Background(), m.ID))

	got, err := msgRepo.GetByID(context.Background(), m.ID)
	require.NoError(t, err)
	assert.NotNil(t, got.DeletedAt)
}

func TestMessageRepo_Search(t *testing.T) {
	db := setupTestDB(t)
	convRepo := pginfra.NewConversationRepo(db)
	msgRepo := pginfra.NewMessageRepo(db)

	conv := &domain.Conversation{ID: uuid.New(), Type: domain.ConvTypeDirect, CreatedBy: uuid.New(), CreatedAt: time.Now()}
	require.NoError(t, convRepo.Create(context.Background(), conv))

	senderID := uuid.New()
	m1 := &domain.Message{ID: uuid.New(), ConversationID: conv.ID, SenderID: senderID, Text: "hello world", CreatedAt: time.Now()}
	m2 := &domain.Message{ID: uuid.New(), ConversationID: conv.ID, SenderID: senderID, Text: "good morning", CreatedAt: time.Now()}
	require.NoError(t, msgRepo.Create(context.Background(), m1))
	require.NoError(t, msgRepo.Create(context.Background(), m2))

	results, err := msgRepo.Search(context.Background(), conv.ID, "world", 10, 0)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, m1.ID, results[0].ID)
}

func TestTransactor_RollbackOnError(t *testing.T) {
	db := setupTestDB(t)
	txDB := pginfra.NewDB(db)
	convRepo := pginfra.NewConversationRepo(db)

	conv := &domain.Conversation{ID: uuid.New(), Type: domain.ConvTypeDirect, CreatedBy: uuid.New(), CreatedAt: time.Now()}
	require.NoError(t, convRepo.Create(context.Background(), conv))

	updateTime := time.Now()
	err := txDB.WithinTransaction(context.Background(), func(ctx context.Context) error {
		if err := convRepo.UpdateLastMessageAt(ctx, conv.ID, updateTime); err != nil {
			return err
		}
		return assert.AnError
	})
	assert.Error(t, err)

	got, err := convRepo.GetByID(context.Background(), conv.ID)
	require.NoError(t, err)
	assert.Nil(t, got.LastMessageAt) // rolled back
}
