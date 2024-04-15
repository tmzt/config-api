package config

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/tmzt/config-api/util"
)

type ConfigNodeORM struct {
	util.ImmutableEmbed `json:",inline"`
	ConfigNode
}

func (c ConfigNodeORM) TableName() string {
	return "config_nodes"
}

func (c *ConfigNodeORM) GetConfigNode() *ConfigNode {
	return &c.ConfigNode
}

type ConfigNodeMetadata struct {
	NodeKind    ConfigNodeKind    `json:"node_kind" gorm:"type:text;not null"`
	Scope       util.ScopeKind    `json:"scope" gorm:"type:text;not null"`
	AccountId   util.AccountId    `json:"account_id" gorm:"type:text;not null"`
	UserId      util.UserId       `json:"user_id" gorm:"type:text"`
	CreatedAt   time.Time         `json:"created_at" gorm:"type:timestamp with time zone;not null"`
	CreatedBy   util.UserId       `json:"created_by" gorm:"type:text"`
	CommittedAt *time.Time        `json:"committed_at" gorm:"index;type:timestamp with time zone"`
	CommittedBy *util.UserId      `json:"committed_by" gorm:"index;type:text"`
	VersionRef  ConfigVersionRef  `json:"version_ref" gorm:"type:jsonb;not null"`
	ParentRef   *ConfigVersionRef `json:"parent_ref" gorm:"type:jsonb"`
}

type ConfigNode struct {
	NodeMetadata ConfigNodeMetadata `json:"node_metadata" gorm:"column:node_metadata;type:jsonb;not null"`
	Contents     *util.Data         `json:"node_contents" gorm:"column:node_contents;type:jsonb"`
}

func NewConfigNode(kind ConfigNodeKind, scope util.ScopeKind, accountId util.AccountId, userId util.UserId, parentRef *ConfigVersionRef, contents *util.Data) (*ConfigNode, error) {
	logger := util.NewLogger("NewConfigNode", 0)

	ts := time.Now()

	versionRef := &ConfigVersionRef{
		ConfigVersionId:   util.ConfigVersionId(util.NewUUID()),
		ConfigVersionHash: util.EmptyHash,
		CreatedAt:         ts,
		CreatedBy:         userId,
	}

	node := &ConfigNode{
		NodeMetadata: ConfigNodeMetadata{
			NodeKind:    kind,
			Scope:       scope,
			AccountId:   accountId,
			UserId:      userId,
			CreatedAt:   ts,
			CommittedAt: nil,
			CommittedBy: nil,
			VersionRef:  *versionRef,
			ParentRef:   parentRef,
		},
		Contents: contents,
	}

	hash, err := node.ComputeHash()
	if err != nil {
		logger.Printf("Error computing hash for new node: %+v\n", err)
		return nil, fmt.Errorf("error computing hash for new node: %w", err)
	}

	node.NodeMetadata.VersionRef.ConfigVersionHash = *hash

	return node, nil
}

func (c *ConfigNode) ParseNode() (interface{}, error) {
	logger := util.NewLogger("ConfigNode.ParseNode", 0)

	switch c.NodeMetadata.NodeKind {
	case ConfigNodeKindEmpty:
		return nil, nil
	case ConfigNodeKindRecord:
		record := &ConfigRecordObject{}
		if err := c.DecodeContents(record); err != nil {
			logger.Printf("ParseNode: Error decoding existing node as record: %+v\n", err)
			return nil, err
		}
		return record, nil
	default:
		return nil, fmt.Errorf("unsupported node kind: %v", c.NodeMetadata.NodeKind)
	}
}

func (c *ConfigNode) GetConfigVersionRef() *ConfigVersionRef {
	return &c.NodeMetadata.VersionRef
}

func (c *ConfigNode) GetContents() *util.Data {
	return c.Contents
}

