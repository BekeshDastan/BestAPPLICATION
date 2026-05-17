-- +goose Up
CREATE TABLE stories (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID NOT NULL,
  media_url   TEXT NOT NULL,
  media_type  VARCHAR(8) NOT NULL CHECK (media_type IN ('image','video')),
  caption     TEXT NOT NULL DEFAULT '',
  expires_at  TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '24 hours',
  views_count INT DEFAULT 0,
  created_at  TIMESTAMPTZ DEFAULT NOW(),
  deleted_at  TIMESTAMPTZ
);
CREATE INDEX idx_stories_user   ON stories(user_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_stories_expire ON stories(expires_at)               WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE stories;
