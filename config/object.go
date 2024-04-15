package config

import (
	"database/sql/driver"
	"fmt"
	"log"

	"github.com/tmzt/config-api/util"
)

// type ConfigDataRecordConverter interface {
// 	// Populate the record from a `ConfigDataObject`
// 	FromConfigDataObject(configDataObject *ConfigDataObject) error

// 	// Update the `ConfigDataObject` from the record and return the updated `ConfigDataObject`
// 	ToConfigDataObject(configDataObject *ConfigDataObject) (*ConfigDataObject, error)
// }

type ConfigDataObject struct {
	internalContainer *configDataObjectInternalContainer `json:"-" gorm:"data_object;type:jsonb;not null"`
}

func NewConfigDataObject() *ConfigDataObject {
	return &ConfigDataObject{
		internalContainer: &configDataObjectInternalContainer{
			contents:   nil,
			hasBeenSet: false,
		},
	}
}

// func (c *ConfigDataObject) SetConfigData(ctx context.Context, src interface{}, handle ConfigSettingHandle) (*ConfigDataObject, error) {

// 	// internalHandle, err := c.getInternalHandle(handle)
// 	// if err != nil {
// 	// 	log.Printf("ConfigDataObject.SetConfigData error calling c.getInternalHandle(): %v\n", err)
// 	// 	return nil, err
// 	// } else if internalHandle == nil {
// 	// 	// This should never happen
// 	// 	log.Printf("ConfigDataObject.SetConfigData error: internalHandle is nil\n")
// 	// 	return nil, NewInvalidConfigSettingHandle(fmt.Errorf("internalHandle is nil"))
// 	// }

// 	// handleSvc := internalHandle.handleService
// 	// if handleSvc == nil {
// 	// 	return nil, NewInvalidConfigSettingHandle(fmt.Errorf("handleSvc is nil"))
// 	// }

// 	// Create the object
// 	// contents, err := handleSvc.createObjectInternal(ctx, internalHandle.accountId, internalHandle.userId, internalHandle.parentRef, src, internalHandle)
// 	// if err != nil {
// 	// 	return nil, err
// 	// }

// 	// log.Printf("ConfigDataObject.SetConfigData() setting contents: %+v\n", contents)

// 	// Set the data object
// 	c.internalContainer.contents = contents
// 	c.internalContainer.hasBeenSet = true

// 	return c, nil
// }

// func (c *ConfigDataObject) setContentsInternal(contents *configDataObjectContents) {
// 	c.internalContainer.contents = contents
// 	c.internalContainer.hasBeenSet = true
// }

// func (c *ConfigDataObject) SetConfigData(data util.Data, handle ConfigContextHandle) error {

// }

func (c *ConfigDataObject) GetConfigData() util.Data {
	// contents, err := c.getContents()
	// if err != nil {
	// 	log.Printf("ConfigDataObject.GetConfigData error calling c.getContents(): %v\n", err)
	// 	return nil
	// }

	contents := c.getContentsInternal()

	return util.Data(contents.innerData)
}

func (c *ConfigDataObject) GetConfigVersionRef() *ConfigVersionRef {
	// contents, err := c.getContents()
	// if err != nil {
	// 	log.Printf("ConfigDataObject.GetConfigVersionRef error calling c.getContents(): %v\n", err)
	// 	return nil
	// }

	contents := c.getContentsInternal()
	metadata := contents.metadata

	return &metadata.VersionRef
}

func (c *ConfigDataObject) GetConfigParentVersionRef() *ConfigVersionRef {
	// contents, err := c.getContentsInternal()
	// if err != nil {
	// 	log.Printf("ConfigDataObject.GetConfigVersionRef error calling c.getContents(): %v\n", err)
	// 	return nil
	// }

	contents := c.getContentsInternal()
	metadata := contents.metadata

	return metadata.ParentRef
}

// Internals

func (c *ConfigDataObject) getOrCreateContainer() *configDataObjectInternalContainer {
	if c == nil {
		panic("ConfigDataObject is nil calling getOrCreateContainer()")
	}

	log.Printf("ConfigDataObject.getOrCreateContainer() called with internalContainer(ptr): %+v\n", c.internalContainer)

	if c.internalContainer == nil {
		log.Printf("ConfigDataObject.getOrCreateContainer() creating internalContainer\n")
		c.internalContainer = createConfigDataObjectInternalContainer()
	}

	log.Printf("ConfigDataObject.getOrCreateContainer() returning internalContainer(deref): %+v\n", *c.internalContainer)

	log.Printf("ConfigDataObject.getOrCreateContainer() returning internalContainer.contents(ptr): %+v\n", c.internalContainer.contents)
	if c.internalContainer.contents != nil {
		log.Printf("ConfigDataObject.getOrCreateContainer() returning internalContainer.contents(deref): %+v\n", *c.internalContainer.contents)
	} else {
		log.Printf("ConfigDataObject.getOrCreateContainer() returning internalContainer.contents(deref): nil\n")
	}

	return c.internalContainer
}

