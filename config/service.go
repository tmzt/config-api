package config

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/tmzt/config-api/util"
	"gorm.io/gorm"
)

type ConfigService struct {
	logger util.SetRequestLogger
	db     *gorm.DB
	rdb    *redis.Client

	dagService *ConfigDagService
	// handleService        *configSettingHandleService
	diffService          *ConfigDiffService
	configContextService *ConfigContextService
	configSchemaService  *ConfigSchemaService
}

func NewConfigService(db *gorm.DB, rdb *redis.Client, cacheService *util.CacheService) *ConfigService {
	logger := util.NewLogger("ConfigService", 0)

	logger.Printf("NewConfigService: db: %v\n", db)

	if db == nil {
		logger.Fatalf("db is nil in NewConfigService\n")
	}

	refService := NewConfigReferenceService(db, rdb, cacheService)
	dagService := NewConfigDagService(db, rdb, cacheService, refService)
	// handleService := newConfigSettingHandleService(db, rdb, versionService)
	diffService := NewConfigDiffService(db, rdb, cacheService, dagService, refService)
	configContextService := NewConfigContextService(db, rdb, cacheService)

	configService := &ConfigService{
		logger: logger,
		db:     db,
		rdb:    rdb,

		dagService: dagService,
		// handleService:        handleService,
		diffService:          diffService,
		configContextService: configContextService,
	}

	configSchemaService := NewConfigSchemaService(db, rdb, configService)
	configService.configSchemaService = configSchemaService

	return configService
}

func (s *ConfigService) GetConfigContextService() *ConfigContextService {
	return s.configContextService
}

func (s *ConfigService) GetConfigDiffService() *ConfigDiffService {
	return s.diffService
}

func (s *ConfigService) GetConfigDagService() *ConfigDagService {
	return s.dagService
}

func (s *ConfigService) GetConfigSchemaService() *ConfigSchemaService {
	return s.configSchemaService
}

func (s *ConfigService) CreateNode(scope util.ScopeKind, accountId util.AccountId, userId util.UserId, nodeKind ConfigNodeKind, data *util.Data, prevNode *ConfigNode) (*ConfigNode, error) {
	return s.dagService.CreateNode(scope, accountId, userId, nodeKind, data, prevNode)
}

func (s *ConfigService) CommitNode(ctx context.Context, tx *gorm.DB, scope util.ScopeKind, accountId util.AccountId, userId util.UserId, node *ConfigNode) error {
	return s.dagService.CommitNode(ctx, tx, scope, accountId, userId, node)
}

type ConfigQueryValues map[string]interface{}

type ConfigModelFunc func() ConfigRecordModel

type ValueSettingMode string

const (
	ValueSettingModeReplace   ValueSettingMode = "replace_all"
	ValueSettingModeDeepMerge ValueSettingMode = "deep_merge"
)

// jsonb_build_object(
// 	'record_kind', record_kind,
// 	'record_id', record_id,
// 	'record_collection_key', record_collection_key,
// 	'record_item_key', record_item_key,
// 	'node_metadata', node_metadata
//   )

type ConfigListEntry struct {
	RecordKey           *util.ConfigRecordKey            `json:"id"`
	RecordKind          *ConfigRecordKind                `json:"record_kind"`
	RecordId            *util.ConfigRecordId             `json:"record_id"`
	RecordCollectionKey *util.ConfigCollectionKey        `json:"record_collection_key"`
	RecordItemKey       *util.ConfigItemKey              `json:"record_item_key"`
	NodeMetadata        *ConfigNodeMetadata              `json:"node_metadata"`
	RecordContents      *util.Data                       `json:"record_contents"`
	RecordHistory       []*ConfigDiffVersionHistoryEntry `json:"record_history"`
}

func (e *ConfigListEntry) String() string {
	return fmt.Sprintf("[ConfigListEntry: %s/%s %+v]", e.RecordCollectionKey, e.RecordItemKey, e.NodeMetadata)
}

