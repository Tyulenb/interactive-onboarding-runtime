-- name: GetTokenByHash :one
SELECT id,
       scenario_id,
       hash,
       created_at,
       expires_at
FROM onboarding.scenario_test_tokens
WHERE hash = sqlc.arg(hash)
  AND expires_at > NOW();
