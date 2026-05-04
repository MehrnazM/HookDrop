CREATE TABLE drop_responses(
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    webhook_event_id UUID NOT NULL UNIQUE REFERENCES webhook_events(id) ON DELETE CASCADE,
    status_code INT NOT NULL,
    response_body JSONB NULL,
    responded_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);