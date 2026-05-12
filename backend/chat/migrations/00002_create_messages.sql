-- +goose Up
CREATE TABLE messages (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE,
  sender_id       UUID NOT NULL,
  reply_to_id     UUID REFERENCES messages(id),
  text            TEXT,
  media_url       TEXT,
  is_pinned       BOOLEAN DEFAULT FALSE,
  edited_at       TIMESTAMPTZ,
  created_at      TIMESTAMPTZ DEFAULT NOW(),
  deleted_at      TIMESTAMPTZ
);
CREATE INDEX idx_messages_conv ON messages(conversation_id, created_at DESC) WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS messages;
