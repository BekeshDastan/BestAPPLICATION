package integration_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bekesh/social/backend/post/internal/infrastructure/messaging"
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
		Cmd:          []string{"-js"},
		ExposedPorts: []string{"4222/tcp"},
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
	t.Cleanup(func() { nc.Drain() })

	return nc
}

func TestNATSPublisher_Publish(t *testing.T) {
	nc := setupNATS(t)

	pub, err := messaging.NewNATSPublisher(nc, "POST_TEST")
	require.NoError(t, err)

	js, err := nc.JetStream()
	require.NoError(t, err)

	sub, err := js.SubscribeSync("post.created", nats.DeliverAll())
	require.NoError(t, err)
	t.Cleanup(func() { _ = sub.Unsubscribe() })

	payload := map[string]string{"post_id": "abc123", "author_id": "user456"}
	err = pub.Publish(context.Background(), "post.created", payload)
	require.NoError(t, err)

	msg, err := sub.NextMsg(3 * time.Second)
	require.NoError(t, err)

	var got map[string]string
	require.NoError(t, json.Unmarshal(msg.Data, &got))
	assert.Equal(t, "abc123", got["post_id"])
	assert.Equal(t, "user456", got["author_id"])
}

func TestNATSPublisher_MultipleSubjects(t *testing.T) {
	nc := setupNATS(t)

	pub, err := messaging.NewNATSPublisher(nc, "POST_TEST2")
	require.NoError(t, err)

	js, err := nc.JetStream()
	require.NoError(t, err)

	subLiked, err := js.SubscribeSync("post.liked", nats.DeliverAll())
	require.NoError(t, err)
	subCommented, err := js.SubscribeSync("post.commented", nats.DeliverAll())
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = subLiked.Unsubscribe()
		_ = subCommented.Unsubscribe()
	})

	require.NoError(t, pub.Publish(context.Background(), "post.liked", map[string]string{"post_id": "p1"}))
	require.NoError(t, pub.Publish(context.Background(), "post.commented", map[string]string{"post_id": "p2"}))

	msgLiked, err := subLiked.NextMsg(3 * time.Second)
	require.NoError(t, err)
	msgCommented, err := subCommented.NextMsg(3 * time.Second)
	require.NoError(t, err)

	var likedPayload, commentedPayload map[string]string
	require.NoError(t, json.Unmarshal(msgLiked.Data, &likedPayload))
	require.NoError(t, json.Unmarshal(msgCommented.Data, &commentedPayload))

	assert.Equal(t, "p1", likedPayload["post_id"])
	assert.Equal(t, "p2", commentedPayload["post_id"])
}

func TestNATSPublisher_IdempotentStreamCreation(t *testing.T) {
	nc := setupNATS(t)

	pub1, err := messaging.NewNATSPublisher(nc, "POST_IDEM")
	require.NoError(t, err)
	pub2, err := messaging.NewNATSPublisher(nc, "POST_IDEM")
	require.NoError(t, err)

	// Both should publish without error
	require.NoError(t, pub1.Publish(context.Background(), "post.created", map[string]string{"a": "1"}))
	require.NoError(t, pub2.Publish(context.Background(), "post.deleted", map[string]string{"b": "2"}))
}
