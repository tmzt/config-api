
-- get_latest_record()
CREATE OR REPLACE FUNCTION get_latest_record(param_scope TEXT, param_account_id TEXT, param_user_id TEXT, param_from_version TEXT, param_to_version TEXT, param_record_kind TEXT, param_collection_key TEXT, param_item_key TEXT)
RETURNS JSONB

LANGUAGE plpgsql
AS $func$
DECLARE

    match_filter JSONB = jsonb_build_object(
        'scope', param_scope,
        'account_id', param_account_id,
        -- 'user_id', param_user_id,
        'record_id', NULL,
        'record_collection_key', param_collection_key,
        'record_item_key', param_item_key
    );

    versions JSONB;
    result JSONB;

BEGIN

    IF scope = 'user' THEN
        match_filter := jsonb_set(match_filter, '{user_id}', to_jsonb(param_user_id));
    END IF;

    result = get_latest_record_with_match_filter(param_scope, param_account_id, param_user_id, param_from_version, param_to_version, match_filter);

    -- versions := get_version_chain(param_scope, param_account_id, param_user_id, param_from_version, param_to_version, match_filter);

    -- result := (
    --     SELECT f.value FROM jsonb_array_elements(versions) f
    --     WHERE f.value->'record_match' = 'true'::JSONB
    --     LIMIT 1
    -- );

    RAISE NOTICE 'Latest record: %', jsonb_pretty(result);

    RETURN result;

END;
$func$;

-- get_latest_record_with_match_filter()
CREATE OR REPLACE FUNCTION get_latest_record_with_match_filter(param_scope TEXT, param_account_id TEXT, param_user_id TEXT, param_from_version TEXT, param_to_version TEXT, param_record_match_filter JSONB)
RETURNS JSONB
LANGUAGE plpgsql
AS $func$

DECLARE
    versions JSONB;
    result JSONB;

BEGIN

    RAISE NOTICE 'Getting latest record with match filter: %', jsonb_pretty(param_record_match_filter);
    RAISE NOTICE 'Calling get_version_chain() with parameters: %, %, %, %, %, %', param_scope, param_account_id, param_user_id, param_from_version, param_to_version, param_record_match_filter;
    RAISE NOTICE 'SELECT * FROM get_version_chain(%, %, %, %, %, %);', param_scope, param_account_id, param_user_id, param_from_version, param_to_version, param_record_match_filter;

    versions := get_version_chain(param_scope, param_account_id, param_user_id, param_from_version, param_to_version, param_record_match_filter);

    result := (
        SELECT f.value FROM jsonb_array_elements(versions) f
        WHERE f.value->'record_match' = 'true'::JSONB
        LIMIT 1
    );

    -- If the result is NULL, return an empty record set
    IF result IS NULL THEN
        result := 'null'::JSONB;
    END IF;

    RAISE NOTICE 'Latest record: %', jsonb_pretty(result);

    RETURN result;

END;
$func$;