func (s *ConfigService) ListConfigs(ctx context.Context, tx *gorm.DB, scope util.ScopeKind, accountId util.AccountId, userId util.UserId, recordQuery *ConfigRecordQuery) ([]*ConfigListEntry, error) {
	// return s.dagService.ListConfigs(ctx, tx, scope, accountId, userId, query)

	query := `SELECT * FROM get_record_list($1, $2, $3, $4, $5, $6)`

	var matchFilter *RecordMatchFilter = nil
	if recordQuery != nil {
		matchFilter = recordQuery.AsMatchFilter()
	}

	entries := []*ConfigListEntry{}
	err := util.RawGetJsonValue(ctx, s.db, tx, &entries, query, scope, accountId, userId, nil, nil, matchFilter)
	if err != nil {
		s.logger.Printf("ListConfigs: Error listing configs: %+v\n", err)
		return nil, fmt.Errorf("error listing configs: %w", err)
	}

	for _, entry := range entries {
		// s.logger.Printf("\n\nListConfigs: Entry: \n%s\n", util.ToJsonPretty(entry))

		// entryCommittedAt := entry.NodeMetadata.CommittedAt
		// entryHash := entry.NodeMetadata.VersionRef.ConfigVersionHash
		// parentHash := entry.NodeMetadata.ParentRef.ConfigVersionHash

		entryKey := ""
		if entry.RecordCollectionKey != nil {
			entryKey = string(*entry.RecordCollectionKey)
		}
		if entry.RecordItemKey != nil {
			entryKey += "/" + string(*entry.RecordItemKey)
		}
		entry.RecordKey = util.ConfigRecordKeyPtr(entryKey)

		if entry.RecordHistory != nil {
			// lastValues := &util.Data{}

			// for _, historyEntry := range entry.RecordHistory {
			// 	if historyEntry.RecordContents != nil {
			// 		s.logger.Printf("ListConfigs: Last values: %+v\n", lastValues)
			// 		diff, err := jsondiff.Compare(lastValues, historyEntry.RecordContents)
			// 		if err != nil {
			// 			s.logger.Printf("ListConfigs: Error comparing values: %+v\n", err)
			// 		} else {
			// 			historyEntry.RecordContentsPatch = &diff
			// 		}
			// 		lastValues = historyEntry.RecordContents
			// 	}
			// }

			AnnotateHistory(&entry.RecordHistory)
		}

		// s.logger.Printf("\n\nListConfigs: Entry [%8s] ^[%8s] %s @ %v\n", entryHash, parentHash, entryKey, entryCommittedAt)
		s.logger.Printf("\n\nListConfigs: Entry: \n%s\n", util.ToJsonPretty(entry))

		if entry.RecordHistory != nil {
			for _, historyEntry := range entry.RecordHistory {
				configVersionHash := historyEntry.NodeMetadata.VersionRef.ConfigVersionHash
				s.logger.Printf("\n[%8s] Record: %s\n", configVersionHash, historyEntry.RecordContents)
				s.logger.Printf("\n[%8s] Patch: %s\n", configVersionHash, historyEntry.RecordContentsPatch)
			}
		}
	}

	return entries, nil
}

// CREATE OR REPLACE FUNCTION get_latest_record(param_scope TEXT, param_account_id TEXT, param_user_id TEXT, param_from_version TEXT, param_to_version TEXT, param_record_kind TEXT, param_collection_key TEXT, param_item_key TEXT)
// RETURNS JSONB

// SELECT jsonb_agg(jsonb_build_object(
// 	'row_number', c.rn,
// 	'cur_hash', c.cur_hash,
// 	'parent_hash', c.parent_hash,
// 	'node_kind', c.node_kind,
// 	'node_metadata', c.node_metadata,
// 	'node_contents', c.node_contents,
// 	'record_metadata', c.record_metadata,
// 	'record_contents', c.node_contents->'record_contents',
// 	'record_match', c.record_match,
// 	'refs', c.refs
// ))

