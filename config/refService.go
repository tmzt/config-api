package config

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/tmzt/config-api/util"
	"gorm.io/gorm"
)

type ConfigReferenceService struct {
	logger util.SetRequestLogger
	db     *gorm.DB
	rdb    *redis.Client

	cacheService *util.CacheService
}

func NewConfigReferenceService(db *gorm.DB, rdb *redis.Client, cacheService *util.CacheService) *ConfigReferenceService {
	logger := util.NewLogger("configReferenceService", 0)

	return &ConfigReferenceService{
		logger: logger,
		db:     db,
		rdb:    rdb,

		cacheService: cacheService,
	}
}

type configRefCache struct {
	Scope      util.ScopeKind      `json:"scope"`
	AccountId  util.AccountId      `json:"account_id"`
	UserId     util.UserId         `json:"user_id"`
	Kind       ConfigReferenceKind `json:"config_ref_kind"`
	VersionRef *ConfigVersionRef   `json:"version_ref"`
}

func (c configRefCache) CacheKey() string {
	return configRefCacheKey(c.Scope, c.AccountId, c.UserId, c.Kind)
}

func (c configRefCache) Ttl() time.Duration {
	return 1 * time.Hour
}

func configRefCacheKey(scope util.ScopeKind, accountId util.AccountId, userId util.UserId, kind ConfigReferenceKind) string {
	userVal := string(userId)
	if scope != util.ScopeKindUser {
		userVal = ""
	}
	return fmt.Sprintf("config_api:config_ref:scope=%s:account_id=%s:user_id=%s:config_ref_kind=%s", scope, accountId, userVal, kind)
}

func (s *ConfigReferenceService) GetConfigReference(ctx context.Context, tx *gorm.DB, scope util.ScopeKind, accountId util.AccountId, userId util.UserId, kind ConfigReferenceKind) (*configRefCache, error) {

	cacheQuery := configRefCache{
		Scope:     scope,
		AccountId: accountId,
		UserId:    userId,
		Kind:      kind,
	}

	var res *configRefCache

	cached, ok, err := s.cacheService.GetCachedObject(ctx, &cacheQuery)
	if err != nil {
		s.logger.Printf("Error getting cached config reference: %s\n", err)
	} else if ok {
		v, ok := cached.(configRefCache)
		if !ok {
			s.logger.Printf("Error getting cached config reference: expected configRefCache, got %T\n", cached)
		} else {
			res = &v
		}
	}

	return res, nil
}

// func (s *ConfigReferenceService) GetConfigReferencedVersion(ctx context.Context, tx *gorm.DB, accountId util.AccountId, userId *util.UserId, kind ConfigReferenceKind) (*ConfigVersionRef, *ConfigVersionRef, bool, error) {
// 	whereClause := "v.account_id = ? AND config_reference_kind = ?"
// 	whereConds := []interface{}{accountId, kind}
// 	if userId != nil {
// 		whereClause += " AND v.user_id = ?"
// 		whereConds = append(whereConds, *userId)
// 	}

// 	res := &configVersionRefCache{
// 		current: ConfigVersionRef{},
// 		parent:  &ConfigVersionRef{},
// 	}

// 	found := true

// 	query := fmt.Sprintf(`
// 		SELECT v.config_version_id, v.hash, v.parent_id, v.parent_hash
// 		FROM config_versions v
// 		JOIN config_refs r ON (
// 			v.config_version_id = r.version_ref->>'config_version_id'
// 			AND
// 			v.hash = r.version_ref->>'config_version_hash'
// 		)
// 		WHERE %s
// 		GROUP BY v.config_version_id, v.hash, v.parent_id, v.parent_hash, v.account_id, v.user_id
// 		LIMIT 1
// 	`, whereClause)

// 	err := util.WithTransaction(s.db, tx, func(tx *gorm.DB) error {
// 		rows, err := tx.Raw(query, whereConds...).Rows()
// 		if err != nil {
// 			return err
// 		}
// 		defer rows.Close()

// 		var currentId string
// 		var currentHash string
// 		var parentId *string
// 		var parentHash *string

// 		if rows.Next() {
// 			if err := rows.Scan(&currentId, &currentHash, &parentId, &parentHash); err != nil {
// 				s.logger.Printf("Error scanning config reference version (account_id: %s user_id: %s kind: %s): %s\n", err, accountId, userId, kind)
// 				return err
// 			}

// 			if currentId != "" {
// 				res.current.ConfigVersionId = util.ConfigVersionId(currentId)
// 			}
// 			if currentHash != "" {
// 				res.current.ConfigVersionHash = util.ConfigVersionHash(currentHash)
// 			}
// 			if parentId != nil {
// 				res.parent.ConfigVersionId = util.ConfigVersionId(*parentId)
// 			}
// 			if parentHash != nil {
// 				res.parent.ConfigVersionHash = util.ConfigVersionHash(*parentHash)
// 			}

// 			return nil
// 		}

// 		found = false

// 		return nil
// 		// return fmt.Errorf("no config reference version found for account_id: %s user_id: %s kind: %s", accountId, userId, kind)
// 	})
// 	if err != nil {
// 		s.logger.Printf("Error getting config reference version (account_id: %s user_id: %s kind: %s): %s\n", accountId, userId, kind, err)
// 		return nil, fmt.Errorf("error getting config reference version: %s", err)
// 	} else if !found {
// 		return nil, sql.ErrNoRows
// 	}
// 	return res, nil
// }

type ConfigReferenceResult struct {
	CurrentRef *ConfigVersionRef `json:"current_ref"`
	ParentRef  *ConfigVersionRef `json:"parent_ref"`
}

