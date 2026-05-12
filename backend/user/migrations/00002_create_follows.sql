-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS follows (
    follower_id UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    followee_id UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (follower_id, followee_id),
    CHECK (follower_id <> followee_id)
);
CREATE INDEX IF NOT EXISTS idx_follows_followee ON follows(followee_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS follows;
-- +goose StatementEnd
