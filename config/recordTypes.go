package config

import "github.com/tmzt/config-api/util"

type ConfigRecordKind string

const (
	ConfigRecordKindData                    ConfigRecordKind = "data"
	ConfigRecordKindKeyed                   ConfigRecordKind = "keyed"
	ConfigRecordKindDocument                ConfigRecordKind = "document"
	ConfigRecordKindConfigSchema            ConfigRecordKind = "config_schema"
	ConfigRecordKindConfigSchemaAssociation ConfigRecordKind = "config_schema_association"
)

func ConfigRecordKindAsPtr(kind ConfigRecordKind) *ConfigRecordKind {
	return &kind
}

type ConfigDataRecord util.Data
type ConfigKeyedRecord util.Data
type ConfigDocumentRecord util.Data

type ConfigSchemaRecord struct {
	SchemaHash     *util.ConfigVersionHash   `json:"schema_hash" gorm:"column:schema_hash;type:varchar(64);not null"`
	SchemaName     *util.ConfigSchemaName    `json:"schema_name" gorm:"column:schema_name;type:varchar(64);not null"`
	SchemaIdValue  *util.ConfigSchemaIdValue `json:"schema_id_value" gorm:"column:schema_id_value;type:varchar(64);not null"`
	SchemaContents util.ConfigSchemaContents `json:"schema_contents" gorm:"column:schema_contents;type:jsonb"`
}

type ConfigSchemaAssociationRecord struct {
	SchemaHash    *util.ConfigSchemaHash    `json:"schema_hash" gorm:"column:schema_hash;type:varchar(64);not null"`
	SchemaIdValue *util.ConfigSchemaIdValue `json:"schema_id_value" gorm:"column:schema_id_value;type:varchar(64);not null"`
	SchemaRef     *ConfigVersionRef         `json:"schema_ref" gorm:"column:schema_ref;type:jsonb;not null"`
}
