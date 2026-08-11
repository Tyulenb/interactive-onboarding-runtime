-- +goose Up
CREATE TABLE onboarding.events
(
    id           UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    session_id   UUID        NOT NULL,
    step_id      UUID,
    type         TEXT        NOT NULL,
    data         JSONB       NOT NULL DEFAULT '{}',
    occurred_at  TIMESTAMPTZ NOT NULL,
    received_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT events_session_fk
        FOREIGN KEY (session_id)
            REFERENCES onboarding.sessions (id)
            ON DELETE RESTRICT,

    CONSTRAINT events_step_fk
        FOREIGN KEY (step_id)
            REFERENCES onboarding.steps (id)
            ON DELETE RESTRICT,

    CONSTRAINT events_type_check
        CHECK (type IN ('step_shown', 'step_completed', 'step_skipped', 'onboarding_completed', 'onboarding_skipped'))
);

CREATE INDEX events_session_id_idx ON onboarding.events (session_id);

-- +goose Down

DROP TABLE onboarding.events;
