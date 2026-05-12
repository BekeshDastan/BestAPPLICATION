-- +goose Up
CREATE TABLE conversations (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  type            VARCHAR(8) NOT NULL CHECK (type IN ('direct','group')),
  name            VARCHAR(100),
  avatar_url      TEXT,
  created_by      UUID NOT NULL,
  last_message_at TIMESTAMPTZ,
  created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE conversation_participants (
  conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE,
  user_id         UUID NOT NULL,
  role            VARCHAR(8) DEFAULT 'member' CHECK (role IN ('owner','admin','member')),
  joined_at       TIMESTAMPTZ DEFAULT NOW(),
  last_read_at    TIMESTAMPTZ,
  unread_count    INT DEFAULT 0,
  PRIMARY KEY (conversation_id, user_id)
);
CREATE INDEX idx_participants_user ON conversation_participants(user_id);

-- +goose Down
DROP TABLE IF EXISTS conversation_participants;
DROP TABLE IF EXISTS conversations;
