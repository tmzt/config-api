package config

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/wI2L/jsondiff"

	"github.com/tmzt/config-api/util"
)

type ConfigDiffService struct {
	logger       util.SetRequestLogger
	db           *gorm.DB
	rdb          *redis.Client
	cacheService *util.CacheService
	dagService   *ConfigDagService
	refService   *ConfigReferenceService
	// versionService *configVersionService

	dmp *diffmatchpatch.DiffMatchPatch
}

func NewConfigDiffService(db *gorm.DB, rdb *redis.Client, cacheService *util.CacheService, dagService *ConfigDagService, refService *ConfigReferenceService) *ConfigDiffService {
	logger := util.NewLogger("ConfigDiffService", 0)

	logger.Printf("NewConfigDiffService: db: %v\n", db)
	logger.Printf("NewConfigDiffService: rdb: %v\n", rdb)
	logger.Printf("NewConfigDiffService: cacheService: %v\n", cacheService)

	if db == nil {
		logger.Fatalf("db is nil in NewConfigDiffService\n")
	}
	if cacheService == nil {
		logger.Fatalf("cacheService is nil\n")
	}

	logger.Printf("NewConfigDiffService: db: %v\n", db)

	dmp := diffmatchpatch.New()

	return &ConfigDiffService{
		logger:       logger,
		db:           db,
		rdb:          rdb,
		cacheService: cacheService,
		dagService:   dagService,
		refService:   refService,
		dmp:          dmp,
	}
}

type ConfigDiffSpec struct {
	// TODO: Support other refs/tags

	Version *ConfigVersionRef `json:"version"`
	// Kind    *ConfigReferenceKind `json:"kind"`
}

type ConfigDiffParams struct {
	From *ConfigDiffSpec
	To   *ConfigDiffSpec

	IncludeObject              bool
	IncludePreviousObject      bool
	IncludeNodeContentsPatch   bool
	IncludeRecordMetadataPatch bool
	IncludeRecordContentsPatch bool

	IncludeRecord bool

	OnlyMatching bool
	// MatchCondFuncs []ConfigQueryFunc

	// RecordMatchFilter *RecordMatchFilter
	ConfigRecordQuery *ConfigRecordQuery

	OnlyLatest bool
}

type ConfigDiffVersions struct {
	FromVersion *ConfigVersionRef          `json:"from_version"`
	ToVersion   *ConfigVersionRef          `json:"to_version"`
	Versions    []ConfigDiffVersion        `json:"versions"`
	ListHash    *util.ConfigSchemaListHash `json:"list_hash"`
}

type ConfigDiffVersionHistoryEntry struct {
	RecordContents       *util.Data                `json:"record_contents"`
	RecordCollectionKey  *util.ConfigCollectionKey `json:"record_collection_key"`
	RecordItemKey        *util.ConfigItemKey       `json:"record_item_key"`
	ConfigRecordMetadata *ConfigRecordMetadata     `json:"config_record_metadata"`
	NodeMetadata         *ConfigNodeMetadata       `json:"node_metadata"`
	// ConfigVersionHash       *util.ConfigVersionHash   `json:"config_version_hash"`
	RecordContentsPatch     *jsondiff.Patch `json:"record_diff"`
	RecordContentsTextPatch string          `json:"record_diff_text"`
}

type ConfigDiffVersion struct {
	FromVersion *ConfigVersionRef `json:"from_version"`
	ToVersion   *ConfigVersionRef `json:"to_version"`
	Match       bool              `json:"match"`

	NodeContents   *util.Data            `json:"object,omitempty"`
	RecordMetadata *ConfigRecordMetadata `json:"record_metadata,omitempty"`
	RecordContents *util.Data            `json:"record_contents,omitempty"`
	RecordHistory  []*ConfigDiffVersionHistoryEntry

	NodeContentsPatch   *jsondiff.Patch `json:"diff"`
	RecordMetadataPatch *jsondiff.Patch `json:"record_metadata_diff"`
	RecordContentsPatch *jsondiff.Patch `json:"record_diff"`

	// Temporary
	PrevObject *util.Data `json:"prev_object,omitempty"`
}

