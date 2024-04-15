package config

import (
	"context"
	"database/sql"

	"github.com/tmzt/config-api/util"
	"gorm.io/gorm"
)

type ConfigDocumentORM struct {
	ConfigDocumentKey util.ConfigDocumentKey `json:"config_document_key" gorm:"type:text;not null"`
	ConfigDocumentId  util.ConfigDocumentId  `json:"config_document_id" gorm:"type:text;not null"`

	// ConfigImmutableDataEmbed `json:",inline" gorm:"embedded"`
	// ConfigImmutableDataEmbed `json:",inline" gorm:"-"`
	// ConfigImmutableDataEmbed `json:",inline" ` //gorm:"-"`
	ConfigImmutableDataEmbed `json:",inline"`
}

func (c *ConfigDocumentORM) TableName() string {
	return "config_documents"
}

func (c *ConfigDocumentORM) AddIndexes(ctx context.Context, tx *sql.Tx) error {
	return c.ConfigImmutableDataEmbed.AddDataIndexes(ctx, tx, c.TableName(), "config_document_key", "config_document_id")
}

func (c *ConfigDocumentORM) RemoveIndexes(ctx context.Context, tx *sql.Tx) error {
	return c.ConfigImmutableDataEmbed.RemoveDataIndexes(ctx, tx, c.TableName())
}

func (c *ConfigDocumentORM) GetConfigDataObject() *ConfigDataObject {
	return &c.ConfigDataObject
}

func (c *ConfigDocumentORM) SetRecordConfigData(ctx context.Context, tx *gorm.DB, data util.Data, handle ConfigContextHandle) error {
	return handle.SetData(ctx, tx, &c.ConfigDataObject, data)
}
