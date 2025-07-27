-- name: CreateAction :one
INSERT INTO workload_action (
    id,
    action_type,
    action_status,
    action_start_time,
    action_end_time,
    action_reason,
    pod_parent_name,
    pod_parent_type,
    pod_parent_uid,
    created_pod_name,
    created_pod_namespace,
    created_node_name,
    deleted_pod_name,
    deleted_pod_namespace,
    deleted_node_name,
    bound_pod_name,
    bound_pod_namespace,
    bound_node_name,
    created_at
) VALUES (
             $1, $2, $3, NOW(), $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, NOW()
         )
RETURNING *;

-- name: CreateActionStartTime :one
INSERT INTO workload_action (
    id,
    action_type,
    action_status,
    action_start_time,
    action_end_time,
    action_reason,
    pod_parent_name,
    pod_parent_type,
    pod_parent_uid,
    created_pod_name,
    created_pod_namespace,
    created_node_name,
    deleted_pod_name,
    deleted_pod_namespace,
    deleted_node_name,
    bound_pod_name,
    bound_pod_namespace,
    bound_node_name,
    created_at
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, NOW()
         )
RETURNING *;

-- name: GetAction :one
SELECT * FROM workload_action
WHERE id = $1 LIMIT 1;

-- name: UpdateAction :one
UPDATE workload_action
SET
    action_type = $2,
    action_status = $3,
    action_end_time = $4,
    action_reason = $5,
    pod_parent_name = $6,
    pod_parent_type = $7,
    pod_parent_uid = $8,
    created_pod_name = $9,
    created_pod_namespace = $10,
    created_node_name = $11,
    deleted_pod_name = $12,
    deleted_pod_namespace = $13,
    deleted_node_name = $14,
    bound_pod_name = $15,
    bound_pod_namespace = $16,
    bound_node_name = $17,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteAction :exec
DELETE FROM workload_action
WHERE id = $1;

-- name: ListActions :many
SELECT * FROM workload_action
ORDER BY created_at DESC;

-- name: ListActionsByTypeAndStatus :many
SELECT * FROM workload_action
WHERE action_type = $1 AND action_status = $2
ORDER BY created_at DESC;