// func (c *ConfigDataObject) getInternalContainer() (*configDataObjectInternalContainer, error) {
// 	container := c.internalContainer
// 	if container == nil {
// 		// This should not happen if the container was initialized properly
// 		return nil, fmt.Errorf("container is nil")
// 	}

// 	return container, nil
// }

// func (c *ConfigDataObject) getInternalHandle(handle ConfigSettingHandle) (*configSettingHandleInternal, error) {
// 	if handle == nil {
// 		return nil, NewInvalidConfigSettingHandle(fmt.Errorf("handle is nil"))
// 	}

// 	container := c.getOrCreateContainer()

// 	if container == nil {
// 		// This should never happen
// 		return nil, NewInvalidConfigSettingHandle(fmt.Errorf("container is nil"))
// 	}

// 	rawHandle := handle.(interface{})

// 	// Get the internal handle
// 	internalHandle, ok := rawHandle.(*configSettingHandleInternal)
// 	if !ok {
// 		return nil, NewInvalidConfigSettingHandle(fmt.Errorf("could not get *configSettingHandleInternal from rawInterface"))
// 	}

// 	// Make sure the handle is not nil
// 	if internalHandle == nil {
// 		return nil, NewInvalidConfigSettingHandle(fmt.Errorf("internalHandle is nil"))
// 	}

// 	// Make sure the handle is not consumed
// 	if internalHandle.consumed.Load() {
// 		return nil, NewConfigSettingHandleConsumed()
// 	}

// 	return internalHandle, nil
// }

func (c *ConfigDataObject) getContentsInternal() *configDataObjectContents {
	// // container := c.getOrCreateContainer()
	// container := c.internalContainer
	// if container == nil {
	// 	// This should never happen
	// 	return nil, fmt.Errorf("container is nil")
	// }

	// return container.contents, nil
	return c.internalContainer.contents
}

// Allow marshalling and unmarshalling of the inner data object
func (c *ConfigDataObject) MarshalJSON() ([]byte, error) {
	log.Printf("ConfigDataObject.MarshalJSON() called\n")

	if c.internalContainer == nil {
		return nil, fmt.Errorf("internalContainer is nil callling ConfigDataObject.MarshalJSON()")
	}

	log.Printf("ConfigDataObject.MarshalJSON() calling c.internalContainer.MarshalJSON(): c.internalContainer: %+v\n", c.internalContainer)
	return c.internalContainer.MarshalJSON()
}

func (c *ConfigDataObject) UnmarshalJSON(data []byte) error {
	container := c.getOrCreateContainer()
	if container == nil {
		// This should never happen
		return fmt.Errorf("container is nil")
	}

	return container.UnmarshalJSON(data)
}

// Implement custom data types for sql and gorm

func (c *ConfigDataObject) Scan(src interface{}) error {
	log.Printf("ConfigDataObject.Scan() called\n")

	if src == nil {
		log.Printf("ConfigDataObject.Scan() called with nil src\n")
		return nil
	}

	container := c.getOrCreateContainer()
	if container == nil {
		// This should never happen
		return fmt.Errorf("container is nil")
	}

	return container.Scan(src)

	// if c == nil {
	// 	log.Printf("ConfigDataObject.Scan() called with nil object\n")
	// 	return fmt.Errorf("called with nil object")
	// }

	// switch src := src.(type) {
	// case []byte:
	// 	log.Printf("ConfigDataObject.Scan() called with []byte\n")

	// 	return c.UnmarshalJSON(src)
	// case string:
	// 	log.Printf("ConfigDataObject.Scan() called with string\n")

	// 	return c.UnmarshalJSON([]byte(src))
	// default:
	// 	log.Printf("ConfigDataObject.Scan() called with unknown type\n")

	// 	return fmt.Errorf("unsupported type: %T", src)
	// }
}

func (c ConfigDataObject) Value() (driver.Value, error) {
	log.Printf("ConfigDataObject.Value() called\n")
	return c.MarshalJSON()
}
