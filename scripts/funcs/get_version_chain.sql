
DROP TYPE IF EXISTS record_match_filter CASCADE;
DROP TYPE IF EXISTS version_chain_entry CASCADE;

CREATE TYPE record_match_filter AS (
	scope TEXT,
	account_id TEXT,
	user_id TEXT,
	record_id TEXT,
	record_collection_key TEXT,
	record_item_key TEXT
);

CREATE TYPE version_chain_entry AS (
	cur_hash TEXT,
	parent_hash TEXT,
	node_kind TEXT,
	node_metadata JSONB,
	node_contents JSONB,
	record_metadata JSONB,
	record_contents JSONB,
	record_match BOOL
);

CREATE OR REPLACE FUNCTION get_version_chain(param_scope TEXT, param_account_id TEXT, param_user_id TEXT, param_from_version TEXT, param_to_version TEXT, param_record_match_filter JSONB) --  DEFAULT '{}'::record_match_filter
-- RETURNS TABLE (cur_hash TEXT, parent_hash TEXT, node_kind TEXT, node_metadata JSONB, node_contents JSONB, record_metadata JSONB, record_contents JSONB, record_match BOOL)
-- RETURNS SETOF version_chain_entry
RETURNS SETOF JSONB

LANGUAGE plpgsql
AS $$
DECLARE
	refs JSONB;
	from_version TEXT := param_from_version;
	to_version TEXT := param_to_version;
BEGIN
	RAISE NOTICE '>>>> Called get_version_chain() with scope %, account %, user %, from_version %, to_version %, and record_match_filter %', param_scope, param_account_id, param_user_id, from_version, to_version, jsonb_pretty(param_record_match_filter);


	RETURN QUERY
		WITH
		raw_chain AS (
			SELECT * from get_version_chain_raw(param_scope, param_account_id, param_user_id, param_from_version, param_to_version, param_record_match_filter)
		),

		record_history_cte AS (
			SELECT 
			rc.node_contents->'record_metadata'->>'record_collection_key' record_collection_key,
			jsonb_agg(jsonb_build_object(
				-- 'config_version_hash', rc.node_metadata->'version_ref'->>'config_version_hash',
				'record_collection_key', rc.node_contents->'record_metadata'->>'record_collection_key',
				'record_item_key', rc.node_contents->'record_metadata'->>'record_item_key',
				'record_metadata', rc.node_contents->'record_metadata',
				'node_metadata', rc.node_metadata,
				'record_contents', rc.node_contents->'record_contents'
			)) record_history
			FROM raw_chain rc
			GROUP BY rc.node_contents->'record_metadata'->>'record_collection_key'
		)

		SELECT jsonb_agg(jsonb_build_object(
			'row_number', c.row_number,
			'cur_hash', c.cur_hash,
			'parent_hash', c.parent_hash,
			'node_kind', c.node_kind,
			'node_metadata', c.node_metadata,
			'node_contents', c.node_contents,
			'record_metadata', c.record_metadata,
			'record_contents', c.node_contents->'record_contents',
			'record_match', c.record_match,
			'refs', c.refs,
			'record_history', rhc.record_history
			-- 'record_history', (SELECT record_history FROM record_history_cte WHERE record_history_cte.record_history->>'record_collection_key' = c.node_contents->'record_metadata'->>'record_collection_key' LIMIT 1)
		))
		-- FROM version_chain c;
		-- FROM match_filter c;
		FROM raw_chain c
		LEFT JOIN record_history_cte rhc ON rhc.record_collection_key = c.node_contents->'record_metadata'->>'record_collection_key';
		-- GROUP BY c.node_contents->'record_metadata'->>'record_collection_key';

END;
$$;
