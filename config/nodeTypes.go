package config

import (
	"fmt"

	"github.com/tmzt/config-api/util"
)

type ConfigNodeKind string

const (
	// Used as a placeholder for a branch with no commits
	ConfigNodeKindEmpty  ConfigNodeKind = "empty"
	ConfigNodeKindData   ConfigNodeKind = "data"
	ConfigNodeKindRecord ConfigNodeKind = "record"
	// ConfigNodeKindSchema            ConfigNodeKind = "schema"
	// ConfigNodeKindSchemaAssociation ConfigNodeKind = "schema_association"
)

type ConfigRecordNode struct {
	RecordMetadata ConfigRecordMetadata `json:"record_metadata" gorm:"column:record_metadata;type:jsonb;not null"`
	RecordContents util.Data            `json:"record_contents" gorm:"column:record_contents;type:jsonb"`
}

func ToRecordNode(data util.Data) (*ConfigRecordNode, error) {
	logger := util.NewLogger("ToRecordNode", 0)

	res := &ConfigRecordNode{}
	err := util.FromDataMap(data, res)
	if err != nil {
		logger.Printf("ToRecordNode: Error parsing ConfigRecordNode from DataMap: %v\n", err)
		return nil, err
	}
	return res, nil
}

func (n *ConfigRecordNode) ParseRecord() (*ConfigRecordKind, interface{}, error) {
	logger := util.NewLogger("ConfigRecordNode.Parse", 0)

	var res interface{}
	var err error

	kind := n.RecordMetadata.RecordKind
	if kind == nil {
		logger.Printf("ConfigRecordNode.Parse: RecordKind is nil\n")
		return nil, nil, fmt.Errorf("nil RecordKind in ConfigRecordNode")
	}

	switch *kind {
	case ConfigRecordKindData:
		// Returns raw *util.Data
		return kind, &n.RecordContents, nil
	case ConfigRecordKindKeyed:
		// Returns raw *util.Data
		return kind, &n.RecordContents, nil
	case ConfigRecordKindDocument:
		// Returns raw *util.Data
		return kind, &n.RecordContents, nil
	case ConfigRecordKindConfigSchema:
		// Returns *ConfigSchemaRecord
		res = &ConfigSchemaRecord{}
		err = util.FromDataMap(n.RecordContents, res)
		if err != nil {
			logger.Printf("ConfigRecordNode.Parse: Error parsing ConfigSchemaRecord from ConfigRecordNode: %v\n", err)
			return nil, nil, err
		}
		return kind, res, nil
	case ConfigRecordKindConfigSchemaAssociation:
		// Returns *ConfigSchemaAssociationRecord
		res = &ConfigSchemaAssociationRecord{}
		err = util.FromDataMap(n.RecordContents, res)
		if err != nil {
			logger.Printf("ConfigRecordNode.Parse: Error parsing ConfigSchemaAssociationRecord from ConfigRecordNode: %v\n", err)
			return nil, nil, err
		}
		return kind, res, nil
	default:
		logger.Printf("ConfigRecordNode.Parse: Unknown ConfigRecordKind: %v\n", *kind)
		return nil, nil, fmt.Errorf("unsupported ConfigRecordKind: %v", *kind)
	}
}

// type ConfigSchemaNode struct {
// 	SchemaHash     string    `json:"schema_hash" gorm:"column:schema_hash;type:varchar(64);not null"`
// 	SchemaName     string    `json:"schema_name" gorm:"column:schema_name;type:varchar(64);not null"`
// 	SchemaIdValue  *string   `json:"schema_id_value" gorm:"column:schema_id_value;type:varchar(64);not null"`
// 	SchemaContents util.Data `json:"schema_contents" gorm:"column:schema_contents;type:jsonb"`
// }

// type ConfigSchemaAssociationNode struct {
// 	SchemaHash    string            `json:"schema_hash" gorm:"column:schema_hash;type:varchar(64);not null"`
// 	SchemaIdValue string            `json:"schema_id_value" gorm:"column:schema_id_value;type:varchar(64);not null"`
// 	SchemaRef     *ConfigVersionRef `json:"schema_ref" gorm:"column:schema_ref;type:jsonb;not null"`
// }
