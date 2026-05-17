-- +goose Up
CREATE TABLE IF NOT EXISTS notifications (
    id             UUID        PRIMARY KEY,
    user_id        UUID        NOT NULL,
    actor_id       UUID        NOT NULL,
    type           VARCHAR(50) NOT NULL,
    reference_id   UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    reference_type VARCHAR(50) NOT NULL DEFAULT '',
    message        TEXT        NOT NULL,
    is_read        BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id
    ON notifications (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_unread
    ON notifications (user_id, is_read)
    WHERE NOT is_read;

-- +goose Down
DROP TABLE IF EXISTS notifications;