func (v *ConfigDiffVersion) DecodeRecordContents(dest interface{}) error {
	if v.RecordContents == nil {
		return fmt.Errorf("diff version has nil record contents")
	}
	return util.FromDataMap(v.RecordContents, dest)
}

// CREATE TYPE record_match_filter AS (
// 	scope TEXT,
// 	account_id TEXT,
// 	user_id TEXT,
// 	record_id TEXT,
// 	record_collection_key TEXT,
// 	record_item_key TEXT
// );

// CREATE TYPE version_chain_entry AS (
// 	cur_hash TEXT,
// 	parent_hash TEXT,
// 	node_kind TEXT,
// 	node_metadata JSONB,
// 	node_contents JSONB,
// 	record_metadata JSONB,
// 	record_contents JSONB,
// 	record_match BOOL
// );

// type DiffChainRecordHistoryEntry struct {
// 	RecordContents      *util.Data                `json:"record_contents"`
// 	RecordCollectionKey *util.ConfigCollectionKey `json:"record_collection_key"`
// 	RecordItemKey       *util.ConfigItemKey       `json:"record_item_key"`
// 	ConfigVersionHash   *util.ConfigVersionHash   `json:"config_version_hash"`
// }

type DiffVersionChainEntry struct {
	CurrentHash    *util.ConfigVersionHash          `json:"cur_hash"`
	ParentHash     *util.ConfigVersionHash          `json:"parent_hash"`
	NodeKind       *ConfigNodeKind                  `json:"node_kind"`
	NodeMetadata   *ConfigNodeMetadata              `json:"node_metadata"`
	NodeContents   *util.Data                       `json:"node_contents"`
	RecordMetadata *ConfigRecordMetadata            `json:"record_metadata"`
	RecordContents *util.Data                       `json:"record_contents"`
	RecordHistory  []*ConfigDiffVersionHistoryEntry `json:"record_history"`
	RecordMatch    bool                             `json:"record_match"`
}

func (e *DiffVersionChainEntry) Value() (driver.Value, error) {
	// return util.ToJson(e), nil
	return json.Marshal(e)
}

func (e *DiffVersionChainEntry) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), e)
}

type RecordMatchFilter struct {
	Scope               *util.ScopeKind           `json:"scope"`
	AccountId           *util.AccountId           `json:"account_id"`
	UserId              *util.UserId              `json:"user_id"`
	RecordKind          *ConfigRecordKind         `json:"record_kind"`
	RecordId            *util.ConfigRecordId      `json:"record_id"`
	RecordCollectionKey *util.ConfigCollectionKey `json:"record_collection_key"`
	RecordItemKey       *util.ConfigItemKey       `json:"record_item_key"`
	OnlyMatching        bool                      `json:"only_matching"`
}

// TODO: Support tags and other refs, not just versions or head..root
// TODO: Support matching conditions

// CREATE OR REPLACE FUNCTION get_version_chain(param_scope TEXT, param_account_id TEXT, param_user_id TEXT, param_from_version TEXT, param_to_version TEXT)
// RETURNS TABLE (cur_hash TEXT, parent_hash TEXT, node_kind TEXT, node_metadata JSONB, node_contents JSONB)

