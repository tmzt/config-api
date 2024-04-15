CREATE OR REPLACE FUNCTION set_record_values(param_scope TEXT, param_account_id TEXT, param_user_id TEXT, param_record_kind TEXT, param_collection_key TEXT, param_item_key TEXT, param_values JSONB, param_merge_mode TEXT DEFAULT 'deepmerge')
RETURNS JSONB AS $func$
DECLARE
    -- refs config_refs[];
    refs JSONB;

    match_filter JSONB;
    versions JSONB;

    head_version_ref JSONB;
    parent_node JSONB;

    logical_parent_hash TEXT;

    starting_values JSONB = '{}';

    record_contents JSONB = '{}';

    node_contents JSONB;
    node_metadata JSONB;

    node_record_metadata JSONB;

    inserted_node_metadata JSONB;
    result JSONB;
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

    IF param_record_kind NOT IN ('keyed', 'document', 'config_schema', 'config_schema_association') THEN
        RAISE EXCEPTION 'Unsupported record kind %', param_record_kind;
    END IF;

    IF param_collection_key IS NULL THEN
        RAISE EXCEPTION 'Collection key must be provided';
    END IF;

    IF param_record_kind = 'document' THEN
        IF param_item_key IS NULL THEN
            RAISE EXCEPTION 'Item key must be provided for document record kind';
        END IF;
    END IF;

    IF param_merge_mode NOT IN ('replace_all', 'deepmerge') THEN
        RAISE EXCEPTION 'Unsupported merge mode %', param_merge_mode;
    END IF;

    IF param_values IS NULL THEN
        RAISE EXCEPTION 'Values must be provided';
    END IF;

    -- Find existing record
    -- For now, assume that the head version is the one we want to modify

    -- SELECT * INTO refs FROM get_or_init_repo(param_scope, param_account_id, param_user_id);
    -- refs = get_or_init_repo(param_scope, param_account_id, param_user_id);

    -- refs = ARRAY(
    --      SELECT jsonb_build_object('kind', r.config_reference_kind, 'version_ref', r.version_ref)
    --         FROM get_or_init_repo(param_scope, param_account_id, param_user_id) r
    -- );

    refs = (SELECT jsonb_object_agg(r.config_reference_kind, r.version_ref)
        FROM get_or_init_repo(param_scope, param_account_id, param_user_id) r);

    RAISE NOTICE 'Refs: %', refs;

    -- head_version_ref = (SELECT version_ref FROM refs WHERE config_reference_kind = 'head');
    head_version_ref = refs->'head';
    RAISE NOTICE 'Head version ref: %', head_version_ref;

    -- SELECT * INTO parent_node FROM config_nodes n WHERE n.node_metadata->'version_ref'->>'config_version_hash' = head_version_ref->>'config_version_hash' LIMIT 1;

    parent_node = (SELECT to_jsonb(n) parent_node FROM config_nodes n WHERE n.node_metadata->'version_ref'->>'config_version_hash' = head_version_ref->>'config_version_hash' LIMIT 1);

    RAISE NOTICE 'Parent node: %', parent_node;

    -- Find the 'logical' parent record, which is to say
    -- the most recent record with the same kind,
    -- collection key, and item key

    match_filter := jsonb_build_object(
        'record_kind', param_record_kind,
        'record_collection_key', param_collection_key
    );
    IF param_record_kind = 'document' THEN
        match_filter := jsonb_set(match_filter, '{record_item_key}', to_jsonb(param_item_key));
    END IF;

    versions = get_version_chain(param_scope, param_account_id, param_user_id, NULL::TEXT, NULL::TEXT, match_filter);

    logical_parent_hash = (
        SELECT value->'version_ref'->>'config_version_hash'
        FROM jsonb_array_elements(versions)
        WHERE value->'record_match' = 'true'::JSONB
        LIMIT 1
    );

    RAISE NOTICE 'Logical parent hash: %', logical_parent_hash;

    IF logical_parent_hash IS NOT NULL THEN
        starting_values = (SELECT n.node_contents->'record_contents' FROM config_nodes n WHERE n.node_metadata->'version_ref'->>'config_version_hash' = logical_parent_hash LIMIT 1);
    END IF;

    RAISE NOTICE 'Starting values: %', starting_values;

    -- -- Verify the record kind and required keys match
    -- IF parent_node->'node_metadata'->>'node_kind' NOT IN ('empty', 'record') THEN
    --     RAISE EXCEPTION 'Parent node is not empty or a record';
    -- END IF;

    -- IF parent_node->'node_metadata'->>'node_kind' = 'empty' THEN
    --     record_contents = param_values;
    -- ELSE
    --     -- Validate the record and perform merge into record_contents

    --         -- Validate the parent record kind and collection key

    --         -- These rules are wrong if the DAG allows more than one type of record
    --         -- instead of parent, we need the most recent record with the same
    --         -- kind, collection key, and item key

    --         IF parent_node->'node_contents'->'record_metadata'->>'record_kind' != param_record_kind THEN
    --             RAISE EXCEPTION 'Parent record kind % does not match param record kind %', parent_node->'node_contents'->>'record_kind', param_record_kind;
    --         END IF;

    --         IF parent_node->'node_contents'->'record_metadata'->>'record_collection_key' != param_collection_key THEN
    --             RAISE EXCEPTION 'Parent collection key % does not match param collection key %', parent_node->'node_contents'->>'record_collection_key', param_collection_key;
    --         END IF;

    --         IF param_record_kind = 'document' THEN
    --             IF parent_node->'node_contents'->'record_metadata'->>'record_item_key' != param_item_key THEN
    --                 RAISE EXCEPTION 'Parent item key % does not match param item key % (for document record kind)', parent_node->'node_contents'->>'record_item_key', param_item_key;
    --             END IF;
    --         END IF;

    --         -- Merge the values using the specified merge mode
    --         IF param_merge_mode = 'replace_all' THEN
    --             record_contents = param_values;
    --         ELSIF param_merge_mode = 'deepmerge' THEN
    --             -- Use the deep merge function from https://gist.github.com/phillip-haydon/54871b746201793990a18717af8d70dc#file-jsonb_merge-sql
    --             record_contents = jsonb_merge(parent_node->'node_contents'->'record_contents', param_values);
    --         END IF;

    --         RAISE NOTICE 'Record contents after merge: %', record_contents;
        
    -- END IF;

    IF param_merge_mode = 'replace_all' THEN
        record_contents = param_values;
    ELSIF param_merge_mode = 'deepmerge' THEN
        -- Use the v8 engine to merge the JSON objects
        record_contents = jsonb_merge(starting_values, param_values);
    END IF;

    RAISE NOTICE 'Final record contents: %', record_contents;

    -- Construct the new node

    node_metadata = jsonb_build_object(
        'node_kind', 'record',
        'parent_ref', parent_node->'node_metadata'->'version_ref'
    );
    RAISE NOTICE 'Node metadata: %', node_metadata;

    node_record_metadata = jsonb_build_object(
        'record_kind', param_record_kind,
        'record_collection_key', param_collection_key,
        'record_item_key', param_item_key
    );
    RAISE NOTICE 'Node record metadata: %', node_record_metadata;

    node_contents = jsonb_build_object(
        'record_metadata', node_record_metadata,
        'record_contents', record_contents
    );
    RAISE NOTICE 'Node contents: %', node_contents;

    -- Insert the new node

    inserted_node_metadata = (SELECT r.node_metadata FROM insert_dag_node(param_scope, param_account_id, param_user_id, node_metadata, node_contents, '["head"]') r LIMIT 1);
    RAISE NOTICE 'Inserted node metadata: %', inserted_node_metadata;

    result = jsonb_build_object(
        'node_metadata', inserted_node_metadata,
        'node_contents', node_contents
    );
    RAISE NOTICE 'set_record_values(): returning result: %', result;

    RETURN result;

END;
$func$ LANGUAGE plpgsql;
