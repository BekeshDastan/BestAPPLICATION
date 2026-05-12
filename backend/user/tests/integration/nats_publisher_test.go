package integration_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bekesh/social/backend/user/internal/infrastructure/messaging"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ── Container helper ───────────────────────────────────────────────────────

func setupNATS(t *testing.T) *nats.Conn {
	t.Helper()
	requireDocker(t)
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "nats:2.10-alpine",
			Cmd:          []string{"-js"},
			ExposedPorts: []string{"4222/tcp"},
			WaitingFor:   wait.ForLog("Server is ready").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "4222")
	require.NoError(t, err)

	url := "nats://" + host + ":" + port.Port()
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)

	return nc
}

// ── Tests ──────────────────────────────────────────────────────────────────

func TestNATSPublisher_Publish(t *testing.T) {
	nc := setupNATS(t)

	pub, err := messaging.NewNATSPublisher(nc, "social")
	require.NoError(t, err)

	js, err := nc.JetStream()
	require.NoError(t, err)

	sub, err := js.SubscribeSync("user.registered", nats.DeliverAll())
	require.NoError(t, err)
	t.Cleanup(func() { _ = sub.Unsubscribe() })

	payload := map[string]string{"user_id": "abc-123", "email": "alice@example.com"}
	require.NoError(t, pub.Publish(context.Background(), "user.registered", payload))

	msg, err := sub.NextMsg(5 * time.Second)
	require.NoError(t, err)

	var got map[string]string
	require.NoError(t, json.Unmarshal(msg.Data, &got))
	assert.Equal(t, "abc-123", got["user_id"])
	assert.Equal(t, "alice@example.com", got["email"])
}

func TestNATSPublisher_MultipleSubjects(t *testing.T) {
	nc := setupNATS(t)

	pub, err := messaging.NewNATSPublisher(nc, "social")
	require.NoError(t, err)

	js, err := nc.JetStream()
	require.NoError(t, err)

	sub, err := js.SubscribeSync(">", nats.DeliverAll())
	require.NoError(t, err)
	t.Cleanup(func() { _ = sub.Unsubscribe() })

	subjects := []string{"user.followed", "user.unfollowed", "user.deleted"}
	for _, s := range subjects {
		require.NoError(t, pub.Publish(context.Background(), s, map[string]string{"subject": s}))
	}

	received := make(map[string]bool)
	for range subjects {
		msg, err := sub.NextMsg(5 * time.Second)
		require.NoError(t, err)
		received[msg.Subject] = true
	}

	for _, s := range subjects {
		assert.True(t, received[s], "expected subject %s to be received", s)
	}
}

func TestNATSPublisher_IdempotentStreamCreation(t *testing.T) {
	nc := setupNATS(t)

	// Creating publisher twice with same stream must not error
	_, err := messaging.NewNATSPublisher(nc, "social")
	require.NoError(t, err)

	_, err = messaging.NewNATSPublisher(nc, "social")
	require.NoError(t, err)
}
