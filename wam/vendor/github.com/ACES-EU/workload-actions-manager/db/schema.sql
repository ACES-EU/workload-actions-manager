CREATE TYPE action_type_enum AS ENUM (
    -- action that binds a pod to a node
    'bind',
    -- action that creates a new pod a node
    'create',
    -- action that deletes a specific pod
    'delete',
    -- action that creates a new pod on the new node,
    -- waits for the pod to be ready before deleting the pod to be moved
    'move',
    -- action that swaps two pods x and y, this actions is related to pod x
    'swap_x',
    -- action that swaps two pods x and y, this actions is related to pod y
    'swap_y'
    );

-- swap_x and swap_y actions belonging to the same swap will have different ids, but the action_start_time will be the same

CREATE TYPE action_status_enum AS ENUM (
    'pending',
    'succeeded',
    'failed'
    );

CREATE TABLE workload_action
(
    id                    UUID PRIMARY KEY,
    action_type           action_type_enum   NOT NULL,
    action_status         action_status_enum NOT NULL,
    action_start_time     TIMESTAMPTZ,
    action_end_time       TIMESTAMPTZ NULL,
    action_reason         VARCHAR(255),
    pod_parent_name       VARCHAR(255),
    pod_parent_type       VARCHAR(255),
    pod_parent_uid        UUID NULL,
    created_pod_name      VARCHAR(255),
    created_pod_namespace VARCHAR(255),
    created_node_name      VARCHAR(255),
    deleted_pod_name      VARCHAR(255),
    deleted_pod_namespace VARCHAR(255),
    deleted_node_name     VARCHAR(255),
    bound_pod_name        VARCHAR(255),
    bound_pod_namespace   VARCHAR(255),
    bound_node_name       VARCHAR(255),
    created_at            TIMESTAMP,
    updated_at            TIMESTAMP NULL
);
