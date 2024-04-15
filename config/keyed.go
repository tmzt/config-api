package config

import (
	"context"
	"database/sql"

	"github.com/tmzt/config-api/util"
	"gorm.io/gorm"
)

type ConfigKeyedRecordObject struct {
	ConfigRecordObject
}

type ConfigKeyedDataORM struct {
	ConfigKey util.ConfigKey `json:"config_key" gorm:"type:text;not null"`

	ConfigImmutableDataEmbed `json:",inline"`
}

func (c *ConfigKeyedDataORM) TableName() string {
	return "config_keyed_data"
}

func (c *ConfigKeyedDataORM) AddIndexes(ctx context.Context, tx *sql.Tx) error {
	return c.ConfigImmutableDataEmbed.AddDataIndexes(ctx, tx, c.TableName(), "config_key")
}

func (c *ConfigKeyedDataORM) RemoveIndexes(ctx context.Context, tx *sql.Tx) error {
	tableName := c.TableName()

	// Drop index on account_id, user_id, and config_key
	if _, err := tx.ExecContext(ctx, "DROP INDEX IF EXISTS "+tableName+"_resource_config_key_unique"); err != nil {
		return err
	}

	return c.ConfigImmutableDataEmbed.RemoveDataIndexes(ctx, tx, c.TableName())
}

func (c *ConfigKeyedDataORM) GetConfigDataObject() *ConfigDataObject {
	return &c.ConfigDataObject
}

func (c *ConfigKeyedDataORM) SetRecordConfigData(ctx context.Context, tx *gorm.DB, data util.Data, handle ConfigContextHandle) error {
	return handle.SetData(ctx, tx, &c.ConfigDataObject, data)
}
