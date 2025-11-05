CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id
ON subscriptions(user_id);

CREATE INDEX IF NOT EXISTS idx_subscriptions_service_name
ON subscriptions(service_name);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_period
ON subscriptions(user_id, start_date, end_date);
