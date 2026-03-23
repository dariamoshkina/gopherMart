CREATE TABLE IF NOT EXISTS orders (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT       NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    order_number VARCHAR(50)  NOT NULL,
    status       VARCHAR(20)  NOT NULL DEFAULT 'NEW',
    accrual      BIGINT         NOT NULL DEFAULT 0,
    uploaded_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT chk_orders_status CHECK (status IN ('NEW', 'PROCESSING', 'PROCESSED', 'INVALID')),
    CONSTRAINT chk_orders_accrual CHECK (accrual >= 0)
);

CREATE UNIQUE INDEX idx_orders_order_number ON orders (order_number);
CREATE INDEX idx_orders_user_id ON orders (user_id);
CREATE INDEX idx_orders_status ON orders (status) WHERE status IN ('NEW', 'PROCESSING');
