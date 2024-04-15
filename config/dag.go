package config

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/tmzt/config-api/util"
	"gorm.io/gorm"
)

type ConfigDagService struct {
	logger       util.SetRequestLogger
	db           *gorm.DB
	rdb          *redis.Client
	cacheService *util.CacheService
	refService   *ConfigReferenceService
}

func NewConfigDagService(db *gorm.DB, rdb *redis.Client, cacheService *util.CacheService, refService *ConfigReferenceService) *ConfigDagService {
	logger := util.NewLogger("ConfigDagService", 0)

	return &ConfigDagService{
		logger:       logger,
		db:           db,
		rdb:          rdb,
		cacheService: cacheService,
		refService:   refService,
	}
}

type scanResult struct {
	Scope     util.ScopeKind `json:"scope" gorm:"column:scope;type:text;not null"`
	AccountId util.AccountId `json:"account_id" gorm:"column:account_id;type:text;not null"`
	UserId    *util.UserId   `json:"user_id" gorm:"column:user_id;type:text"`
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at;type:timestamp with time zone;not null"`
	CreatedBy util.UserId    `json:"created_by" gorm:"column:created_by;type:text;not null"`

	NodeMetadata []byte  `json:"node_metadata" gorm:"column:node_metadata;type:jsonb;not null"`
	NodeContents *[]byte `json:"node_contents" gorm:"column:node_contents;type:jsonb"`
}

func (r *scanResult) ScanInto(node *ConfigNodeORM) error {
	node.ImmutableEmbed.Scope = r.Scope
	node.ImmutableEmbed.AccountId = r.AccountId
	node.ImmutableEmbed.UserId = r.UserId
	node.ImmutableEmbed.CreatedAt = r.CreatedAt
	node.ImmutableEmbed.CreatedBy = r.CreatedBy

	// if err := util.FromDataMap(r.NodeMetadata, &node.NodeMetadata); err != nil {
	// 	return err
	// }
	// if err := util.FromDataMap(r.NodeContents, &node.Contents); err != nil {
	// 	return err
	// }

	if err := json.Unmarshal(r.NodeMetadata, &node.NodeMetadata); err != nil {
		return err
	}

	if r.NodeContents != nil && len(*r.NodeContents) > 0 {
		if err := json.Unmarshal(*r.NodeContents, &node.Contents); err != nil {
			return err
		}
	} else {
		node.Contents = nil
	}

	return nil
}

func (s *ConfigDagService) GetExistingNode(ctx context.Context, tx *gorm.DB, scope util.ScopeKind, accountId util.AccountId, userId *util.UserId, queryFuncs ...ConfigQueryFunc) (*ConfigNodeORM, error) {

	if scope == util.ScopeKindUser && userId == nil {
		return nil, fmt.Errorf("userId is required for user scope")
	}

	scanResult := &scanResult{}

	query := `
		SELECT
			scope, account_id, user_id, created_at, created_by,
			node_metadata, node_contents
		FROM
			config_nodes n
		WHERE
			n.scope = $1 AND n.account_id = $2 AND (
				CASE WHEN $1 = 'user' THEN n.user_id = $3 ELSE n.user_id IS NULL END
			)
	`

	found := false
	err := util.WithTransaction(s.db, tx, func(tx *gorm.DB) error {
		gormQuerier := tx.Raw(query, scope, accountId, userId)
		adapter := &ConfigGormQuerier{querier: gormQuerier}

		for _, queryFunc := range queryFuncs {
			adapter = queryFunc(adapter).(*ConfigGormQuerier)
		}

		err := gormQuerier.First(scanResult).Error
		if err == gorm.ErrRecordNotFound || err == sql.ErrNoRows {
			s.logger.Printf("ConfigDagService.GetExistingNode: node not found\n")
			return nil
		} else if err != nil {
			s.logger.Printf("ConfigDagService.GetExistingNode: error getting existing node (other than not found): %v\n", err)
			return err
		}

		found = true
		return nil
	})
	s.logger.Printf("ConfigDagService.GetExistingNode: error: %v\n", err)
	s.logger.Printf("ConfigDagService.GetExistingNode: found: %v\n", found)
	if err != nil {
		return nil, err
	} else if !found {
		s.logger.Printf("ConfigDagService.GetExistingNode: node not found, returning nil, nil\n")
		return nil, nil
	}

	s.logger.Printf("ConfigDagService.GetExistingNode: got scanResult: %+v\n", scanResult)

	node := &ConfigNodeORM{}
	if err := scanResult.ScanInto(node); err != nil {
		s.logger.Printf("ConfigDagService.GetExistingNode: error scanning scanResult into node: %v\n", err)
		return nil, err
	}

	j, _ := json.MarshalIndent(node, "", "  ")
	s.logger.Printf("ConfigDagService.GetExistingNode: got node: %s\n", j)

	return node, nil
}

