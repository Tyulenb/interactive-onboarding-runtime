-- name: GetProjectByProjectKey :one
SELECT id,
       name,
       project_key,
       created_at,
       updated_at,
       deleted_at
FROM onboarding.projects
WHERE project_key = sqlc.arg(project_key)
  AND deleted_at IS NULL;
