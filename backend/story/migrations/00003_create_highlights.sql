-- +goose Up
CREATE TABLE highlights (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID NOT NULL,
  title      VARCHAR(50) NOT NULL,
  cover_url  TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_highlights_user ON highlights(user_id);

CREATE TABLE highlight_stories (
  highlight_id UUID REFERENCES highlights(id) ON DELETE CASCADE,
  story_id     UUID REFERENCES stories(id) ON DELETE CASCADE,
  position     SMALLINT NOT NULL,
  PRIMARY KEY (highlight_id, story_id)
);

-- +goose Down
DROP TABLE highlight_stories;
DROP TABLE highlights;
