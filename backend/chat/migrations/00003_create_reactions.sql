-- +goose Up
CREATE TABLE message_reactions (
  message_id UUID REFERENCES messages(id) ON DELETE CASCADE,
  user_id    UUID NOT NULL,
  emoji      VARCHAR(8) NOT NULL,
  PRIMARY KEY (message_id, user_id, emoji)
);

-- +goose Down
DROP TABLE IF EXISTS message_reactions;
