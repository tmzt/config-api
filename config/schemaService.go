package config

import (
	"context"
	"crypto/sha256"
	"fmt"

	redis "github.com/go-redis/redis/v8"

	"github.com/tmzt/config-api/util"

	"gorm.io/gorm"
)

type ConfigSchemaService struct {
	logger        util.SetRequestLogger
	db            *gorm.DB
	rdb           *redis.Client
	configService *ConfigService
}

func NewConfigSchemaService(db *gorm.DB, rdb *redis.Client, configService *ConfigService) *ConfigSchemaService {
	logger := util.NewLogger("ConfigSchemaService", 0)

	return &ConfigSchemaService{
		logger:        logger,
		db:            db,
		rdb:           rdb,
		configService: configService,
	}
}

func (s *ConfigSchemaService) getConfigVersion(query *ConfigRecordQuery) (*ConfigVersionRef, error) {
	accountId := query.AccountId
	userId := query.UserId
	scope := query.Scope
	versionHash := query.ConfigVersionHash

	if accountId == nil {
		return nil, fmt.Errorf("accountId is required")
	}
	if scope == nil {
		return nil, fmt.Errorf("scope is required")
	}

	// TODO: We should not be calling this if we don't have a hash
	if versionHash == nil {
		return nil, fmt.Errorf("configVersionHash is required")
	}

	version := &ConfigVersionRef{
		Scope:             *scope,
		AccountId:         *accountId,
		ConfigVersionHash: *versionHash,
	}
	if *scope == util.ScopeKindUser {
		if userId == nil {
			return nil, fmt.Errorf("userId is required for user scope")
		}
		version.UserId = userId
	}

	return version, nil
}

func (s *ConfigSchemaService) GetSchema(ctx context.Context, tx *gorm.DB, schemaQuery *ConfigRecordQuery) (*ConfigSchemaRecord, error) {
	s.logger.Printf("GetSchema called with query: %s\n", util.ToJsonPretty(schemaQuery))

	if schemaQuery.ConfigVersionHash == nil {
		return nil, fmt.Errorf("configVersionHash is required for SchemaService.GetSchema()")
	}

	version, err := s.getConfigVersion(schemaQuery)
	if err != nil {
		return nil, err
	}

	dagService := s.configService.GetConfigDagService()
	node, err := dagService.GetExistingNode(ctx, nil, version.Scope, version.AccountId, version.UserId, ConfigRefQueryFunc(*version))
	if err != nil {
		return nil, err
	}

	schema := &ConfigSchemaRecord{}
	if err := node.DecodeContents(schema); err != nil {
		return nil, err
	}

	return schema, nil
}

func (s *ConfigSchemaService) getSchemaQueryScope(schemaQuery *ConfigRecordQuery) (util.ScopeKind, *util.AccountId, *util.UserId, error) {
	if schemaQuery.Scope == nil {
		return util.ScopeKindInvalid, nil, nil, fmt.Errorf("scope is required")
	} else if schemaQuery.AccountId == nil {
		return util.ScopeKindInvalid, nil, nil, fmt.Errorf("accountId is required")
	} else if *schemaQuery.Scope == util.ScopeKindUser && schemaQuery.UserId == nil {
		return util.ScopeKindInvalid, nil, nil, fmt.Errorf("userId is required for user scope")
	}

	return *schemaQuery.Scope, schemaQuery.AccountId, schemaQuery.UserId, nil
}

func (s *ConfigSchemaService) GetLatestSchemaVersion(ctx context.Context, tx *gorm.DB, schemaQuery *ConfigRecordQuery) (*ConfigSchemaRecord, error) {
	s.logger.Printf("GetLatestSchemaVersion called with query: %s\n", util.ToJsonPretty(schemaQuery))

	scope, accountId, userId, err := s.getSchemaQueryScope(schemaQuery)
	if err != nil {
		return nil, err
	}

	diffVersion, err := s.configService.GetLatestRecord(ctx, nil, scope, *accountId, *userId, nil, nil, schemaQuery)
	if err != nil {
		return nil, err
	} else if diffVersion == nil {
		s.logger.Printf("GetLatestSchemaVersion: No schema found for query: %s\n", util.ToJsonPretty(schemaQuery))
		return nil, nil
	}

	schema := &ConfigSchemaRecord{}
	if err := diffVersion.DecodeRecordContents(schema); err != nil {
		s.logger.Printf("Error decoding record contents: %v\n", err)
		return nil, err
	}

	if diffVersion.ToVersion != nil {
		schema.SchemaHash = &diffVersion.ToVersion.ConfigVersionHash
	}

	return schema, nil
}

func (s *ConfigSchemaService) GetConfigSchemaVersions(ctx context.Context, tx *gorm.DB, schemaQuery *ConfigRecordQuery) (*ConfigSchemaVersions, error) {
	s.logger.Printf("GetConfigSchemaVersions called with query: %s\n", util.ToJsonPretty(schemaQuery))

	hasher := sha256.New()

	diffService := s.configService.GetConfigDiffService()
	diffParams := &ConfigDiffParams{
		ConfigRecordQuery: schemaQuery,
		IncludeObject:     true,
		IncludeRecord:     true,
		OnlyMatching:      true,
	}
	versions, err := diffService.GetVersionChain(ctx, nil, *schemaQuery.Scope, *schemaQuery.AccountId, *schemaQuery.UserId, diffParams)
	if err != nil {
		s.logger.Printf("getAllSchemas: Error getting record versions: %v\n", err)
		return nil, err
	}

	schemas := []*ConfigSchemaRecord{}

	if versions != nil {
		for _, version := range versions.Versions {
			// r.logger.Printf("getAllSchemas: version: %s\n", util.ToJsonPretty(version))
			// r.logger.Printf("getAllSchemas: node contents: %s\n", util.ToJsonPretty(version.NodeContents))
			s.logger.Printf("getAllSchemas: record contents: %s\n", util.ToJsonPretty(version.RecordContents))
			schema := &ConfigSchemaRecord{}
			if err := util.FromDataMap(version.RecordContents, schema); err != nil {
				s.logger.Printf("getAllSchemas: Error decoding schema: %v\n", err)
				continue
			}
			s.logger.Printf("getAllSchemas: versions.ToVersion: %s\n", util.ToJsonPretty(versions.ToVersion))
			s.logger.Printf("getAllSchemas: versions.FromVersion: %s\n", util.ToJsonPretty(versions.FromVersion))
			if v := version.ToVersion; v != nil {
				schema.SchemaHash = &v.ConfigVersionHash

				// Update the overall hash
				fmt.Fprintf(hasher, "%s", v.ConfigVersionHash)
			}
			s.logger.Printf("getAllSchemas: schema: %s\n", util.ToJsonPretty(schema))
			schemas = append(schemas, schema)
		}
	}

	listHash := fmt.Sprintf("%x", hasher.Sum(nil))

	res := &ConfigSchemaVersions{
		Versions: schemas,
		ListHash: listHash,
	}

	return res, nil
}
