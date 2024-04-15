CREATE OR REPLACE FUNCTION get_config_ref(param_scope TEXT, param_account_id TEXT, param_user_id TEXT, param_version_hash TEXT, param_record_kind TEXT, param_collection_key TEXT, param_item_key TEXT)
RETURNS JSONB

LANGUAGE plpgsql
AS $func$

DECLARE head_ref JSONB;
DECLARE result JSONB;

BEGIN

    RAISE NOTICE 'Called get_config_ref()';

    -- Get the head ref
    head_ref := (
        SELECT
            r.version_ref
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

    RAISE NOTICE 'Head hash: %', head_ref->>'config_version_hash';

    -- If it's still null, raise an error
    IF head_ref IS NULL THEN
        RAISE EXCEPTION 'No head version found for scope % account % and user %', param_scope, param_account_id, param_user_id;
    END IF;

    result = (
        SELECT
        jsonb_build_object(
            'node_metadata', n.node_metadata,
            'record_metadata', n.node_contents->'record_metadata',
            'record_contents', n.node_contents->'record_contents'
        )
        FROM config_nodes n
        WHERE (
            (n.scope = param_scope AND n.account_id = param_account_id)
            AND (CASE WHEN param_scope = 'user' THEN n.user_id = param_user_id ELSE n.user_id IS NULL END)
            AND (
                CASE
                    WHEN param_version_hash IS NOT NULL THEN
                        n.node_metadata->'version_ref'->>'config_version_hash' = param_version_hash
                    ELSE
                        CASE WHEN param_record_kind = 'keyed' THEN
                            n.node_metadata->'version_ref'->>'config_version_hash' = head_ref->>'config_version_hash'
                        WHEN param_record_kind = 'document' THEN
                            n.node_metadata->'version_ref'->>'config_version_hash' = head_ref->>'config_version_hash'
                        ELSE false END
                END
            )
        )
        LIMIT 1
    );

    RETURN result;
END;
$func$;
