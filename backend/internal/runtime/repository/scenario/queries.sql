-- name: GetScenarioById :one
SELECT s.id,
       s.project_id,
       s.name,
       s.description,
       s.page_pattern,
       s.status,
       s.published_at,
       s.created_at,
       s.updated_at,
       s.deleted_at
FROM onboarding.scenarios AS s
WHERE s.id = sqlc.arg(scenario_id)
  AND s.deleted_at IS NULL;

-- name: GetScenarioByIdAndProjectKey :one
SELECT s.id,
       s.project_id,
       s.name,
       s.description,
       s.page_pattern,
       s.status,
       s.published_at,
       s.created_at,
       s.updated_at,
       s.deleted_at
FROM onboarding.scenarios AS s
JOIN onboarding.projects AS p ON p.id = s.project_id
WHERE s.id = sqlc.arg(scenario_id)
  AND p.project_key = sqlc.arg(project_key)
  AND s.deleted_at IS NULL
  AND p.deleted_at IS NULL;

-- name: GetScenariosByPagePatternAndProjectkey :many
SELECT s.id,
       s.project_id,
       s.name,
       s.description,
       s.page_pattern,
       s.status,
       s.published_at,
       s.created_at,
       s.updated_at,
       s.deleted_at
FROM onboarding.scenarios AS s
JOIN onboarding.projects AS p ON p.id = s.project_id
WHERE s.page_pattern = sqlc.arg(page_pattern)
  AND p.project_key = sqlc.arg(project_key)
  AND s.status = 'enabled'
  AND s.published_at <= NOW()
  AND s.deleted_at IS NULL
  AND p.deleted_at IS NULL
ORDER BY s.published_at, s.id;
