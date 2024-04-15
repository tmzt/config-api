package config

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/tmzt/config-api/util"
)

type ConfigRecordModel interface {
	// Gorm methods
	TableName() string

	GetConfigDataObject() *ConfigDataObject
	// SetRecordConfigData(ctx context.Context, data interface{}, handle ConfigSettingHandle) (ConfigRecordModel, error)
	// SetRecordConfigData(ctx context.Context, tx *gorm.DB, data util.Data, handle ConfigContextHandle) (ConfigRecordModel, error)
}

type ConfigORM struct {
	Id        string         `json:"id" gorm:"primaryKey"`
	AccountId util.AccountId `json:"account_id" gorm:"index;not null"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt *time.Time     `json:"deleted_at"`
	CreatedBy *util.UserId   `json:"created_by" gorm:"index"`
}

func (c *ConfigORM) TableName() string {
	return "configs"
}

type ConfigDagNodeKind string

const (
	ConfigDagNodeKindRoot         ConfigDagNodeKind = "root"
	ConfigDagNodeKindData         ConfigDagNodeKind = "data"
	ConfigDagNodeKindRecordChange ConfigDagNodeKind = "record_change"
)

type ConfigDagNode interface {
	GetConfigDagNode() *ConfigDagNodeEmbed
}

type ConfigDagNodeEmbed struct {
	AccountId       util.AccountId       `json:"account_id" gorm:"index;not null"`
	UserId          *util.UserId         `json:"user_id" gorm:"index;null"`
	ConfigVersionId util.ConfigVersionId `json:"config_version_id" gorm:"id;primaryKey"`

	// Git like DAG
	NodeKind   ConfigDagNodeKind       `json:"node_kind"`
	Hash       util.ConfigVersionHash  `json:"config_version_hash"`
	ParentId   *util.ConfigVersionId   `json:"parent_id"`
	ParentHash *util.ConfigVersionHash `json:"parent_hash"`

	Note string `json:"note"`

	// AdditionalParents []ParentRef             `json:"additional_parents"`
	Data *ConfigDataObject `json:"data" gorm:"type:jsonb"`
}

type ConfigVersionORM struct {
	Id util.ConfigVersionId `json:"id"`

	// AccountId util.AccountId       `json:"account_id" gorm:"index;not null"`

	// CreatedAt time.Time  `json:"created_at"`
	// UpdatedAt time.Time  `json:"updated_at"`
	// DeletedAt *time.Time `json:"deleted_at"`

	ConfigDagNodeEmbed `json:",inline" gorm:"embedded"`
}

func (c *ConfigVersionORM) TableName() string {
	return "config_versions"
}

//
// ConfigReferenceORM
//

type ConfigStageKind string

const (
	ConfigStageKindProd             ConfigStageKind = "prod"
	ConfigStageKindBeta             ConfigStageKind = "beta"
	ConfigStageKindClientValidation ConfigStageKind = "client_validation"
	ConfigStageKindTest             ConfigStageKind = "test"
	ConfigStageKindDev              ConfigStageKind = "dev"
)

type ConfigStageAudienceKind string

const (
	ConfigStageAudienceKindClient    ConfigStageAudienceKind = "client"
	ConfigStageAudienceKindMarketing ConfigStageAudienceKind = "marketing"
	ConfigStageAudienceKindInternal  ConfigStageAudienceKind = "internal"
	ConfigStageAudienceKindPartner   ConfigStageAudienceKind = "partner"
	ConfigStageAudienceKindTest      ConfigStageAudienceKind = "test"
	ConfigStageAudienceKindDev       ConfigStageAudienceKind = "dev"
	ConfigStageAudienceKindOther     ConfigStageAudienceKind = "other"
)

type ConfigReferenceKind string

const (
	// ConfigReferenceKindGlobalRoot ConfigReferenceKind = "global_root"
	// ConfigReferenceKindRegionRoot ConfigReferenceKind = "region_root"
	// ConfigReferenceKindAZRoot     ConfigReferenceKind = "az_root"

	// ConfigReferenceKindDomainRoot   ConfigReferenceKind = "domain_root"
	// ConfigReferenceKindServerRoot   ConfigReferenceKind = "server_root"
	// ConfigReferenceKindInstanceRoot ConfigReferenceKind = "instance_root"

	// ConfigReferenceKindCustomDomainRoot ConfigReferenceKind = "custom_domain_root"

	// ConfigReferenceKindAccountRoot    ConfigReferenceKind = "account_root"
	// ConfigReferenceKindSubAccountRoot ConfigReferenceKind = "sub_account_root"
	// ConfigReferenceKindUserRoot       ConfigReferenceKind = "user_root"

	ConfigReferenceKindRoot ConfigReferenceKind = "root"
	ConfigReferenceKindHead ConfigReferenceKind = "head"

	// TODO: Add a scope kind
	// ConfigReferenceKindAccountCurrentHead ConfigReferenceKind = "account_current_head"
	// ConfigReferenceKindUserCurrentHead    ConfigReferenceKind = "account_current_head"

	// The earliest version of the config for the given stage
	ConfigReferenceKindStageRoot   ConfigReferenceKind = "stage_root"
	ConfigReferenceKindTaggedStage ConfigReferenceKind = "tagged_stage"
	ConfigReferenceKindTag         ConfigReferenceKind = "tag"
)

type ConfigTagObject struct {
	AccountId util.AccountId `json:"account_id" gorm:"uniqueIndex:config_tag_object_tag;not null"`
	UserId    *util.UserId   `json:"user_id" gorm:"uniqueIndex:config_tag_object_tag;null"`

	Tag        string            `json:"tag" gorm:"uniqueIndex:config_tag_object_tag;not null"`
	VersionRef *ConfigVersionRef `json:"version_ref"`

	ConfigReferenceKind     *ConfigReferenceKind     `json:"config_reference_kind" gorm:"index"`
	ConfigStageAudienceKind *ConfigStageAudienceKind `json:"config_stage_audience_kind" gorm:"index"`
	ConfigStageKind         *ConfigStageKind         `json:"config_stage_kind" gorm:"index"`
}

type ConfigTagORM struct {
	ConfigTagObject `json:",inline" gorm:"embedded"`
}

func (c *ConfigTagORM) TableName() string {
	return "config_tags"
}

type ConfigReferenceORM struct {
	Scope     util.ScopeKind `json:"scope" gorm:"uniqueIndex:ref_unique;not null"`
	AccountId util.AccountId `json:"account_id" gorm:"uniqueIndex:ref_unique;not null"`
	UserId    *util.UserId   `json:"user_id" gorm:"uniqueIndex:ref_unique"`

	ConfigReferenceKind ConfigReferenceKind `json:"reference_kind" gorm:"uniqueIndex:ref_unique;not null"`

	VersionRef *ConfigVersionRef `json:"version_ref" gorm:"type:jsonb;not null"`
}

func (c *ConfigReferenceORM) TableName() string {
	return "config_refs"
}

func (c *ConfigReferenceORM) AddIndexes(ctx context.Context, tx *sql.Tx) error {
	logger := util.NewLogger("ConfigNodeORM.AddIndexes", 0)

	tableName := c.TableName()

	exec := func(query string) error {
		logger.Printf("Executing query: %s\n", query)
		_, err := tx.ExecContext(ctx, query)
		return err
	}

	// Add a unique index on the account_id and config_reference_kind,
	// where scope is account.
	err := exec(fmt.Sprintf(`
		CREATE UNIQUE INDEX IF NOT EXISTS %s_account_scope_reference_kind_idx
		ON %s (account_id, config_reference_kind)
		WHERE scope = 'account'
	`, tableName, tableName))
	if err != nil {
		logger.Printf("Error creating index (%s_account_scope_reference_kind_idx): %v\n", tableName, err)
		return err
	}

	// Add a unique index on the account_id, user_id, and config_reference_kind,
	// where scope is user.
	err = exec(fmt.Sprintf(`
		CREATE UNIQUE INDEX IF NOT EXISTS %s_user_scope_reference_kind_idx
		ON %s (account_id, user_id, config_reference_kind)
		WHERE scope = 'user'
	`, tableName, tableName))
	if err != nil {
		logger.Printf("Error creating index (%s_user_scope_reference_kind_idx): %v\n", tableName, err)
		return err
	}

	return nil
}

func (c *ConfigReferenceORM) AddConstraints(ctx context.Context, tx *sql.Tx) error {
	logger := util.NewLogger("ConfigNodeORM.AddConstraints", 0)

	tableName := c.TableName()

	exec := func(query string) error {
		logger.Printf("Executing query: %s\n", query)
		_, err := tx.ExecContext(ctx, query)
		return err
	}

	// Scope must be either account or user.
	err := exec(fmt.Sprintf(`
		ALTER TABLE %s
		ADD CONSTRAINT %s_valid_scope
		CHECK (scope IN ('account', 'user'))
	`, tableName, tableName))
	if err != nil {
		logger.Printf("Error adding constraint (%s_valid_scope): %v\n", tableName, err)
		return err
	}

	// If scope is not user, then user_id must be null.
	err = exec(fmt.Sprintf(`
		ALTER TABLE %s
		ADD CONSTRAINT %s_account_scope_user_id_null
		CHECK (scope = 'user' OR user_id IS NULL)
	`, tableName, tableName))
	if err != nil {
		logger.Printf("Error adding constraint (%s_account_scope_user_id_null): %v\n", tableName, err)
		return err
	}

	// If scope is user, then user_id must be non null.
	err = exec(fmt.Sprintf(`
		ALTER TABLE %s
		ADD CONSTRAINT %s_user_scope_user_id_not_null
		CHECK (scope != 'user' OR user_id IS NOT NULL)
	`, tableName, tableName))
	if err != nil {
		logger.Printf("Error adding constraint (%s_user_scope_user_id_not_null): %v\n", tableName, err)
		return err
	}

	return nil
}

func (c *ConfigReferenceORM) RemoveIndexes(ctx context.Context, tx *sql.Tx) error {
	logger := util.NewLogger("ConfigNodeORM.RemoveIndexes", 0)

	tableName := c.TableName()

	// Remove the unique index on the account_id and config_reference_kind,
	// where scope is account.
	_, err := tx.ExecContext(ctx, fmt.Sprintf(`
		DROP INDEX IF EXISTS %s_account_scope_reference_kind_idx
	`, tableName))
	if err != nil {
		logger.Printf("Error dropping index (%s_account_scope_reference_kind_idx): %v\n", tableName, err)
		return err
	}

	// Remove the unique index on the account_id, user_id, and config_reference_kind,
	// where scope is user.
	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
		DROP INDEX IF EXISTS %s_user_scope_reference_kind_idx
	`, tableName))
	if err != nil {
		logger.Printf("Error dropping index (%s_user_scope_reference_kind_idx): %v\n", tableName, err)
		return err
	}

	return nil
}

