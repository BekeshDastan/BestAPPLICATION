package messaging

import (
	"context"
	"encoding/json"

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
	if _, err = js.AddStream(&nats.StreamConfig{
		Name:     stream,
		Subjects: []string{">"},
	}); err != nil && err != nats.ErrStreamNameAlreadyInUse {
		return nil, err
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
