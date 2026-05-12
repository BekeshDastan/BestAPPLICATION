-- +goose Up
CREATE TABLE story_views (
  story_id  UUID REFERENCES stories(id) ON DELETE CASCADE,
  viewer_id UUID NOT NULL,
  viewed_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (story_id, viewer_id)
);

CREATE TABLE story_replies (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  story_id   UUID REFERENCES stories(id) ON DELETE CASCADE,
  user_id    UUID NOT NULL,
  text       TEXT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE story_reactions (
  story_id UUID REFERENCES stories(id) ON DELETE CASCADE,
  user_id  UUID NOT NULL,
  emoji    VARCHAR(8) NOT NULL,
  PRIMARY KEY (story_id, user_id)
);

-- +goose Down
DROP TABLE story_reactions;
DROP TABLE story_replies;
DROP TABLE story_views;
