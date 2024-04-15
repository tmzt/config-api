package config

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tmzt/config-api/util"
)

type ConfigVersionRef struct {
	Scope             util.ScopeKind         `json:"scope" gorm:"index;not null"`
	AccountId         util.AccountId         `json:"account_id" gorm:"index;not null"`
	UserId            *util.UserId           `json:"user_id" gorm:"index;null"`
	ConfigVersionId   util.ConfigVersionId   `json:"config_version_id" gorm:"index;not null"`
	ConfigVersionHash util.ConfigVersionHash `json:"config_version_hash" gorm:"index;not null"`
	CreatedAt         time.Time              `json:"created_at"`
	CreatedBy         util.UserId            `json:"created_by" gorm:"index;null"`
	CommittedAt       *time.Time             `json:"committed_at"`
	CommittedBy       *util.UserId           `json:"committed_by" gorm:"index;null"`
	Note              *string                `json:"note"`
}

func (v ConfigVersionRef) String() string {
	return fmt.Sprintf("ConfigVersionRef{AccountId: %s, UserId: %s, Id: %s, Hash: %s, CreatedAt: %s, CreatedBy: %s}",
		v.AccountId, util.UserIdPtrStr(v.UserId), v.ConfigVersionId, v.ConfigVersionHash, v.CreatedAt, v.CreatedBy)
}

func (v ConfigVersionRef) Value() (driver.Value, error) {
	return json.Marshal(v)
}

func (v *ConfigVersionRef) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		return json.Unmarshal(src, v)
	case string:
		return json.Unmarshal([]byte(src), v)
	}
	return fmt.Errorf("unsupported type: %T", src)
}
