--  scope                 | text  |           | not null | 
--  account_id            | text  |           | not null | 
--  user_id               | text  |           |          | 
--  config_reference_kind | text  |           | not null | 
--  version_ref           | jsonb |           | not null | 

-- CREATE TYPE repo_ref_result AS (
--     config_reference_kind TEXT,
--     version_hash TEXT
-- );

-- This will drop the function as well
DROP TYPE IF EXISTS repo_ref_result CASCADE;

CREATE TYPE repo_ref_result AS (
    scope TEXT,
    account_id TEXT,
    user_id TEXT,
    config_reference_kind TEXT,
    version_ref JSONB
);

CREATE OR REPLACE FUNCTION get_or_init_repo(param_scope TEXT, param_account_id TEXT, param_user_id TEXT)
-- RETURNS TABLE (scope TEXT, account_id TEXT, user_id TEXT, config_reference_kind TEXT, version_ref JSONB)
-- RETURNS SETOF config_refs
-- RETURNS TABLE (config_reference_kind TEXT, version_hash TEXT)
RETURNS SETOF repo_ref_result
LANGUAGE plpgsql
AS $$
DECLARE
    record_user_id TEXT = param_user_id;
    root_hash TEXT;
    head_hash TEXT;
    version_ref JSONB;
    version_hash TEXT;

    node_metadata JSONB;

    EMPTY_HASH TEXT = '44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a';
BEGIN

    -- Create the pgcrypto extension if it doesn't exist
    CREATE EXTENSION IF NOT EXISTS pgcrypto;

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

    root_hash = (
        SELECT
            r.version_ref->>'config_version_hash'
            FROM config_refs r
            WHERE (
                -- Scope and account match
                (r.scope = param_scope AND r.account_id = param_account_id)
                -- And user matches if it's a user scope
                AND (CASE WHEN param_scope = 'user' THEN r.user_id = param_user_id ELSE r.user_id IS NULL END)
                -- And it's the root
                AND r.config_reference_kind = 'root'
            )
            LIMIT 1
    );

    IF root_hash IS NULL THEN

        --------------------------------------------------
        -- Insert an 'empty' node into config_nodes
        --------------------------------------------------

        -- TODO: Replace this with a call to insert_dag_node_internal

        node_metadata = jsonb_build_object(
            'scope', param_scope,
            'account_id', param_account_id,
            'user_id', record_user_id,
            'node_kind', 'empty',
            'parent_ref', NULL,
            'version_ref', NULL
        );

        RAISE NOTICE 'Inserting empty node into config_nodes: %', node_metadata;

        node_metadata = insert_dag_node_internal(param_scope, param_account_id, param_user_id, node_metadata, NULL, '["root", "head"]'::JSONB);

        RAISE NOTICE 'Inserted empty node into config_nodes: %', node_metadata;

        RETURN QUERY
            SELECT param_scope scope, param_account_id account_id, record_user_id user_id, 'root' config_reference_kind, node_metadata->'version_ref'
            UNION
            SELECT param_scope scope, param_account_id account_id, record_user_id user_id, 'head' config_reference_kind, node_metadata->'version_ref';

        -- RETURN QUERY
        --     SELECT 'root' config_reference_kind, version_ref->>'config_version_hash' version_hash
        --     UNION
        --     SELECT 'head' config_reference_kind, version_ref->>'config_version_hash' version_hash;

        -- Return after outputing the version hash
        RAISE NOTICE 'Returning version hash %', version_hash;

        RETURN;

    END IF;

    head_hash = (
        SELECT
            r.version_ref->>'config_version_hash'
            FROM config_refs r
            WHERE (
                -- Scope and account match
                (r.scope = param_scope AND r.account_id = param_account_id)
                -- And user matches if it's a user scope
                AND (CASE WHEN param_scope = 'user' THEN r.user_id = param_user_id ELSE r.user_id IS NULL END)
                -- And it's the head
                AND r.config_reference_kind = 'head'
            )
            LIMIT 1
    );

    -- Sanity check, root_hash should never be null here
    IF root_hash IS NULL THEN
        RAISE EXCEPTION 'Root hash is null after initing repo';
    END IF;

    -- If we don't have a head, but we have a root, we'll just update the head to root
    IF head_hash IS NULL THEN
        INSERT INTO config_refs (scope, account_id, user_id, config_reference_kind, version_ref)
        VALUES (param_scope, param_account_id, record_user_id, 'head', (SELECT version_ref FROM config_refs WHERE version_ref->>'config_version_hash' = root_hash LIMIT 1))
        RETURNING version_ref->>'config_version_hash' INTO head_hash;
    END IF;

    RETURN QUERY SELECT r.scope, r.account_id, r.user_id, r.config_reference_kind, r.version_ref FROM config_refs r
        WHERE r.scope = param_scope AND r.account_id = param_account_id AND
            (CASE WHEN param_scope = 'user' THEN r.user_id = param_user_id ELSE r.user_id IS NULL END);

    -- RETURN QUERY SELECT r.config_reference_kind, r.version_ref->>'config_version_hash' version_hash FROM config_refs r
    --     WHERE r.scope = param_scope AND r.account_id = param_account_id AND
    --         (CASE WHEN param_scope = 'user' THEN r.user_id = param_user_id ELSE r.user_id IS NULL END);

END;
$$;
