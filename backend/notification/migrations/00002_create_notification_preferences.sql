-- +goose Up
CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id       UUID        NOT NULL,
    type          VARCHAR(50) NOT NULL,
    email_enabled BOOLEAN     NOT NULL DEFAULT TRUE,
    push_enabled  BOOLEAN     NOT NULL DEFAULT TRUE,
    PRIMARY KEY (user_id, type)
);

-- +goose Down
DROP TABLE IF EXISTS notification_preferences;