func (c *ConfigReferenceORM) RemoveConstraints(ctx context.Context, tx *sql.Tx) error {
	logger := util.NewLogger("ConfigNodeORM.RemoveConstraints", 0)

	tableName := c.TableName()

	// Remove the constraint ensuring that scope is either account or user.
	_, err := tx.ExecContext(ctx, fmt.Sprintf(`
		ALTER TABLE %s
		DROP CONSTRAINT %s_valid_scope
	`, tableName, tableName))
	if err != nil {
		logger.Printf("Error dropping constraint (%s_valid_scope): %v\n", tableName, err)
		return err
	}

	// Remove the constraint ensuring that if scope is account, then user_id is null.
	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
		ALTER TABLE %s
		DROP CONSTRAINT %s_account_scope_user_id_null
	`, tableName, tableName))
	if err != nil {
		logger.Printf("Error dropping constraint (%s_account_scope_user_id_null): %v\n", tableName, err)
		return err
	}

	// Remove the constraint ensuring that if scope is user, then user_id is non null.
	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
		ALTER TABLE %s
		DROP CONSTRAINT %s_user_scope_user_id_not_null
	`, tableName, tableName))
	if err != nil {
		logger.Printf("Error dropping constraint (%s_user_scope_user_id_not_null): %v\n", tableName, err)
		return err
	}

	return nil
}