func (c *ConfigNode) SetContents(contents *util.Data) error {
	logger := util.NewLogger("ConfigNode.SetContents", 0)

	if c.NodeMetadata.CommittedAt != nil {
		return fmt.Errorf("cannot modify committed node")
	}

	c.Contents = contents

	hash, err := c.ComputeHash()
	if err != nil {
		logger.Printf("Error computing hash for modified node: %+v\n", err)
		return fmt.Errorf("error computing hash for modified node: %w", err)
	}

	c.NodeMetadata.VersionRef.ConfigVersionHash = *hash

	return nil
}

func (c *ConfigNode) AsRecord() *ConfigRecordObject {
	logger := util.NewLogger("ConfigNode.AsRecord", 0)

	if c.NodeMetadata.NodeKind != ConfigNodeKindRecord {
		logger.Printf("AsRecord: Node is not a record: %+v\n", c.NodeMetadata.NodeKind)
		return nil
	}

	record := &ConfigRecordObject{}
	if err := c.DecodeContents(record); err != nil {
		logger.Printf("AsRecord: Error decoding existing node as record: %+v\n", err)
		return nil
	}

	return record
}

func (c *ConfigNode) GetRecordContents() *util.Data {
	logger := util.NewLogger("ConfigNode.GetRecordContents", 0)

	record := c.AsRecord()

	if record == nil {
		logger.Printf("Node is not a record, cannot get record contents: %+v\n", c.NodeMetadata.NodeKind)
		return nil
	}

	return &record.Contents
}

func (c *ConfigNode) DecodeContents(dest interface{}) error {
	return util.FromDataMap(c.Contents, dest)
}

func (c *ConfigNode) EncodeContents(src interface{}) error {
	contents, err := util.ToDataMap(src)
	if err != nil {
		return err
	}
	c.Contents = &contents
	return nil
}

func (c *ConfigNode) GetNodeMetadata() *ConfigNodeMetadata {
	return &c.NodeMetadata
}

func (c *ConfigNode) GetVersionRef() *ConfigVersionRef {
	return &c.NodeMetadata.VersionRef
}

func (c *ConfigNode) GetParentRef() *ConfigVersionRef {
	return c.NodeMetadata.ParentRef
}

func (c *ConfigNode) ComputeHash() (*util.ConfigVersionHash, error) {

	// Copy the node structure
	node := *c

	node.NodeMetadata.VersionRef.ConfigVersionHash = util.EmptyHash

	hash, err := hashObject(node)
	if err != nil {
		return nil, err
	}
	versionHash := util.ConfigVersionHash(*hash)

	return &versionHash, nil
}

func (m *ConfigNodeORM) AddIndexes(ctx context.Context, tx *sql.Tx) error {
	logger := util.NewLogger("ConfigNodeORM.AddIndexes", 0)

	tableName := m.TableName()

	// Create the extension if it does not exist
	if _, err := tx.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS btree_gin"); err != nil {
		return err
	}

	addIndex := func(indexName string, unique bool, gin bool, cond *string, fields ...string) error {
		spec := strings.Join(fields, ", ")
		uniqueStr := ""
		if unique {
			uniqueStr = "UNIQUE "
		}
		ginStr := ""
		if gin {
			ginStr = " USING gin "
		}
		query := "CREATE " + uniqueStr + "INDEX IF NOT EXISTS " + indexName + " ON " + tableName + " " + ginStr + " (" + spec + ")"
		if cond != nil {
			query += "\n\tWHERE " + *cond
		}
		logger.Printf("Creating index (%s): \n%v\n", indexName, query)
		if _, err := tx.ExecContext(ctx, query); err != nil {
			logger.Printf("Error creating index (%s): %v\n", indexName, err)
			return err
		}
		return nil
	}

	combinedFields := []string{
		"scope",
		"account_id",
		"user_id",
		"(node_metadata->>'node_kind')",
		"(node_metadata->'version_ref'->>'config_version_id')",
		"(node_metadata->'version_ref'->>'config_version_hash')",
		"node_contents",
	}
	// combinedFieldsSpec := strings.Join(combinedFields, ", ")

	// Create a gin index on the combined fields
	if err := addIndex(tableName+"_combined_gin", false, true, nil, combinedFields...); err != nil {
		return err
	}

	// Add unique index on version_id and version_hash
	if err := addIndex(tableName+"_version_id", true, false, nil, "(node_metadata->'version_ref'->>'config_version_id')"); err != nil {
		return err
	}
	if err := addIndex(tableName+"_version_hash", true, false, nil, "(node_metadata->'version_ref'->>'config_version_hash')"); err != nil {
		return err
	}

	recordCond := "((node_metadata->>'node_kind') = 'record')"

	// Add unique index on record_id
	// conditional on it being a record node kind
	if err := addIndex(tableName+"_unique_record_id", true, false, &recordCond, "(node_contents->'record_metadata'->>'record_id')"); err != nil {
		return err
	}

	// This is wrong, the combination of collection/item will not be unique

	// uniqueRecordCollectionAndItem := []string{
	// 	"scope",
	// 	"account_id",
	// 	"user_id",
	// 	"(node_contents->'record_metadata'->>'collection_key')",
	// 	"(node_contents->'record_metadata'->>'item_key')",
	// }

	// if err := addIndex(tableName+"_unique_record_collection_item", true, false, uniqueRecordCollectionAndItem...); err != nil {
	// 	return err
	// }

	return nil
}

