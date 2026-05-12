package integration_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bekesh/social/backend/notification/internal/infrastructure/messaging"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupNATS(t *testing.T) *nats.Conn {
	t.Helper()
	requireDocker(t)

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "nats:2.10-alpine",
		ExposedPorts: []string{"4222/tcp"},
		Cmd:          []string{"-js"},
		WaitingFor:   wait.ForListeningPort("4222/tcp"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "4222")
	require.NoError(t, err)

	nc, err := nats.Connect("nats://" + host + ":" + port.Port())
	require.NoError(t, err)
	t.Cleanup(func() { _ = nc.Drain() })

	return nc
}

func TestNATSPublisher_Publish(t *testing.T) {
	nc := setupNATS(t)

	pub, err := messaging.NewNATSPublisher(nc, "SOCIAL_TEST")
	require.NoError(t, err)

	received := make(chan []byte, 1)
	sub, err := nc.Subscribe("notification.created", func(msg *nats.Msg) {
		received <- msg.Data
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	payload := map[string]string{
		"notification_id": "notif-1",
		"user_id":         "user-1",
		"type":            "like",
	}
	err = pub.Publish(context.Background(), "notification.created", payload)
	require.NoError(t, err)

	select {
	case data := <-received:
		var got map[string]string
		require.NoError(t, json.Unmarshal(data, &got))
		assert.Equal(t, "notif-1", got["notification_id"])
		assert.Equal(t, "user-1", got["user_id"])
		assert.Equal(t, "like", got["type"])
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for NATS message")
	}
}