const recordQueryFmt = `
SELECT
	r.version_ref->>'config_version_hash' current_hash,
	n.node_metadata->'parent_ref'->>'config_version_hash' parent_hash
FROM config_refs r
JOIN config_nodes n ON (
	(r.version_ref->>'config_version_hash' = n.node_metadata->'version_ref'->>'config_version_hash')
	AND (r.scope = n.scope AND r.account_id = n.account_id AND %s)
)
WHERE r.scope = ? AND r.account_id = ? AND %s AND r.config_reference_kind = ?
`

func (s *ConfigReferenceService) createRecordQuery(scope util.ScopeKind, accountId util.AccountId, userId util.UserId, kind ConfigReferenceKind) string {
	userJoin := "r.user_id = n.user_id"
	userWhere := "r.user_id = ?"
	if scope != util.ScopeKindUser {
		userJoin = "r.user_id IS NULL AND n.user_id IS NULL"
		userWhere = "r.user_id IS NULL"
	}

	return fmt.Sprintf(recordQueryFmt, userJoin, userWhere)
}

func (s *ConfigReferenceService) GetRecord(ctx context.Context, tx *gorm.DB, scope util.ScopeKind, accountId util.AccountId, userId util.UserId, kind ConfigReferenceKind) (*ConfigReferenceResult, error) {
	res := &ConfigReferenceResult{}

	query := s.createRecordQuery(scope, accountId, userId, kind)
	params := []interface{}{scope, accountId, kind}
	if scope == util.ScopeKindUser {
		params = []interface{}{scope, accountId, userId, kind}
	}

	out := struct {
		CurrentHash string `json:"current_hash"`
		ParentHash  string `json:"parent_hash"`
	}{}

	err := util.WithTransaction(s.db, tx, func(tx *gorm.DB) error {
		return tx.Raw(query, params...).First(&out).Error
	})
	if err == gorm.ErrRecordNotFound || err == sql.ErrNoRows {
		s.logger.Printf("No config reference found (account_id %s, user_id, %s kind: %s)\n", accountId, userId, kind)
		return nil, nil
	} else if err != nil {
		s.logger.Printf("Error getting config reference (other than not found) (account_id %s, user_id, %s kind: %s) from database: %s\n", accountId, userId, kind, err)
		return nil, err
	} else {
		s.logger.Printf("\n\nGot config reference (account_id %s, user_id, %s kind: %s) from database\n\n", accountId, userId, kind)

		res.CurrentRef = &ConfigVersionRef{
			ConfigVersionHash: util.ConfigVersionHash(out.CurrentHash),
		}
		if out.ParentHash != "" {
			res.ParentRef = &ConfigVersionRef{
				ConfigVersionHash: util.ConfigVersionHash(out.ParentHash),
			}
		}
	}
	return res, nil
}

func (s *ConfigReferenceService) upsertRecord(ctx context.Context, tx *gorm.DB, scope util.ScopeKind, accountId util.AccountId, userId util.UserId, kind ConfigReferenceKind, ref *ConfigVersionRef) error {
	s.logger.Printf("Upserting config reference (scope %s, account_id %s, user_id, %s kind: %s) -> %s\n", scope, accountId, userId, kind, ref.ConfigVersionHash)

	ts := time.Now()

	if string(scope) == "" {
		return fmt.Errorf("invalid scope")
	}

	if string(accountId) == "" {
		return fmt.Errorf("invalid account id")
	}

	if scope == util.ScopeKindUser && string(userId) == "" {
		return fmt.Errorf("invalid user id")
	}

	if string(kind) == "" {
		return fmt.Errorf("invalid kind")
	}

	attrs := &ConfigReferenceORM{
		Scope:               scope,
		AccountId:           accountId,
		ConfigReferenceKind: kind,
	}

	if scope == util.ScopeKindUser {
		attrs.UserId = util.UserIdPtr(string(userId))
	}

	if string(ref.Scope) == "" {
		ref.Scope = scope
	}

	if string(ref.AccountId) == "" {
		ref.AccountId = accountId
	}

	if scope == util.ScopeKindUser && (ref.UserId == nil || string(*ref.UserId) == "") {
		ref.UserId = &userId
	}

	if ref.CreatedAt.IsZero() {
		ref.CreatedAt = ts
	}

	assign := &ConfigReferenceORM{
		VersionRef: ref,
	}

	err := util.WithTransaction(s.db, tx, func(tx *gorm.DB) error {
		return tx.Where(attrs).
			Attrs(attrs).
			Assign(assign).
			FirstOrCreate(&ConfigReferenceORM{}).Error
	})
	if err != nil {
		s.logger.Printf("Error upserting config reference (account_id %s, user_id, %s kind: %s) in database: %s\n", accountId, userId, kind, err)
		return err
	}
	return nil
}

func (s *ConfigReferenceService) SetConfigReference(ctx context.Context, tx *gorm.DB, scope util.ScopeKind, accountId util.AccountId, userId util.UserId, kind ConfigReferenceKind, ref *ConfigVersionRef) error {

	cacheObj := configRefCache{
		AccountId:  accountId,
		Scope:      scope,
		UserId:     userId,
		Kind:       kind,
		VersionRef: ref,
	}

	if err := s.cacheService.SaveCacheObject(ctx, &cacheObj); err != nil {
		s.logger.Printf("Error saving cached config reference: %s\n", err)
	}

	if err := s.upsertRecord(ctx, tx, scope, accountId, userId, kind, ref); err != nil {
		s.logger.Printf("Error upserting config reference: %s\n", err)
	}

	return nil
}
