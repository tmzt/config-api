package config

import (
	"encoding/json"
	"log"
	"time"

	"github.com/tmzt/config-api/util"
)

// The contents of this file are intended to be
// used only in this package.

type innerData map[string]interface{}

type ConfigResourceData map[string]interface{}

type ConfigMetadata struct {
	CreatedAt time.Time `json:"created_at"`
	// TODO: Make this non-optional and always pass the user id
	// use ScopeKind to determine if it's an account or user scope
	CreatedBy *util.UserId `json:"created_by"`

	VersionRef ConfigVersionRef  `json:"version_ref"`
	ParentRef  *ConfigVersionRef `json:"parent_ref"`

	// Information on the associated resource
	// ResourceData ConfigResourceData `json:"resource_data"`
}

// This is the actual object returned by CreateObject
type configDataObjectContents struct {
	metadata  ConfigMetadata
	innerData innerData
}

func (c *configDataObjectContents) GetMetadata() *ConfigMetadata {
	return &c.metadata
}

func (c *configDataObjectContents) ComputeHash() (*string, error) {
	hash, err := hashObject(*c)
	if err != nil {
		return nil, err
	}
	return hash, nil
}

// func (c *configDataObjectContents) GetConfigImmutableEmbed() *ConfigImmutableDataEmbed {
// 	return &c.ConfigImmutableDataEmbed
// }

func (c *configDataObjectContents) GetConfigData() util.Data {
	return util.Data(c.innerData)
}

// Allow marshalling and unmarshalling of the contents
// via a wrapper

type configDataObjectContentsWrapper struct {
	Metadata  ConfigMetadata `json:"metadata"`
	InnerData util.Data      `json:"data"`
}

func (c *configDataObjectContents) MarshalJSON() ([]byte, error) {

	// if string(c.AccountId) == "" {
	// 	log.Printf("configDataObjectContents.MarshalJSON() AccountId is empty\n")
	// 	return nil, fmt.Errorf("AccountId is empty")
	// }

	// Wrap contents
	wrapped := configDataObjectContentsWrapper{
		Metadata:  c.metadata,
		InnerData: util.Data(c.innerData),

		// ConfigImmutableEmbedOld: c.ConfigImmutableEmbedOld,
	}

	b, err := json.Marshal(wrapped)
	log.Printf("configDataObjectContents.MarshalJSON() wrapped: %+v\n", wrapped)
	log.Printf("configDataObjectContents.MarshalJSON() string(b): %s\n", string(b))

	return b, err
}

func (c *configDataObjectContents) UnmarshalJSON(data []byte) error {
	// Unwrap contents
	wrapped := configDataObjectContentsWrapper{}

	err := json.Unmarshal(data, &wrapped)
	if err != nil {
		return err
	}

	c.innerData = innerData(wrapped.InnerData)
	c.metadata = wrapped.Metadata
	// c.versionRef = wrapped.VersionRef
	// c.ConfigImmutableEmbedOld = wrapped.ConfigImmutableEmbedOld

	return nil
}
