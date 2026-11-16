-- +goose Up
CREATE TABLE tasks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    title       TEXT      NOT NULL,
    description TEXT      NOT NULL,
    status      TEXT      NOT NULL CHECK (status IN ('pending', 'in_progress', 'done')),
    user_id     INTEGER   NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    version     INTEGER   NOT NULL DEFAULT 1,
    created_at  TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL
);

CREATE INDEX idx_tasks_user_id ON tasks (user_id);
CREATE INDEX idx_tasks_status ON tasks (status);

-- +goose Down
DROP TABLE tasks;