func (s *ConfigService) GetLatestRecord(ctx context.Context, tx *gorm.DB, scope util.ScopeKind, accountId util.AccountId, userId util.UserId, fromVersion *ConfigVersionRef, toVersion *ConfigVersionRef, recordQuery *ConfigRecordQuery) (*ConfigDiffVersion, error) {
	return s.diffService.GetLatestRecord(ctx, tx, scope, accountId, userId, fromVersion, toVersion, recordQuery)
}

func (s *ConfigService) mergeData(existingValues *util.Data, newValues *util.Data, mode ValueSettingMode) (*util.Data, error) {
	if existingValues == nil || mode == ValueSettingModeReplace {
		return newValues, nil
	}

	switch mode {
	case ValueSettingModeDeepMerge:
		oldValues := *existingValues
		mergedValues := oldValues
		if err := util.MergeDataInto(&mergedValues, newValues); err != nil {
			return nil, err
		}
		return &mergedValues, nil
	default:
		return nil, fmt.Errorf("invalid merge mode: %s", mode)
	}
}

// func (s *ConfigService) SetRecordValues(ctx context.Context, tx *gorm.DB, scope util.ScopeKind, accountId util.AccountId, userId util.UserId, kind ConfigRecordKind, recordQuery ConfigRecordQuery, recordMetadata ConfigRecordMetadata, mode ValueSettingMode, values *util.Data) (*ConfigNode, error) {
// 	if values == nil {
// 		s.logger.Printf("SetRecordValues: Values cannot be nil\n")
// 		return nil, fmt.Errorf("values cannot be nil")
// 	}

// 	if string(recordMetadata.CollectionKey) == "" {
// 		s.logger.Printf("SetRecordValues: CollectionKey is required\n")
// 		return nil, fmt.Errorf("collection key is required")
// 	}

// 	if kind == ConfigRecordKindDocument && recordMetadata.ItemKey == nil {
// 		s.logger.Printf("SetRecordValues: ItemKey is required for document records\n")
// 		return nil, fmt.Errorf("item key is required for document records")
// 	}

// 	if string(recordMetadata.RecordId) == "" {
// 		recordMetadata.RecordId = util.ConfigRecordId(util.NewUUID())
// 	}

// 	// TODO: Implement merging in the postgres function

// 	// existingRecordNode, err := s.GetRecordNode(ctx, tx, scope, accountId, userId, recordQuery)
// 	// if refErr, ok := err.(*ErrReferenceNotFound); ok {
// 	// 	s.logger.Printf("SetRecordValues: Reference not found: %+v\n", refErr)
// 	// } else if err != nil && err != gorm.ErrRecordNotFound {
// 	// 	s.logger.Printf("SetRecordValues: Error getting existing record (other than not found): %+v\n", err)
// 	// 	return nil, err
// 	// }

// 	// var existingData *util.Data = nil

// 	// if existingRecordNode != nil {
// 	// 	existingData = existingRecordNode.GetRecordContents()
// 	// }

// 	// TODO: Restore functionality
// 	var existingRecordNode *ConfigNode = nil
// 	var existingData *util.Data = nil

// 	s.logger.Printf("SetRecordValues: existingData: %+v\n", existingData)

// 	newData, err := s.mergeData(existingData, values, mode)
// 	if err != nil {
// 		s.logger.Printf("SetRecordValues: Error merging data: %+v\n", err)
// 		return nil, fmt.Errorf("error merging data: %w", err)
// 	}

// 	newRecord := &ConfigRecordObject{
// 		ConfigRecordKind: kind,
// 		RecordMetadata:   recordMetadata,
// 		Contents:         *newData,
// 	}

// 	recordData, err := util.ToDataMap(*newRecord)
// 	if err != nil {
// 		s.logger.Printf("SetRecordValues: Error converting record to data map: %+v\n", err)
// 		return nil, fmt.Errorf("error converting record to data map: %w", err)
// 	}

// 	newNode, err := s.dagService.CreateNode(scope, accountId, userId, ConfigNodeKindRecord, &recordData, existingRecordNode)
// 	if err != nil {
// 		s.logger.Printf("SetRecordValues: Error creating new node: %+v\n", err)
// 		return nil, fmt.Errorf("error creating new node: %w", err)
// 	}

