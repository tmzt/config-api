package config

import (
	"context"
	"database/sql"
	"strings"

	"github.com/tmzt/config-api/util"
)

type ConfigRecordORM struct {
	util.ImmutableEmbed `json:",inline"`
	ConfigRecordObject  `json:"config_record_object"`
}

func (c *ConfigRecordORM) TableName() string {
	return "config_records"
}

func (c *ConfigRecordORM) AddIndexes(ctx context.Context, tx *sql.Tx) error {
	logger := util.NewLogger("ConfigRecordORM.AddIndexes", 0)

	tableName := c.TableName()

	// Create the extension if it does not exist
	if _, err := tx.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS btree_gin"); err != nil {
		return err
	}

	combinedFields := []string{
		"scope",
		"account_id",
		"user_id",
		// "record_metadata->>'record_id'",
		// "record_metadata->>'collection_key'",
		// "record_metadata->>'item_key'",
		// "config_metadata->'version_ref'->>'config_version_id'",
		// "config_metadata->'version_ref'->>'config_version_hash'",
		// "record_id",
		// "collection_id",
		"record_metadata",
		"config_metadata",
	}
	combinedFieldsSpec := strings.Join(combinedFields, ", ")

	// Create a gin index on the combined fields
	if _, err := tx.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS "+tableName+"_record_metadata_gin ON "+tableName+" USING gin ("+combinedFieldsSpec+")"); err != nil {
		logger.Printf("Error creating combined fields gin index: %v\n", err)
		return err
	}

	scalarFields := []string{
		"scope",
		"account_id",
		"user_id",
		"(record_metadata->>'record_id')",
		"(record_metadata->>'collection_key')",
		"(record_metadata->>'item_key')",
		"(config_metadata->'version_ref'->>'config_version_id')",
		"(config_metadata->'version_ref'->>'config_version_hash')",
	}
	scalarFieldsSpec := strings.Join(scalarFields, ", ")

	scalarQuery := "CREATE INDEX IF NOT EXISTS " + tableName + "_record_metadata_btree ON " + tableName + " (" + scalarFieldsSpec + ")"
	logger.Printf("Creating scalar fields btree index: \n%v\n", scalarQuery)

	// Create a btree index on the scalar fields
	if _, err := tx.ExecContext(ctx, scalarQuery); err != nil {
		logger.Printf("Error creating scalar fields btree index: %v\n", err)
		return err
	}

	addUniqueIndex := func(name string, fields []string) error {
		fieldsSpec := strings.Join(fields, ", ")
		indexName := tableName + "_" + name
		query := "CREATE UNIQUE INDEX IF NOT EXISTS " + indexName + " ON " + tableName + " (" + fieldsSpec + ")"
		logger.Printf("Creating unique index (%s): \n%v\n", indexName, query)

		if _, err := tx.ExecContext(ctx, query); err != nil {
			logger.Printf("Error creating unique index (%s): %v\n", indexName, err)
			return err
		}

		return nil
	}

	uniqueScalarRecordFields := []string{
		"(record_metadata->>'record_id')",
		"(record_metadata->>'collection_key')",
		"(record_metadata->>'item_key')",
	}

	if err := addUniqueIndex("record_metadata_unique", uniqueScalarRecordFields); err != nil {
		return err
	}

	uniqueScalarNodeIdFields := []string{
		"(config_metadata->'version_ref'->>'config_version_id')",
		"(config_metadata->'version_ref'->>'config_version_hash')",
	}

	if err := addUniqueIndex("config_metadata_node_id_unique", uniqueScalarNodeIdFields); err != nil {
		return err
	}

	return nil
}

func (c *ConfigRecordORM) RemoveIndexes(ctx context.Context, tx *sql.Tx) error {
	logger := util.NewLogger("ConfigRecordORM.RemoveIndexes", 0)

	tableName := c.TableName()

	dropIndex := func(name string) error {
		indexName := tableName + "_" + name
		query := "DROP INDEX IF EXISTS " + indexName
		logger.Printf("Dropping index (%s): \n%v\n", indexName, query)
		if _, err := tx.ExecContext(ctx, query); err != nil {
			logger.Printf("Error dropping index (%s): %v\n", indexName, err)
			return err
		}
		return nil
	}

	// Drop the gin index on the combined fields
	if err := dropIndex("record_metadata_gin"); err != nil {
		return err
	}

	// if _, err := tx.ExecContext(ctx, "DROP INDEX IF EXISTS "+tableName+"_record_metadata_gin"); err != nil {
	// 	logger.Printf("Error dropping combined fields gin index: %v\n", err)
	// 	return err
	// }

	// Drop the btree index on the scalar fields
	if err := dropIndex("record_metadata_btree"); err != nil {
		return err
	}

	// Drop the unique index on the record metadata fields
	if err := dropIndex("record_metadata_unique"); err != nil {
		return err
	}

	// Drop the unique index on the config metadata node id fields
	if err := dropIndex("config_metadata_node_id_unique"); err != nil {
		return err
	}

	// if _, err := tx.ExecContext(ctx, "DROP INDEX IF EXISTS "+tableName+"_record_metadata_btree"); err != nil {
	// 	logger.Printf("Error dropping scalar fields btree index: %v\n", err)
	// 	return err
	// }

	return nil
}