func (m *ConfigNodeORM) RemoveIndexes(ctx context.Context, tx *sql.Tx) error {
	logger := util.NewLogger("ConfigNodeORM.RemoveIndexes", 0)

	tableName := m.TableName()

	dropIndex := func(indexName string) error {
		query := "DROP INDEX IF EXISTS " + indexName
		logger.Printf("Dropping index (%s): \n%v\n", indexName, query)
		if _, err := tx.ExecContext(ctx, query); err != nil {
			logger.Printf("Error dropping index (%s): %v\n", indexName, err)
			return err
		}
		return nil
	}

	// Drop the combined index
	if err := dropIndex(tableName + "_combined_gin"); err != nil {
		return err
	}

	// Drop the version_id and version_hash indexes
	if err := dropIndex(tableName + "_version_id"); err != nil {
		return err
	}
	if err := dropIndex(tableName + "_version_hash"); err != nil {
		return err
	}

	// Drop the unique record_id index
	if err := dropIndex(tableName + "_unique_record_id"); err != nil {
		return err
	}

	// This is wrong, the combination of collection/item will not be unique

	// // Drop the unique record collection and item index
	// if err := dropIndex(tableName + "_unique_record_collection_item"); err != nil {
	// 	return err
	// }

	return nil
}

func (m *ConfigNodeORM) AddConstraints(ctx context.Context, tx *sql.Tx) error {
	logger := util.NewLogger("ConfigNodeORM.AddConstraints", 0)

	tableName := m.TableName()

	// Add a check constraint on parent_ref when node_kind is not ConfigNodeKindEmpty ('empty')
	checkParentRefFmt := `
		ALTER TABLE %s ADD CONSTRAINT check_parent_ref_node_kind_not_empty
		CHECK (node_metadata->>'node_kind' = 'empty' OR node_metadata->'parent_ref' IS NOT NULL);
	`

	checkParentRef := fmt.Sprintf(checkParentRefFmt, tableName)

	logger.Printf("Adding check constraint on parent_ref: \n%v\n", checkParentRef)

	if _, err := tx.ExecContext(ctx, checkParentRef); err != nil {
		logger.Printf("%s: Error adding check constraint on parent_ref: %v\n", tableName, err)
		return err
	}

	return nil
}

func (m *ConfigNodeORM) RemoveConstraints(ctx context.Context, tx *sql.Tx) error {
	logger := util.NewLogger("ConfigNodeORM.RemoveConstraints", 0)
	logger.Printf("Removing constraints from config_nodes\n")

	tableName := m.TableName()

	// Drop the check constraint on parent_ref
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT check_parent_ref_node_kind_not_empty", tableName)); err != nil {
		logger.Printf("%s: Error dropping check constraint on parent_ref: %v\n", tableName, err)
		return err
	}

	return nil
}
