-- +goose Up
CREATE TABLE IF NOT EXISTS saves (
    post_id    UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (post_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_saves_user_id ON saves (user_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS saves;
