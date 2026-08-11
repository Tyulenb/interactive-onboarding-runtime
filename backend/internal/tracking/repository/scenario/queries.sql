-- name: GetScenarioByIdAndProjectKey :one
SELECT scenarios.*
FROM onboarding.scenarios AS scenarios
JOIN onboarding.projects AS projects ON projects.id = scenarios.project_id
WHERE scenarios.id = $1
  AND projects.project_key = $2
  AND scenarios.deleted_at IS NULL
  AND projects.deleted_at IS NULL;
