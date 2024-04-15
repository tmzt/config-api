CREATE OR REPLACE FUNCTION insert_dag_node(param_scope TEXT, param_account_id TEXT, param_user_id TEXT, param_node_metadata JSONB, param_contents JSONB, param_update_refs JSONB)
RETURNS JSONB AS $$
-- RETURNS TABLE(node_metadata JSONB) AS $$
DECLARE
    record_user_id TEXT = param_user_id;

    node_metadata JSONB = param_node_metadata;
    node_parent_ref JSONB;

    version_hash TEXT;

    -- Constants
    EMPTY_HASH JSONB = '"44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a"';
BEGIN

    -- Validate the parameters

    IF param_scope NOT IN ('account', 'user') THEN
        RAISE EXCEPTION 'Unsupported scope %', param_scope;
    END IF;

    IF param_account_id IS NULL THEN
        RAISE EXCEPTION 'Account ID must be provided';
    END IF;

    IF param_user_id IS NULL THEN
        RAISE EXCEPTION 'User ID must be provided';
    END IF;

    IF param_scope <> 'user' THEN
        record_user_id = NULL;
    END IF;

    IF node_metadata->>'node_kind' NOT IN ('empty', 'data', 'record') THEN
        RAISE EXCEPTION 'Unsupported node kind %', node_metadata->>'node_kind';
    END IF;

    IF node_metadata->>'node_kind' = 'empty' THEN

        -- Contents must not be provided
        IF param_contents IS NOT NULL THEN
            RAISE EXCEPTION 'Contents must be NULL for empty node kind';
        END IF;

        -- Parent ref must be NULL
        IF node_metadata->'parent_ref' IS NOT NULL AND node_metadata->'parent_ref' <> 'null'::JSONB THEN
            RAISE EXCEPTION 'Parent ref must be NULL for empty node kind';
        END IF;

    ELSE

        -- Contents must be provided
        IF param_contents IS NULL THEN
            RAISE EXCEPTION 'Contents must be provided for non-empty node kind';
        END IF;

    END IF;

    -- If a parent_ref was not provided for a non-empty node, call get_or_init_repo to get the head ref
    IF (node_metadata->>'node_kind' != 'empty') AND (param_node_metadata->'parent_ref' IS NULL OR param_node_metadata->'parent_ref' = 'null'::JSONB) THEN
        RAISE NOTICE 'Parent ref not provided, getting head ref';

        node_parent_ref = (SELECT version_ref FROM get_or_init_repo(param_scope, param_account_id, param_user_id) WHERE config_reference_kind = 'head' LIMIT 1);
        node_metadata = jsonb_set(node_metadata, '{parent_ref}', node_parent_ref);
    END IF;

    RAISE NOTICE 'Node parent ref (after init): %', node_parent_ref;

    -- Call insert_dag_node_internal()

    node_metadata = (SELECT f.node_metadata FROM insert_dag_node_internal(param_scope, param_account_id, param_user_id, node_metadata, param_contents, param_update_refs) f LIMIT 1);

    -- RETURN QUERY SELECT node_metadata node_metadata;

    return node_metadata;

END;
$$ LANGUAGE plpgsql;
