CREATE TABLE IF NOT EXISTS users (
    id             BIGINT       PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    login          VARCHAR(255) NOT NULL,
    password_hash  VARCHAR(255) NOT NULL,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_users_login ON users (login);