// func (s *ConfigDagService) getVersionByRefKind(ctx context.Context, tx *gorm.DB, scope util.ScopeKind, accountId util.AccountId, userId util.UserId, refKind ConfigReferenceKind) (*ConfigVersionRef, error) {
// 	ref, err := s.refService.GetRecord(ctx, tx, scope, accountId, userId, refKind)
// 	if err != nil {
// 		s.logger.Printf("ConfigDagService.getVersionByRefKind[%s]: error getting record: %v\n", refKind, err)
// 		return nil, err
// 	} else if ref == nil {
// 		s.logger.Printf("ConfigDagService.getVersionByRefKind[%s]: record not found\n", refKind)
// 		return nil, nil
// 	}

// 	return ref.CurrentRef, nil
// }

// // Returns the version ref of the root, creates root node if it doesn't exist
// func (s *ConfigDagService) getOrCreateRootReference(ctx context.Context, tx *gorm.DB, scope util.ScopeKind, accountId util.AccountId, userId util.UserId) (*ConfigVersionRef, error) {
// 	s.logger.Printf("ConfigDagService.getOrCreateRootReference: getting or creating head reference\n")

// 	// First get the root ref
// 	rootRef, err := s.getVersionByRefKind(ctx, tx, scope, accountId, userId, ConfigReferenceKindRoot)
// 	if err != nil {
// 		s.logger.Printf("ConfigDagService.getOrCreateRootReference: error getting root node: %v\n", err)
// 		return nil, err
// 	}

// 	// Check if the root node exists
// 	if rootRef != nil {
// 		rootNode, err := s.GetExistingNode(ctx, tx, scope, accountId, userId, ConfigRefQueryFunc(*rootRef))
// 		if err != nil {
// 			s.logger.Printf("ConfigDagService.getOrCreateRootReference: error getting existing root node: %v\n", err)
// 			return nil, err
// 		}

// 		// TODO: Handle this case better, but it should not occur unless
// 		// the database is an inconsistent state
// 		if rootNode == nil {
// 			s.logger.Printf("ConfigDagService.getOrCreateRootReference: ROOT NODE NOT FOUND\n")
// 			return nil, fmt.Errorf("root node not found")
// 		}
// 	}

// 	// If the root ref doesn't exist, we create a new empty node
// 	if rootRef == nil {
// 		s.logger.Printf("ConfigDagService.getOrCreateRootReference: creating root node\n")
// 		rootNode, err := s.CreateNode(scope, accountId, userId, ConfigNodeKindEmpty, nil, nil)
// 		if err != nil {
// 			s.logger.Printf("ConfigDagService.getOrCreateRootReference: error creating root node: %v\n", err)
// 			return nil, err
// 		}

// 		if err := s.CommitNode(ctx, tx, scope, accountId, userId, rootNode); err != nil {
// 			s.logger.Printf("ConfigDagService.getOrCreateRootReference: error committing root node: %v\n", err)
// 			return nil, err
// 		}

// 		rootRef = &rootNode.NodeMetadata.VersionRef
// 	}

// 	// Next we get the head ref
// 	headRef, err := s.getVersionByRefKind(ctx, tx, scope, accountId, userId, ConfigReferenceKindHead)
// 	if err != nil {
// 		s.logger.Printf("ConfigDagService.getOrCreateRootReference: error getting head node: %v\n", err)
// 		return nil, err
// 	}

// 	// If we don't have head but we have root, we set head to root
// 	if headRef == nil {
// 		s.logger.Printf("ConfigDagService.getOrCreateRootReference: setting head to root\n")
// 		if err := s.refService.SetConfigReference(ctx, tx, scope, accountId, userId, ConfigReferenceKindHead, rootRef); err != nil {
// 			s.logger.Printf("ConfigDagService.getOrCreateRootReference: error setting head reference: %v\n", err)
// 			return nil, err
// 		}
// 	}

// 	// Return the root version ref
// 	return rootRef, nil
// }

