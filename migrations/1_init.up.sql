CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE subscriptions (
    id SERIAL PRIMARY KEY,
    service_name TEXT NOT NULL,
    price INT NOT NULL,
    user_id UUID NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE
);