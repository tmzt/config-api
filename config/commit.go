package config

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/tmzt/config-api/util"
)

type CommitKind string

const (
	// Add more later
	CommitKindConfigDataObject CommitKind = "config_data_object"
)

type CommitObject struct {
	CommitKind       CommitKind        `json:"commit_kind"`
	ConfigDataObject *ConfigDataObject `json:"data_object"`
}

func (o *CommitObject) Value() (driver.Value, error) {
	// Prevent GORM from treating the CDO as a relation
	return json.Marshal(o)
}

func (o *CommitObject) Scan(src interface{}) error {
	// Prevent GORM from treating the CDO as a relation
	return json.Unmarshal(src.([]byte), o)
}

type CommitORM struct {
	util.ImmutableEmbed `json:",inline"`
	CommitObject        `json:"commit_object"`
}
