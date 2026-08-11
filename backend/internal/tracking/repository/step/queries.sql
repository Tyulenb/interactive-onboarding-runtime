-- name: GetStepById :one
SELECT *
FROM onboarding.steps
WHERE id = $1 AND deleted_at IS NULL;
