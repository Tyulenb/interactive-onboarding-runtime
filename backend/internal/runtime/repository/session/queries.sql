-- name: GetSessionByScenarioAndUser :one
SELECT id,
       scenario_id,
       user_id,
       status,
       started_at,
       finished_at
FROM onboarding.sessions
WHERE scenario_id = sqlc.arg(scenario_id)
  AND user_id = sqlc.arg(user_id)
ORDER BY started_at DESC, id DESC
LIMIT 1;