func (s *ConfigDiffService) GetVersionChain(ctx context.Context, tx *gorm.DB, scope util.ScopeKind, accountId util.AccountId, userId util.UserId, params *ConfigDiffParams) (*ConfigDiffVersions, error) {

	res := &ConfigDiffVersions{}

	var fromVersion *ConfigVersionRef = nil
	var toVersion *ConfigVersionRef = nil

	var fromHash *string = nil
	var toHash *string = nil

	matchFilter := &RecordMatchFilter{}

	// TODO: Support other refs/tags
	if params != nil {
		if params.From != nil && params.From.Version != nil {
			fromVersion = params.From.Version
			fromHash = util.StrPtr(string(params.From.Version.ConfigVersionHash))
			if res.FromVersion == nil {
				res.FromVersion = fromVersion
			}
		}
		if params.To != nil && params.To.Version != nil {
			toVersion = params.To.Version
			toHash = util.StrPtr(string(params.To.Version.ConfigVersionHash))
			res.ToVersion = toVersion
		}

		if params.ConfigRecordQuery != nil {
			// matchFilter = params.ConfigRecordQuery.AsMatchFilter()
			params.ConfigRecordQuery.PopulateMatchFilter(matchFilter)
		}

		if params.OnlyMatching {
			matchFilter.OnlyMatching = true
		}
	}

	// query := `SELECT jsonb_array_elements(get_version_chain($1, $2, $3, $4, $5, $6))`
	query := `SELECT get_version_chain($1, $2, $3, $4, $5, $6)`

	entries := []ConfigDiffVersion{}

	rawEntries := [][]byte{}

	err := util.WithTransaction(s.db, tx, func(tx *gorm.DB) error {

		s.logger.Printf("Query: %s\n", util.FormatDebugQuery(query, scope, accountId, userId, fromHash, toHash, matchFilter))

		s.logger.Printf("To version: %s\n", toVersion)
		s.logger.Printf("From version: %s\n", fromVersion)

		rs := tx.Raw(query, scope, accountId, userId, fromHash, toHash, matchFilter)
		if err := rs.Error; err != nil {
			s.logger.Printf("Error getting version chain: %s\n", err)
			return fmt.Errorf("error getting version chain: %w", err)
		}

		if err := rs.First(&rawEntries).Error; err != nil {
			s.logger.Printf("Error scanning version chain: %s\n", err)
			return fmt.Errorf("error scanning version chain: %w", err)
		}

		s.logger.Printf("Raw entries: %s\n", string(rawEntries[0]))

		return nil
	})
	if err != nil {
		s.logger.Printf("Error getting version chain: %s\n", err)
		return nil, err
	}

	if len(rawEntries) == 0 {
		s.logger.Printf("No entries found\n")
		return nil, nil
	}

	chainEntries := []DiffVersionChainEntry{}

	// s.logger.Printf("Raw entries: %s\n", rawEntries[0])

	if err := json.Unmarshal(rawEntries[0], &chainEntries); err != nil {
		s.logger.Printf("Error unmarshalling entries: %s\n", err)
		return nil, fmt.Errorf("error unmarshalling entries: %w", err)
	}

	// s.logger.Printf("Chain entries: %s\n", util.ToJsonPretty(chainEntries))

	numEntries := len(chainEntries)

	for i := 0; i < numEntries; i++ {
		chainEntry := chainEntries[i]

		if params.OnlyMatching && !chainEntry.RecordMatch {
			continue
		}

		entry := ConfigDiffVersion{}

		if i > 0 {
			entry.FromVersion = &ConfigVersionRef{
				ConfigVersionHash: *chainEntries[i-1].CurrentHash,
			}
		}

		entry.ToVersion = &ConfigVersionRef{
			ConfigVersionHash: *chainEntry.CurrentHash,
		}
		s.logger.Printf("Entry.ToVersion: %+v\n", entry.ToVersion)

		if chainEntry.NodeContents != nil {
			if params.IncludeObject {
				entry.NodeContents = chainEntry.NodeContents
			}

			if params.IncludeNodeContentsPatch && i < numEntries-1 {
				patch, err := jsondiff.Compare(chainEntries[i+1].NodeContents, chainEntry.NodeContents)
				if err != nil {
					s.logger.Printf("Error comparing objects: %s\n", err)
				} else {
					entry.NodeContentsPatch = &patch
				}
			}

			if params.IncludePreviousObject {
				entry.PrevObject = chainEntries[i+1].NodeContents
			}

			// s.logger.Printf("Include record: %v\n", params.IncludeRecord)

			if params.IncludeRecord {
				entry.RecordMetadata = chainEntry.RecordMetadata
				entry.RecordContents = chainEntry.RecordContents
			}

			// s.logger.Printf("Record contents (entry): %s\n", util.ToJsonPretty(entry.RecordContents))

			if params.IncludeRecordMetadataPatch && i < numEntries-1 {
				patch, err := jsondiff.Compare(chainEntries[i+1].RecordMetadata, chainEntry.RecordMetadata)
				if err != nil {
					s.logger.Printf("Error comparing record metadata: %s\n", err)
				} else {
					entry.RecordMetadataPatch = &patch
				}
			}

			if params.IncludeRecordContentsPatch && chainEntry.RecordHistory != nil {
				entry.RecordHistory = chainEntry.RecordHistory
				s.logger.Printf("\nGetVersionChain: Record history: %s\n", util.ToJsonPretty(entry.RecordHistory))
				s.logger.Printf("\nGetVersionChain: Calling AnnotateHistory\n")
				AnnotateHistory(&entry.RecordHistory)
			}

			entry.Match = chainEntry.RecordMatch

			entries = append(entries, entry)

			if params.OnlyLatest {
				break
			}
		}
	}

	s.logger.Printf("Entries: %+v\n", entries)

	res.Versions = entries

	// if fromVersion != nil {
	// 	res.FromVersion = fromVersion
	// }

	// if toVersion != nil {
	// 	res.ToVersion = toVersion
	// }

	return res, nil

}