// 	if err := s.dagService.CommitNode(ctx, tx, scope, accountId, userId, newNode); err != nil {
// 		s.logger.Printf("SetRecordValues: Error committing node: %+v\n", err)
// 		return nil, fmt.Errorf("error committing node: %w", err)
// 	}

// 	return newNode, nil
// }

// CREATE OR REPLACE FUNCTION set_record_values(param_scope TEXT, param_account_id TEXT, param_user_id TEXT, param_record_kind TEXT, param_collection_key TEXT, param_item_key TEXT, param_values JSONB, param_merge_mode TEXT DEFAULT 'deepmerge')
// RETURNS JSONB AS $func$

type SetRecordValuesResult struct {
	NodeMetadata ConfigNodeMetadata `json:"node_metadata"`
	NodeContents *util.Data         `json:"node_contents"`
}

func (s *ConfigService) SetRecordValues(ctx context.Context, tx *gorm.DB, scope util.ScopeKind, accountId util.AccountId, userId util.UserId, kind ConfigRecordKind, recordMetadata *ConfigRecordMetadata, mode ValueSettingMode, values *util.Data) (*ConfigNodeMetadata, error) {
	if values == nil {
		s.logger.Printf("SetRecordValues: Values cannot be nil\n")
		return nil, fmt.Errorf("values cannot be nil")
	}

	if string(recordMetadata.CollectionKey) == "" {
		s.logger.Printf("SetRecordValues: CollectionKey is required\n")
		return nil, fmt.Errorf("collection key is required")
	}

	if kind == ConfigRecordKindDocument && recordMetadata.ItemKey == nil {
		s.logger.Printf("SetRecordValues: ItemKey is required for document records\n")
		return nil, fmt.Errorf("item key is required for document records")
	}

	if string(recordMetadata.RecordId) == "" {
		recordMetadata.RecordId = util.ConfigRecordId(util.NewUUID())
	}

	query := `SELECT * FROM set_record_values($1, $2, $3, $4, $5, $6, $7, $8)`

	result := &SetRecordValuesResult{}

	err := util.RawGetJsonValue(ctx, s.db, tx, &result, query, scope, accountId, userId, kind, recordMetadata.CollectionKey, recordMetadata.ItemKey, values, mode)
	if err != nil {
		s.logger.Printf("SetRecordValues: Error setting record values: %+v\n", err)
		return nil, fmt.Errorf("error setting record values: %w", err)
	}

	return &result.NodeMetadata, nil
}

func (s *ConfigService) InsertRecord(ctx context.Context, tx *gorm.DB, scope util.ScopeKind, accountId util.AccountId, userId util.UserId, recordMetadata *ConfigRecordMetadata, recordObject interface{}) (*ConfigNodeMetadata, error) {

	if recordMetadata == nil {
		s.logger.Printf("InsertRecord: Record metadata cannot be nil\n")
		return nil, fmt.Errorf("record metadata cannot be nil")
	}

	if recordObject == nil {
		s.logger.Printf("InsertRecord: Record object cannot be nil\n")
		return nil, fmt.Errorf("record object cannot be nil")
	}

	dataMap, err := util.ToDataMap(recordObject)
	if err != nil {
		s.logger.Printf("InsertRecord: Error converting schema object to data map: %+v\n", err)
		return nil, fmt.Errorf("error converting schema object to data map: %w", err)
	}

	v := *recordMetadata

	k := ConfigRecordKindConfigSchema
	v.RecordKind = &k

	node, err := s.SetRecordValues(ctx, tx, scope, accountId, userId, ConfigRecordKindConfigSchema, &v, ValueSettingModeReplace, &dataMap)
	if err != nil {
		s.logger.Printf("InsertRecord: Error setting record values: %+v\n", err)
		return nil, fmt.Errorf("error inserting record: error inserting record values: %w", err)
	}

	return node, nil
}
