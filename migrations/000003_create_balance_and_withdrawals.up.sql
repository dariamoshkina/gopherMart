CREATE TABLE IF NOT EXISTS balance (
    user_id   BIGINT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    current   BIGINT NOT NULL DEFAULT 0 CHECK (current >= 0),
    withdrawn BIGINT NOT NULL DEFAULT 0 CHECK (withdrawn >= 0)
);

CREATE TABLE IF NOT EXISTS withdrawals (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT            NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    order_number VARCHAR(50)       NOT NULL,
    sum          BIGINT           NOT NULL CHECK (sum > 0),
    processed_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_withdrawals_user_id ON withdrawals (user_id);