// func (s *ConfigDagService) GetOrCreateReferenceByKind(ctx context.Context, tx *gorm.DB, scope util.ScopeKind, accountId util.AccountId, userId util.UserId, refKind ConfigReferenceKind) (*ConfigVersionRef, error) {
// 	s.logger.Printf("ConfigDagService.GetOrCreateReferenceByKind[%s]: getting or creating reference\n", refKind)

// 	// First, get or create the root reference
// 	rootRef, err := s.getOrCreateRootReference(ctx, tx, scope, accountId, userId)
// 	if err != nil {
// 		s.logger.Printf("ConfigDagService.GetOrCreateReferenceByKind[%s]: error getting or creating root reference: %v\n", refKind, err)
// 		return nil, err
// 	} else if rootRef == nil {
// 		s.logger.Printf("ConfigDagService.GetOrCreateReferenceByKind[%s]: root reference is nil, this should not happen\n", refKind)
// 		return nil, fmt.Errorf("root reference is nil")
// 	}

// 	// If the ref kind is root, we return the root ref
// 	if refKind == ConfigReferenceKindRoot {
// 		return rootRef, nil
// 	}

// 	// First get the ref
// 	ref, err := s.getVersionByRefKind(ctx, tx, scope, accountId, userId, refKind)
// 	if err != nil {
// 		s.logger.Printf("ConfigDagService.GetOrCreateReferenceByKind[%s]: error getting reference: %v\n", refKind, err)
// 		return nil, err
// 	}

// 	// Check if the referred to node exists
// 	if ref != nil {
// 		node, err := s.GetExistingNode(ctx, tx, scope, accountId, userId, ConfigRefQueryFunc(*ref))
// 		if err != nil {
// 			s.logger.Printf("ConfigDagService.GetOrCreateReferenceByKind[%s]: error getting existing node: %v\n", refKind, err)
// 			return nil, err
// 		} else if node == nil {
// 			s.logger.Printf("ConfigDagService.GetOrCreateReferenceByKind[%s]: referred node not found\n", refKind)
// 			return nil, fmt.Errorf("referred node not found")
// 		}

// 		return ref, nil
// 	}

// 	// If this is the head ref, we will set it to the root reference
// 	if refKind == ConfigReferenceKindHead {
// 		s.logger.Printf("ConfigDagService.GetOrCreateReferenceByKind[%s]: setting head to root\n", refKind)
// 		if err := s.refService.SetConfigReference(ctx, tx, scope, accountId, userId, ConfigReferenceKindHead, rootRef); err != nil {
// 			s.logger.Printf("ConfigDagService.GetOrCreateReferenceByKind[%s]: error setting head reference: %v\n", refKind, err)
// 			return nil, err
// 		}

// 		return rootRef, nil
// 	}

// 	// Otherwise, return nil, nil
// 	return nil, nil
// }

// func (s *ConfigDagService) CreateNode(scope util.ScopeKind, accountId util.AccountId, userId util.UserId, nodeKind ConfigNodeKind, data *util.Data, prevNode *ConfigNode) (*ConfigNode, error) {
// 	if nodeKind != ConfigNodeKindEmpty && data == nil {
// 		return nil, fmt.Errorf("invalid data, cannot be nil, except for empty node kind")
// 	}

// 	s.logger.Printf("ConfigDagService.CreateNode: creating node: prevNode(ptr): %p (is nil: %v)\n", prevNode, prevNode == nil)

// 	j, _ := json.MarshalIndent(data, "", "  ")
// 	s.logger.Printf("ConfigDagService.CreateNode: creating node: data: %s\n", j)

// 	var parentRef *ConfigVersionRef = nil
// 	if prevNode != nil {
// 		parentRef = &prevNode.NodeMetadata.VersionRef
// 	}

// 	node, err := NewConfigNode(nodeKind, scope, accountId, userId, parentRef, data)
// 	if err != nil {
// 		s.logger.Printf("ConfigDagService.CreateNode: error creating node: \n", err)
// 		return nil, err
// 	}

// 	return node, nil
// }

type RefMap map[ConfigReferenceKind]ConfigReferenceORM

// CREATE OR REPLACE FUNCTION get_or_init_repo(param_scope TEXT, param_account_id TEXT, param_user_id TEXT)