type ConfigRecordMetadata struct {
	RecordId      util.ConfigRecordId      `json:"record_id"`
	CollectionKey util.ConfigCollectionKey `json:"record_collection_key"`
	ItemKey       *util.ConfigItemKey      `json:"record_item_key"`
	RecordKind    *ConfigRecordKind        `json:"record_kind"`
}

func (m *ConfigRecordMetadata) AsRecordQuery() *ConfigRecordQuery {
	return &ConfigRecordQuery{
		RecordId:      &m.RecordId,
		CollectionKey: &m.CollectionKey,
		ItemKey:       m.ItemKey,
	}
}

type ConfigRecordQuery struct {
	Scope             *util.ScopeKind           `json:"scope_kind"`
	AccountId         *util.AccountId           `json:"account_id"`
	UserId            *util.UserId              `json:"user_id"`
	RecordKind        *ConfigRecordKind         `json:"record_kind"`
	RecordId          *util.ConfigRecordId      `json:"record_id"`
	CollectionKey     *util.ConfigCollectionKey `json:"collection_key"`
	ItemKey           *util.ConfigItemKey       `json:"item_key"`
	ConfigVersionHash *util.ConfigVersionHash   `json:"config_version_hash"`
}

func (q *ConfigRecordQuery) AsMetadata() *ConfigRecordMetadata {
	logger := util.NewLogger("ConfigRecordQuery.AsMetadata", 0)
	if q.CollectionKey == nil {
		logger.Printf("ConfigRecordQuery.AsMetadata: CollectionKey is nil\n")
		return nil
	}

	return &ConfigRecordMetadata{
		CollectionKey: *q.CollectionKey,
		ItemKey:       q.ItemKey,
	}
}

func (q *ConfigRecordQuery) AsMatchFilter() *RecordMatchFilter {
	// TODO: Support record_kind
	return &RecordMatchFilter{
		Scope:               q.Scope,
		AccountId:           q.AccountId,
		UserId:              q.UserId,
		RecordId:            q.RecordId,
		RecordCollectionKey: q.CollectionKey,
		RecordItemKey:       q.ItemKey,
	}
}

func (q *ConfigRecordQuery) PopulateMatchFilter(filter *RecordMatchFilter) {
	if q.Scope != nil {
		filter.Scope = q.Scope
	}
	if q.AccountId != nil {
		filter.AccountId = q.AccountId
	}
	if q.UserId != nil {
		filter.UserId = q.UserId
	}
	if q.RecordKind != nil {
		filter.RecordKind = q.RecordKind
	}
	if q.RecordId != nil {
		filter.RecordId = q.RecordId
	}
	if q.CollectionKey != nil {
		filter.RecordCollectionKey = q.CollectionKey
	}
	if q.ItemKey != nil {
		filter.RecordItemKey = q.ItemKey
	}
}

type ConfigRecordObject struct {
	ConfigRecordKind ConfigRecordKind `json:"record_kind"`

	RecordMetadata ConfigRecordMetadata `json:"record_metadata" gorm:"column:record_metadata;type:jsonb;not null"`
	Contents       util.Data            `json:"record_contents" gorm:"column:record_contents;type:jsonb"`
}

func (c *ConfigRecordObject) AsDataMap() (*util.Data, bool) {
	return &c.Contents, true
}

func (c *ConfigRecordObject) DecodeContents(dest interface{}) error {
	return util.FromDataMap(c.Contents, dest)
}

func (c *ConfigRecordObject) EncodeContents(src interface{}) error {
	contents, err := util.ToDataMap(src)
	if err != nil {
		return err
	}
	c.Contents = contents
	return nil
}

type ConfigVersions struct {
	Versions []*ConfigRecordObject `json:"versions"`
	ListHash string                `json:"list_hash"`
}

type ConfigSchemaVersions struct {
	Versions []*ConfigSchemaRecord `json:"versions"`
	ListHash string                `json:"list_hash"`
}
