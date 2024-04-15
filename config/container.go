package config

import (
	"fmt"
	"log"
)

// The contents of this file are intended to be
// used only in this package.

// Modification of the internal object is protected
// so that the config version can be accurately tracked
// and the DAG updated.

// This is accomplished by requiring a "handle" to be
// passed to the SetConfigData method. The handle wraps
// access to the the handle service, which maintains
// a cache of the config version for each account and
// (optional) user.

// This object contains the configDataObjectInternal
// allowing for interior mutation
type configDataObjectInternalContainer struct {
	hasBeenSet bool
	contents   *configDataObjectContents //`json:",inline" gorm:"embedded"`
}

func createConfigDataObjectInternalContainer() *configDataObjectInternalContainer {
	// contents := &configDataObjectContents{}

	return &configDataObjectInternalContainer{
		hasBeenSet: false,
		contents:   nil,
	}
}

// Allow marshalling and unmarshalling of the internal object
// by invoking it's own marshalling and unmarshalling methods

func (c *configDataObjectInternalContainer) MarshalJSON() ([]byte, error) {
	log.Printf("configDataObjectInternalContainer.MarshalJSON() calling contents.MarshalJSON(): contents: %+v\n", c.contents)
	if c.contents == nil {
		// return nil, fmt.Errorf("contents is nil")
		return []byte("null"), nil
	}
	return c.contents.MarshalJSON()
}

func (c *configDataObjectInternalContainer) UnmarshalJSON(data []byte) error {
	return c.contents.UnmarshalJSON(data)
}

// Implement custom data types for sql and gorm

func (c *configDataObjectInternalContainer) scanJson(src interface{}) error {
	switch v := src.(type) {
	case []byte:
		return c.contents.UnmarshalJSON(v)
	case string:
		return c.contents.UnmarshalJSON([]byte(v))
	default:
		return fmt.Errorf("unsupported type for configDataObjectInternalContainer.Scan: %T", src)
	}
}

func (c *configDataObjectInternalContainer) Scan(src interface{}) error {
	return c.scanJson(src)
}

func (c *configDataObjectInternalContainer) Value() (interface{}, error) {
	return c.MarshalJSON()
}