func (s *ConfigDagService) GetReferences(ctx context.Context, tx *gorm.DB, scope util.ScopeKind, accountId util.AccountId, userId util.UserId) (RefMap, error) {
	results := []ConfigReferenceORM{}

	query := `SELECT * FROM get_or_init_repo($1, $2, $3);`

	err := util.WithTransaction(s.db, tx, func(tx *gorm.DB) error {

		rows, err := tx.Raw(query, scope, accountId, userId).Rows()
		if err != nil {
			s.logger.Printf("ConfigDagService.GetReferences: error calling get_or_init_repo(): %v\n", err)
			return err
		}

		defer rows.Close()

		if err := tx.ScanRows(rows, results); err != nil {
			s.logger.Printf("ConfigDagService.GetReferences: error scanning rows: %v\n", err)
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	refs := make(map[ConfigReferenceKind]ConfigReferenceORM, len(results))

	for _, result := range results {
		refs[result.ConfigReferenceKind] = result
	}

	return refs, nil
}

func (s *ConfigDagService) CreateNode(scope util.ScopeKind, accountId util.AccountId, userId util.UserId, nodeKind ConfigNodeKind, data *util.Data, prevNode *ConfigNode) (*ConfigNode, error) {
	if nodeKind != ConfigNodeKindEmpty && data == nil {
		return nil, fmt.Errorf("invalid data, cannot be nil, except for empty node kind")
	}

	s.logger.Printf("ConfigDagService.CreateNode: creating node: prevNode(ptr): %p (is nil: %v)\n", prevNode, prevNode == nil)

	j, _ := json.MarshalIndent(data, "", "  ")
	s.logger.Printf("ConfigDagService.CreateNode: creating node: data: %s\n", j)

	var parentRef *ConfigVersionRef = nil
	if prevNode != nil {
		parentRef = &prevNode.NodeMetadata.VersionRef
	}

	node, err := NewConfigNode(nodeKind, scope, accountId, userId, parentRef, data)
	if err != nil {
		s.logger.Printf("ConfigDagService.CreateNode: error creating node: \n", err)
		return nil, err
	}

	return node, nil
}

func (s *ConfigDagService) CommitNode(ctx context.Context, tx *gorm.DB, scope util.ScopeKind, accountId util.AccountId, userId util.UserId, node *ConfigNode) error {
	s.logger.Printf("ConfigDagService.CommitNode: preparing to commit node: %p\n", node)
	// TODO: clean up this method

	// Validations

	nodeKind := node.NodeMetadata.NodeKind
	if nodeKind == ConfigNodeKindEmpty {
		if node.Contents != nil {
			return fmt.Errorf("invalid contents, must be nil for empty node kind")
		}

		if node.NodeMetadata.ParentRef != nil {
			return fmt.Errorf("invalid parent ref, must be nil for empty node kind")
		}
	} else if node.Contents == nil {
		return fmt.Errorf("invalid contents, cannot be nil, except for empty node kind")
	}

	// CREATE OR REPLACE FUNCTION insert_dag_node(param_scope TEXT, param_account_id TEXT, param_user_id TEXT, param_parent_hash TEXT, param_node_metadata JSONB, param_contents JSONB)

	query := `SELECT node_metadata FROM insert_dag_node($1, $2, $3, $4, $5, $6);`

	err := util.WithTransaction(s.db, tx, func(tx *gorm.DB) error {

		// refs, err := s.GetReferences(ctx, tx, scope, accountId, userId)
		// if err != nil {
		// 	s.logger.Printf("ConfigDagService.CommitNode: error getting versions: %v\n", err)
		// 	return err
		// }

		var updateRefs []string = nil

		// Print the call to insert_dag_node
		s.logger.Printf("ConfigDagService.CommitNode: calling \n\n\nSELECT * FROM insert_dag_node('%s', '%s', '%s', '%s', '%s', '%s');\n\n\n", scope, accountId, userId, util.ToJson(node.NodeMetadata), util.ToJson(node.Contents), util.ToJson(updateRefs))

		var nodeMetadata ConfigNodeMetadata
		if err := util.RawGetJsonValue(ctx, s.db, tx, &nodeMetadata, query, scope, accountId, userId, node.NodeMetadata, node.Contents, updateRefs); err != nil {
			s.logger.Printf("ConfigDagService.CommitNode: error calling insert_dag_node(): %v\n", err)
			return err
		}

		s.logger.Printf("ConfigDagService.CommitNode: got node metadata: %s\n", util.ToJson(nodeMetadata))

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
