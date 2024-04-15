
CREATE OR REPLACE FUNCTION get_version_chain_raw(param_scope TEXT, param_account_id TEXT, param_user_id TEXT, param_from_version TEXT, param_to_version TEXT, param_record_match_filter JSONB) --  DEFAULT '{}'::record_match_filter
RETURNS TABLE (row_number BIGINT, cur_hash TEXT, parent_hash TEXT, node_kind TEXT, node_metadata JSONB, node_contents JSONB, record_metadata JSONB, record_contents JSONB, record_match BOOL, is_matching BOOL, refs JSONB)
-- RETURNS SETOF version_chain_entry
-- RETURNS SETOF JSONB
-- RETURNS TABLE (record_collection_key TEXT, record_history JSONB)
-- RETURNS JSONB

LANGUAGE plpgsql
AS $$
DECLARE
	refs JSONB;
	from_version TEXT := param_from_version;
	to_version TEXT := param_to_version;
BEGIN
	RAISE NOTICE '>>>> Called get_version_chain() with scope %, account %, user %, from_version %, to_version %, and record_match_filter %', param_scope, param_account_id, param_user_id, from_version, to_version, jsonb_pretty(param_record_match_filter);

	refs := get_config_refs(param_scope, param_account_id, param_user_id);
	RAISE NOTICE 'Refs: %', refs;

	RAISE NOTICE 'From version IS NULL: %', from_version IS NULL;
	RAISE NOTICE 'To version IS NULL: %', to_version IS NULL;
	RAISE NOTICE 'From version is empty string: %', from_version = '';
	RAISE NOTICE 'To version is empty string: %', to_version = '';

	-- If we weren't given a FromVersion, we'll just go up to the root
	IF from_version IS NULL OR from_version = '' THEN
		from_version := refs->'by_ref'->'root'->>'config_version_hash';

		-- If it's still null, raise an error
		IF from_version IS NULL THEN
			RAISE EXCEPTION 'No root version found for scope % account % and user %', param_scope, param_account_id, param_user_id;
		END IF;
	END IF;

	-- If we weren't given a ToVersion, we'll just go down to the head
	IF to_version IS NULL OR to_version = '' THEN
		to_version = refs->'by_ref'->'head'->>'config_version_hash';

		-- If it's still null, raise an error
		IF to_version IS NULL THEN
			RAISE EXCEPTION 'No head version found for scope % account % and user %', scope, account_id, user_id;
		END IF;
	END IF;

    -- Use a recursive CTE to get the chain of versions from
    -- starting at the newest commit (ToVersion) and proceeding
    -- up to the oldest commit (FromVersion).

	RAISE NOTICE 'Using range: % .. %', from_version, to_version;

	-- RAISE NOTICE 'Record match filter: %', jsonb_pretty(param_record_match_filter);
	-- RAISE NOTICE 'Matching only: %', param_record_match_filter->'only_matching';
	-- RAISE NOTICE 'Matching only (coalesce): %', COALESCE(param_record_match_filter->'only_matching', 'false'::JSONB);
	-- -- RAISE NOTICE 'Matching only (cast): %', CAST(COALESCE(param_record_match_filter->'only_matching', 'false'::JSONB) AS BOOLEAN);
	-- -- RAISE NOTICE 'Is matching: %', (CASE WHEN (CAST(COALESCE(param_record_match_filter->'only_matching', 'false'::JSONB) AS BOOLEAN)) THEN '= record_match' ELSE 'always true' END);
	-- RAISE NOTICE 'Is matching (not equal): %', (CASE WHEN (COALESCE(param_record_match_filter->'only_matching', 'false'::JSONB) != 'true'::JSONB) THEN '= record_match' ELSE 'always true' END);

	RETURN QUERY
		WITH RECURSIVE version_chain AS (
			-- Base case, starting with ToVersion
			SELECT NULL rn, tn.node_metadata->'version_ref'->>'config_version_hash' cur_hash, NULL parent_hash, (tn.node_metadata->>'node_kind') node_kind, tn.node_metadata, (tn.node_contents) node_contents, (tn.node_contents->'record_metadata') record_metadata, (tn.node_contents->'record_contents') record_contents, FALSE record_match
				FROM config_nodes tn
				WHERE (
					-- Scope and account match
					(tn.scope = param_scope AND tn.account_id = param_account_id)
					AND
					-- User matches if it's a user scope
					(CASE WHEN param_scope = 'user' THEN tn.user_id = param_user_id ELSE tn.user_id IS NULL END)
					-- And we're starting with the ToVersion
					AND tn.node_metadata->'version_ref'->>'config_version_hash' = to_version
				)
			UNION
			-- Recursive case, going up the chain
			SELECT NULL rn, n.node_metadata->'version_ref'->>'config_version_hash' cur_hash, n.node_metadata->'parent_ref'->>'config_version_hash' parent_hash, (n.node_metadata->>'node_kind') node_kind, n.node_metadata, n.node_contents, (n.node_contents->'record_metadata') record_metadata, (n.node_contents->'record_contents') record_contents, FALSE record_match
				FROM config_nodes n
				JOIN version_chain vc ON (
					-- Join on the parent of the previous node
					(n.node_metadata->'version_ref'->>'config_version_hash' = vc.node_metadata->'parent_ref'->>'config_version_hash')
					AND (
						-- Scope and account match
						(n.scope = param_scope AND n.account_id = param_account_id)
						AND
						-- User matches if it's a user scope
						(CASE WHEN param_scope = 'user' THEN n.user_id = param_user_id ELSE n.user_id IS NULL END)
					)
					-- And we haven't reached the root (FromVersion) yet
					-- AND n.node_metadata->'current_ref'->>'config_version_hash' != from_version
					AND (n.node_metadata->>'node_kind' != 'empty' AND n.node_metadata->'parent_ref' IS NOT NULL)
				)
		),

		row_numbers AS (
			SELECT ROW_NUMBER() OVER() rn, vc.cur_hash, vc.parent_hash, vc.node_kind, vc.node_metadata, vc.node_contents, vc.record_metadata, vc.record_contents, vc.record_match
			FROM version_chain vc
		),

		match_filter AS (

			SELECT
					fvc.rn, fvc.cur_hash, fvc.parent_hash, fvc.node_kind, fvc.node_metadata, fvc.node_contents, fvc.record_metadata, fvc.record_contents,

					-- If we have a filter, we'll check if the current node matches
					CASE
						WHEN (param_record_match_filter IS NULL) THEN true ELSE (
							CASE WHEN (param_record_match_filter->>'record_kind' IS NULL) THEN true ELSE (fvc.node_contents->'record_metadata'->>'record_kind' = param_record_match_filter->>'record_kind') END
							AND
							CASE WHEN (param_record_match_filter->>'record_id' IS NULL) THEN true ELSE (fvc.record_metadata->>'record_id' = param_record_match_filter->>'record_id') END
							AND
							CASE
								WHEN fvc.node_contents->'record_metadata'->>'record_kind' = 'keyed' THEN (
									CASE WHEN (param_record_match_filter->>'record_collection_key' IS NULL) THEN true ELSE (fvc.node_contents->'record_metadata'->>'record_collection_key' = param_record_match_filter->>'record_collection_key') END
								)
								WHEN fvc.node_contents->'record_metadata'->>'record_kind' = 'document' THEN (
									CASE
										WHEN (param_record_match_filter->>'record_collection_key' IS NULL) THEN true ELSE (fvc.node_contents->'record_metadata'->>'record_collection_key' = param_record_match_filter->>'record_collection_key')
									END
									AND
									CASE
										WHEN (param_record_match_filter->>'record_item_key' IS NULL) THEN true ELSE (fvc.node_contents->'record_metadata'->>'record_item_key' = param_record_match_filter->>'record_item_key')
									END
								)
								WHEN fvc.node_contents->'record_metadata'->>'record_kind' = 'config_schema' THEN (
									CASE
										WHEN (param_record_match_filter->>'record_collection_key' IS NULL) THEN true ELSE (fvc.node_contents->'record_metadata'->>'record_collection_key' = param_record_match_filter->>'record_collection_key')
									END
									AND
									CASE
										WHEN (param_record_match_filter->>'record_item_index' IS NULL) THEN true ELSE (fvc.node_contents->'record_metadata'->>'record_item_key' = param_record_match_filter->>'record_item_key')
									END
								)
								ELSE false
							END
						)
					END record_match
				FROM row_numbers fvc
				-- WHERE (CASE (param_record_match_filter IS NULL OR param_record_match_filter->'only_matching' IS NULL OR param_record_match_filter->'only_matching' = 'false'::JSONB) WHEN TRUE THEN TRUE ELSE fvc.record_match END)
		), -- End match_filter cte
		add_refs AS (
			SELECT *,
				(CASE WHEN (COALESCE(param_record_match_filter->'only_matching', 'false'::JSONB) = 'true'::JSONB) THEN fmf.record_match ELSE TRUE END) is_matching,
				(SELECT jsonb_agg(cr.config_reference_kind)
					FROM config_refs cr WHERE
						(cr.scope = param_scope AND cr.account_id = param_account_id AND CASE WHEN param_scope = 'user' THEN cr.user_id = param_user_id ELSE cr.user_id IS NULL END)
						AND
						(cr.version_ref->>'config_version_hash' = fmf.node_metadata->'version_ref'->>'config_version_hash')
				) refs
			FROM match_filter fmf
		)
		SELECT *
		FROM add_refs ar
		WHERE (CASE WHEN (COALESCE(param_record_match_filter->'only_matching', 'false'::JSONB) = 'true'::JSONB) THEN ar.record_match ELSE TRUE END);

END;
$$;
