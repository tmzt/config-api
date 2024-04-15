package config

import "github.com/tmzt/config-api/util"

type ConfigSchemaObject struct {
	SchemaName     *util.ConfigSchemaName     `json:"schema_name"`
	SchemaIdValue  *util.ConfigSchemaIdValue  `json:"schema_id_value"`
	SchemaContents *util.ConfigSchemaContents `json:"schema_contents"`
}
