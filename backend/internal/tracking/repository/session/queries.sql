-- name: CreateOrGetActiveSession :one
INSERT INTO onboarding.sessions
  ("id", "scenario_id", "user_id", "status", "started_at", "finished_at")
VALUES
  ($1, $2, $3, 'active', $4, NULL)
ON CONFLICT ("scenario_id", "user_id") WHERE "status" = 'active'
DO UPDATE SET "user_id" = EXCLUDED."user_id"
RETURNING *;

-- name: SelectSessionById :one
SELECT *
FROM onboarding.sessions
WHERE "id" = $1;

-- name: ChangeSessionStatus :one
UPDATE onboarding.sessions
SET "status" = $1, "finished_at" = $2
WHERE "id" = $3 AND "status" = 'active'
RETURNING *;
