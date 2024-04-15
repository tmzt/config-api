package config

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/tmzt/config-api/util"
)

type ConfigImmutableDataEmbed struct {
	// These objects are immutable
	// if they need to be updated, a new record
	// is created pointing to a new config version

	AccountId util.AccountId `json:"account_id" gorm:"not null"`
	UserId    *util.UserId   `json:"user_id" gorm:""`
	CreatedAt time.Time      `json:"created_at"`
	CreatedBy *util.UserId   `json:"created_by" gorm:""`
	DeletedAt *time.Time     `json:"deleted_at"`

	// ConfigImmutableEmbedOld `json:",inline" gorm:"embedded"`
	// ConfigDataObject `json:",inline" gorm:"config_data_object;type:jsonb"`
	// ConfigDataObject `json:",inline" gorm:"config_data_object"`
	// ConfigDataObject `json:",inline" gorm:"-"`
	// ConfigDataObject `json:",inline" gorm:"embedded;type:jsonb"`
	// ConfigDataObject `json:",inline"`
	ConfigDataObjectEmbed `json:",inline"`
}

func (c *ConfigImmutableDataEmbed) AddDataIndexes(ctx context.Context, tx *sql.Tx, tableName string, fields ...string) error {
	// Create the extension if it does not exist
	if _, err := tx.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS btree_gin"); err != nil {
		return err
	}

	// scalarFields := append([]string{"account_id", "user_id"}, fields...)
	// scalarFieldsSpec := strings.Join(scalarFields, ", ")

	// // Create a unique index on the scalar fields
	// if _, err := tx.ExecContext(ctx, "CREATE UNIQUE INDEX IF NOT EXISTS "+tableName+"_resource_unique ON "+tableName+" ("+scalarFieldsSpec+")"); err != nil {
	// 	return err
	// }

	// TODO: Add gin index on the resource fields and
	// 		cdo->version_ref->config_version_id
	// 		cdo->version_ref->config_version_hash

	combinedFields := append([]string{"account_id", "user_id", "config_data_object"}, fields...)
	combinedFieldsSpec := strings.Join(combinedFields, ", ")

	// Create a gin index on the combined fields
	if _, err := tx.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS "+tableName+"_resource_config_data_object_gin ON "+tableName+" USING gin ("+combinedFieldsSpec+")"); err != nil {
		return err
	}

	// // Create a gin index on the whole object
	// if _, err := tx.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS "+tableName+"_config_data_object_gin ON "+tableName+" USING gin (config_data_object)"); err != nil {
	// 	return err
	// }

	// Add indexes here
	return nil
}

func (c *ConfigImmutableDataEmbed) RemoveDataIndexes(ctx context.Context, tx *sql.Tx, tableName string) error {
	// Remove the gin index on account_id, user_id, and the whole object
	if _, err := tx.ExecContext(ctx, "DROP INDEX IF EXISTS "+tableName+"_resource_config_data_object_gin"); err != nil {
		return err
	}

	// // Remove the gin index on the whole object
	// if _, err := tx.ExecContext(ctx, "DROP INDEX IF EXISTS "+tableName+"_config_data_object_gin"); err != nil {
	// 	return err
	// }

	// Add indexes to remove here
	return nil
}

func (c *ConfigImmutableDataEmbed) GetConfigDataObject() *ConfigDataObject {
	return &c.ConfigDataObject
}
