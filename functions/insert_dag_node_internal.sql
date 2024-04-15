CREATE OR REPLACE FUNCTION insert_dag_node_internal(param_scope TEXT, param_account_id TEXT, param_user_id TEXT, param_node_metadata JSONB, param_contents JSONB, param_update_refs JSONB)
-- Safe to call from get_or_init_repo
RETURNS TABLE(node_metadata JSONB) AS $$
DECLARE
    record_user_id TEXT = param_user_id;

    node_metadata JSONB = param_node_metadata;

    -- node_parent_ref JSONB = param_node_metadata->'parent_ref';
    -- node_version_ref JSONB = param_node_metadata->'version_ref';

    node_version_ref JSONB;

    version_hash TEXT;

    ref_kind TEXT;

    -- Constants
    EMPTY_HASH JSONB = '"44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a"';
BEGIN

    -- Create the pgcrypto extension if it doesn't exist
    CREATE EXTENSION IF NOT EXISTS pgcrypto;

    RAISE NOTICE 'Called insert_dag_node_internal()';

    IF param_scope <> 'user' THEN
        record_user_id = NULL;
    END IF;

    -- Validate the parameters

    RAISE NOTICE 'Node metadata: %', node_metadata;
    RAISE NOTICE 'Node metadata -> version_ref: %', node_metadata->'version_ref';
    RAISE NOTICE 'Node metadata -> parent_ref: %', node_metadata->'parent_ref';

    IF param_scope NOT IN ('account', 'user') THEN
        RAISE EXCEPTION 'Unsupported scope %', param_scope;
    END IF;

    IF param_account_id IS NULL THEN
        RAISE EXCEPTION 'Account ID must be provided';
    END IF;

    IF param_user_id IS NULL THEN
        RAISE EXCEPTION 'User ID must be provided';
    END IF;

    IF (node_metadata IS NULL OR node_metadata = 'null'::JSONB) THEN
        RAISE EXCEPTION 'Node metadata must be provided';
    END IF;

    IF node_metadata->>'scope' <> param_scope THEN
        RAISE EXCEPTION 'Scope mismatch: metadata scope % != param scope %', node_metadata->>'scope', param_scope;
    END IF;

    IF node_metadata->>'account_id' <> param_account_id THEN
        RAISE EXCEPTION 'Account ID mismatch: metadata account_id % != param account_id %', node_metadata->>'account_id', param_account_id;
    END IF;

    IF param_scope = 'user' AND ((node_metadata->>'user_id') IS NULL OR (node_metadata->>'user_id' = 'null')) THEN
        RAISE EXCEPTION 'User ID must be provided for user scope';
    END IF;
    IF param_scope = 'user' AND (node_metadata->>'user_id') <> param_user_id THEN
        RAISE EXCEPTION 'User ID mismatch: metadata user_id % != param user_id %', node_metadata->>'user_id', param_user_id;
    END IF;

    -- Node kind must be valid
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

        -- Parent ref must be provided
        IF node_metadata->'parent_ref' IS NULL OR node_metadata->'parent_ref' = 'null'::JSONB THEN
            RAISE EXCEPTION 'Parent ref must be provided for non-empty node kind';
        END IF;

        -- Parent ref config_node_version must be provided
        -- TODO: check that this actually works
        IF node_metadata->'version_ref'->>'config_version_hash' = 'null' THEN
            RAISE EXCEPTION 'Parent ref config_node_version must be provided for non-empty node kind';
        END IF;

    END IF;

    -- Build the version object if not provided

    -- IF node_version_ref IS NULL OR node_version_ref = 'null'::JSONB THEN

    IF node_metadata->'version_ref' IS NULL OR node_metadata->'version_ref' = 'null'::JSONB THEN

        node_version_ref = jsonb_build_object(
            'created_at', (to_jsonb(now()))::JSONB,
            'created_by', param_user_id,
            'scope', param_scope,
            'account_id', param_account_id,
            'config_version_hash', EMPTY_HASH,
            -- Do this to prevent conflicts until we remove the config_version_id field
            'config_version_id', gen_random_uuid()
        );

        IF param_scope = 'user' THEN
            node_version_ref = jsonb_set(node_version_ref, '{user_id}', to_jsonb(param_user_id));
        ELSE
            node_version_ref = jsonb_set(node_version_ref, '{user_id}', 'null');
        END IF;

        node_metadata = jsonb_set(node_metadata, '{version_ref}', node_version_ref);
    END IF;

    -- Fill in the node metadata object

    -- Set the committed at/by fields
    -- node_metadata = jsonb_set(node_metadata, '{committed_at}', ('"' || to_jsonb(now())::TEXT || '"')::JSONB);
    node_metadata = jsonb_set(node_metadata, '{committed_at}', (to_jsonb(now()))::JSONB);

    node_metadata = jsonb_set(node_metadata, '{committed_by}', ('"' || param_user_id || '"')::JSONB);

    -- If a config_version_id was not provided, generate one
    -- do this to prevent conflicts until we remove the config_version_id field
    RAISE NOTICE 'Node metadata -> version_ref -> config_version_id: %', node_metadata->'version_ref'->>'config_version_id';

    IF node_metadata->'version_ref'->>'config_version_id' IS NULL THEN
        RAISE NOTICE 'Generating config_version_id';
        node_metadata = jsonb_set(node_metadata, '{version_ref, config_version_id}', ('"' || gen_random_uuid() || '"')::JSONB);
    END IF;

    --
    -- Compute a hash (SHA256) of the node_metadata object
    --

    -- Set the hash to the empty hash before hashing
    node_metadata = jsonb_set(node_metadata, '{version_ref, config_version_hash}', EMPTY_HASH);

    -- Remove the \x prefix from the resulting hash
    version_hash = substr(digest(node_metadata::TEXT, 'sha256')::TEXT, 3);

    -- Update the version_ref with the hash
    node_metadata = jsonb_set(node_metadata, '{version_ref, config_version_hash}', ('"' || version_hash || '"')::JSONB);

    RAISE NOTICE 'Final node metadata: %', node_metadata;

    -- 
    -- Final sanity check
    -- 

    -- TODO: Make sure both cases handled by constraints, including NULL and 'null'::JSONB
    -- and the config_version_hash field

    RAISE NOTICE 'Node metadata: %', node_metadata;

    RAISE NOTICE 'Node kind (->>): %', node_metadata->>'node_kind';

    IF node_metadata->>'node_kind' = 'empty' THEN
        IF (node_metadata->'parent_ref' IS NOT NULL AND node_metadata->'parent_ref' <> 'null'::JSONB) THEN
            RAISE EXCEPTION 'Sanity check failed: Parent ref must be NULL for empty node kind';
        END IF;
    ELSE
        IF (node_metadata->'parent_ref' IS NULL OR node_metadata->'parent_ref' = 'null'::JSONB) THEN
            RAISE EXCEPTION 'Sanity check failed: Parent ref must be provided for non-empty node kind';
        END IF;
    END IF;

    --
    -- Insert the node object into config_nodes
    --

    INSERT INTO config_nodes (scope, account_id, user_id, created_at, created_by, node_metadata, node_contents)
    VALUES (param_scope, param_account_id, record_user_id, now(), param_user_id, node_metadata, param_contents);

    -- If the param_update_refs is not provided, just return the new node metadata
    IF (param_update_refs IS NULL OR param_update_refs = 'null'::JSONB) THEN
        RAISE NOTICE 'Not updating refs';
        RETURN QUERY SELECT node_metadata node_metadata;
        RETURN;
    END IF;

    -- First check that this is a JSONB array
    IF NOT jsonb_typeof(param_update_refs) = 'array' THEN
        RAISE EXCEPTION 'param_update_refs must be a JSONB array';
    END IF;

    RAISE NOTICE 'param_update_refs: %', param_update_refs;

    -- Loop through the array
    FOR i IN 0..jsonb_array_length(param_update_refs) - 1 LOOP
        -- Get the element
        ref_kind = param_update_refs->>i;

        -- Check if the element is a valid config_reference_kind
        IF ref_kind NOT IN ('root', 'head') THEN
            RAISE EXCEPTION 'Invalid config_reference_kind %', ref_kind;
        END IF;

        RAISE NOTICE 'Updating ref_kind % -> %s', ref_kind, node_metadata->'version_ref'->>'config_version_hash';

        -- Upsert the corresponding row in config_refs

        IF param_scope = 'user' THEN
            INSERT INTO config_refs (scope, account_id, user_id, config_reference_kind, version_ref)
            VALUES (param_scope, param_account_id, record_user_id, ref_kind, node_metadata->'version_ref')
            ON CONFLICT (account_id, user_id, config_reference_kind) WHERE scope = 'user'
            DO UPDATE SET version_ref = node_metadata->'version_ref';
        ELSE
            INSERT INTO config_refs (scope, account_id, config_reference_kind, version_ref)
            VALUES (param_scope, param_account_id, ref_kind, node_metadata->'version_ref')
            ON CONFLICT (account_id, config_reference_kind) WHERE scope = 'account'
            DO UPDATE SET version_ref = node_metadata->'version_ref';
        END IF;

    END LOOP;

    -- RETURN QUERY SELECT node_metadata node_metadata;
    RETURN node_metadata;
END;
$$ LANGUAGE plpgsql;
