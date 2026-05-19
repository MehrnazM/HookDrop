CREATE TABLE IF NOT EXISTS webhook_events(
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    drop_id UUID NOT NULL REFERENCES drops(id) ON DELETE CASCADE,
    http_method VARCHAR(10) NOT NULL CHECK (http_method IN ('GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS')),
    path TEXT NOT NULL,
    headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    query_params JSONB NULL,
    body JSONB NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address INET NULL
);