// -- get_latest_record_with_match_filter()
// CREATE OR REPLACE FUNCTION get_latest_record_with_match_filter(param_scope TEXT, param_account_id TEXT, param_user_id TEXT, param_from_version TEXT, param_to_version TEXT, param_record_match_filter JSONB)
// RETURNS JSONB

func (s *ConfigDiffService) GetLatestRecord(ctx context.Context, tx *gorm.DB, scope util.ScopeKind, accountId util.AccountId, userId util.UserId, fromVersion *ConfigVersionRef, toVersion *ConfigVersionRef, recordQuery *ConfigRecordQuery) (*ConfigDiffVersion, error) {
	// return nil, fmt.Errorf("not implemented")

	var matchFilter *RecordMatchFilter
	if recordQuery != nil {
		matchFilter = recordQuery.AsMatchFilter()
	}

	var fromHash *string = nil
	if fromVersion != nil {
		fromHash = util.StrPtr(string(fromVersion.ConfigVersionHash))
	}
	var toHash *string = nil
	if toVersion != nil {
		toHash = util.StrPtr(string(toVersion.ConfigVersionHash))
	}

	query := `SELECT * FROM get_latest_record_with_match_filter($1, $2, $3, $4, $5, $6)`

	res := &DiffVersionChainEntry{}

	err := util.RawGetJsonValue(ctx, s.db, tx, &res, query, scope, accountId, userId, fromHash, toHash, matchFilter)
	if err != nil {
		s.logger.Printf("Error getting latest record: %s\n", err)
		return nil, fmt.Errorf("error getting latest record: %w", err)
	}

	if res == nil {
		return nil, nil
	}

	s.logger.Printf("res: %+v\n", res)
	if res.RecordMetadata != nil {
		s.logger.Printf("Record metadata: %+v\n", res.RecordMetadata)
	}

	version := &ConfigDiffVersion{
		RecordMetadata: res.RecordMetadata,
		RecordContents: res.RecordContents,
		RecordHistory:  res.RecordHistory,
	}

	s.logger.Printf("\nGetLatestRecord: Record history: %s\n", util.ToJsonPretty(version.RecordHistory))
	s.logger.Printf("\nGetLatestRecord: Calling AnnotateHistory\n")
	AnnotateHistory(&version.RecordHistory)

	if res.NodeMetadata != nil {
		version.ToVersion = &ConfigVersionRef{
			ConfigVersionHash: res.NodeMetadata.VersionRef.ConfigVersionHash,
		}
	}

	return version, nil
}
