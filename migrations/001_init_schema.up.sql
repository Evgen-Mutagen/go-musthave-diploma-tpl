DO $$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'order_status') THEN
            CREATE TYPE order_status AS ENUM (
                'NEW',
                'PROCESSING',
                'INVALID',
                'PROCESSED'
                );
        END IF;
    END$$;

CREATE TABLE IF NOT EXISTS users (
                                     id BIGSERIAL PRIMARY KEY,
                                     login TEXT NOT NULL UNIQUE,
                                     password_hash TEXT NOT NULL,
                                     balance DOUBLE PRECISION NOT NULL DEFAULT 0,
                                     withdrawn DOUBLE PRECISION NOT NULL DEFAULT 0,
                                     created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS orders (
                                      number TEXT PRIMARY KEY,
                                      user_id BIGINT NOT NULL REFERENCES users(id),
                                      status order_status NOT NULL DEFAULT 'NEW'::order_status,
                                      accrual DOUBLE PRECISION NOT NULL DEFAULT 0,
                                      uploaded_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS orders_user_id_idx ON orders(user_id);
CREATE INDEX IF NOT EXISTS orders_status_idx ON orders(status);
CREATE INDEX IF NOT EXISTS orders_uploaded_at_idx ON orders(uploaded_at);

CREATE TABLE IF NOT EXISTS withdrawals (
                                           id BIGSERIAL PRIMARY KEY,
                                           order_number TEXT NOT NULL,
                                           user_id BIGINT NOT NULL REFERENCES users(id),
                                           sum DOUBLE PRECISION NOT NULL,
                                           processed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS withdrawals_user_id_idx ON withdrawals(user_id);
CREATE INDEX IF NOT EXISTS withdrawals_processed_at_idx ON withdrawals(processed_at);
