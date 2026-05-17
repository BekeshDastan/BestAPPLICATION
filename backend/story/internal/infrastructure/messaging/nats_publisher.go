package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
)

type NATSPublisher struct {
	js     nats.JetStreamContext
	stream string
}

func NewNATSPublisher(nc *nats.Conn, stream string) (*NATSPublisher, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}
	_, err = js.StreamInfo(stream)
	if err != nil {
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     stream,
			Subjects: []string{"user.>", "post.>", "chat.>", "notification.>", "story.>"},
			Storage:  nats.FileStorage,
		})
		if err != nil {
			return nil, fmt.Errorf("create stream %s: %w", stream, err)
		}
	}
	return &NATSPublisher{js: js, stream: stream}, nil
}

func (p *NATSPublisher) Publish(_ context.Context, subject string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = p.js.Publish(subject, data)
	return err
}
