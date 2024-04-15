

CREATE OR REPLACE FUNCTION get_record_list(param_scope TEXT, param_account_id TEXT, param_user_id TEXT, param_from_version TEXT, param_to_version TEXT, param_match_filter JSONB)
  RETURNS JSONB
  LANGUAGE plpgsql
  AS $func$

DECLARE
  versions JSONB;

BEGIN

  -- refs := get_config_refs();

  -- head_hash := refs->'by_ref'->>'head';

  versions := get_version_chain(param_scope, param_account_id, param_user_id, param_from_version, param_to_version, param_match_filter);

  RAISE NOTICE '***** get_record_list(): versions: \n%\n', jsonb_pretty(versions);

  RETURN (
    WITH
      entries AS (
        SELECT value entry FROM jsonb_array_elements(versions)
      ),
      records AS (
        SELECT DISTINCT ON(record_kind, record_collection_key, record_item_key)
          -- entry->'row_number' row_number,
          entry->'record_metadata'->'record_kind' record_kind,
          -- entry->'record_metadata'->'record_id' record_id,
          entry->'record_metadata'->'record_collection_key' record_collection_key,
          entry->'record_metadata'->'record_item_key' record_item_key,
          -- entry->'node_metadata' node_metadata,
          entry->'record_contents' record_contents,
          entry->'record_history' record_history
        FROM entries
        GROUP BY record_kind, record_collection_key, record_item_key, record_contents, record_history
      )
      -- objects AS (
      -- SELECT DISTINCT ON(record_kind, record_collection_key, record_item_key)
      SELECT
        jsonb_agg(
          jsonb_build_object(
            -- 'row_number', row_number,
            'record_kind', record_kind,
            -- 'record_id', record_id,
            'record_collection_key', record_collection_key,
            'record_item_key', record_item_key,
            -- 'node_metadata', node_metadata,
            'record_contents', record_contents,
            'record_history', record_history
          )
        ) record
      FROM records
      -- GROUP BY record_kind, record_collection_key, record_item_key, record_contents, record_history
  );

      -- )
    -- SELECT jsonb_agg(o.*) FROM objects o
    -- GROUP BY record->'record_collection_key', record->'record_item_key'
  -- );

END;
$func$;
