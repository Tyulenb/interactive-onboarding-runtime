-- name: GetStepsByScenarioId :many
SELECT s.id AS scenario_id,
       s.name AS scenario_name,
       s.description AS scenario_description,
       s.page_pattern AS scenario_page_pattern,
       st.id AS step_id,
       st.step_num,
       st.title AS step_title,
       st.description AS step_description,
       st.frontend_data,
       e.id AS element_id,
       e.key AS element_key,
       e.label AS element_label,
       e.description AS element_description
FROM onboarding.scenarios AS s
LEFT JOIN onboarding.steps AS st
  ON st.scenario_id = s.id
 AND st.deleted_at IS NULL
LEFT JOIN onboarding.elements AS e
  ON e.id = st.element_id
 AND e.deleted_at IS NULL
WHERE s.id = sqlc.arg(scenario_id)
  AND s.deleted_at IS NULL
ORDER BY st.step_num, st.id;
