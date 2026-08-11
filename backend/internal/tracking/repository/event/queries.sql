-- name: CreateEvent :one
WITH "active_session" AS (
  SELECT "id"
  FROM onboarding.sessions
  WHERE "id" = $2
    AND "status" = 'active'
  FOR UPDATE
)
INSERT INTO onboarding.events
  ("id", "session_id", "step_id", "type", "data", "occurred_at", "received_at")
SELECT
  $1, $2, $3, $4, $5, $6, $7
FROM "active_session"
RETURNING *;

-- name: GetEventByIdAndProjectKey :one
SELECT events.*,
       events.session_id = sqlc.arg(session_id)
         AND events.step_id IS NOT DISTINCT FROM sqlc.narg(step_id)::uuid
         AND events.type = sqlc.arg(type)
         AND events.data = sqlc.arg(data)::jsonb
         AND events.occurred_at = sqlc.arg(occurred_at) AS request_matches
FROM onboarding.events AS events
JOIN onboarding.sessions AS sessions ON sessions.id = events.session_id
JOIN onboarding.scenarios AS scenarios ON scenarios.id = sessions.scenario_id
JOIN onboarding.projects AS projects ON projects.id = scenarios.project_id
WHERE events.id = sqlc.arg(event_id)
  AND projects.project_key = sqlc.arg(project_key)
  AND projects.deleted_at IS NULL;
