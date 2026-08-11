-- +goose Up

CREATE TABLE onboarding.sessions
(
    id          UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    scenario_id UUID        NOT NULL,
    user_id     TEXT        NOT NULL,
    status      TEXT        NOT NULL,
    started_at  TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,

    CONSTRAINT sessions_status_check
        CHECK (status IN ('active', 'completed', 'skipped')),

    CONSTRAINT sessions_scenario_fk
        FOREIGN KEY (scenario_id)
            REFERENCES onboarding.scenarios (id)
            ON DELETE RESTRICT
);

CREATE UNIQUE INDEX sessions_active_scenario_user_unique
    ON onboarding.sessions (scenario_id, user_id)
    WHERE status = 'active';

-- +goose Down

DROP TABLE onboarding.sessions;
