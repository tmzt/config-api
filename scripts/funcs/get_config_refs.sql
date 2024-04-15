CREATE OR REPLACE FUNCTION get_config_refs(param_scope TEXT, param_account_id TEXT, param_user_id TEXT)
RETURNS SETOF JSONB

LANGUAGE plpgsql
AS $func$

DECLARE refs JSONB;

BEGIN

    RAISE NOTICE 'Called get_config_refs()';

    -- Get the head ref
    RETURN QUERY (
        WITH refs AS (
            SELECT
                r.version_ref,
                r.config_reference_kind
            FROM config_refs r
            WHERE (
                -- Scope and account match
                (r.scope = param_scope AND r.account_id = param_account_id)
                -- And user matches if it's a user scope
                AND (CASE WHEN param_scope = 'user' THEN r.user_id = param_user_id ELSE r.user_id IS NULL END)
            )
        )
        SELECT jsonb_build_object(
            'by_hash', jsonb_object_agg(
                r.version_ref->>'config_version_hash',
                jsonb_build_object(
                    'version_ref', r.version_ref,
                    'config_reference_kind', r.config_reference_kind
                )
            ),
            'by_ref', jsonb_object_agg(
                r.config_reference_kind,
                jsonb_build_object(
                    'version_ref', r.version_ref,
                    'config_version_hash', r.version_ref->>'config_version_hash'
                )
            )
        )
        FROM refs r
    );

END;
$func$;
