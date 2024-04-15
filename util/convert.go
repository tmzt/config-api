package util

import (
	"encoding/json"
	"fmt"
	"log"

	"dario.cat/mergo"
	"github.com/elgris/stom"
	"github.com/mitchellh/mapstructure"
)

type Data map[string]interface{}

type ConversionError struct {
	Err error
}

func NewConversionError(err error) *ConversionError {
	return &ConversionError{Err: err}
}

func (e ConversionError) Error() string {
	return e.Err.Error()
}

func (e ConversionError) Unwrap() error {
	return e.Err
}

func FromDataMap(src interface{}, dest interface{}) error {

	decoderConfig := &mapstructure.DecoderConfig{
		TagName: "json",
		Result:  dest,
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return NewConversionError(fmt.Errorf("error creating decoder: %w", err))
	}

	err = decoder.Decode(src)
	if err != nil {
		return NewConversionError(err)
	}
	return nil
}

func ToDataMap(src interface{}) (Data, error) {
	var dataMap Data

	log.Printf("ToDataMap: src(%T): %+v\n", src, src)

	j, _ := json.MarshalIndent(src, "", "  ")
	log.Printf("ToDataMap: src(as json): %s\n", j)

	if d, ok := src.(*Data); ok {
		log.Printf("ToDataMap(using *Data): src(%T): %+v\n", src, src)
		dataMap = Data(*d)
	} else if d, ok := src.(Data); ok {
		log.Printf("ToDataMap(using Data): src(%T): %+v\n", src, src)
		dataMap = Data(d)
	} else if d, ok := src.(map[string]interface{}); ok {
		log.Printf("ToDataMap(using map[string]interface{}): src(%T): %+v\n", src, src)
		dataMap = Data(d)
	} else if d, ok := src.(*map[string]interface{}); ok {
		log.Printf("ToDataMap(using *map[string]interface{}): src(%T): %+v\n", src, src)
		dataMap = Data(*d)
	} else {
		log.Printf("ToDataMap(using stom): src(%T): %+v\n", src, src)

		// Use json tags
		stom.SetTag("json")

		// Note: this option will be faster if the ToMap interface is implemented on the object
		d, err := stom.ConvertToMap(src)
		if err != nil {
			log.Printf("ToDataMap(using stom): error: %v\n", err)
			return nil, NewConversionError(err)
		}

		dataMap = Data(d)
	}

	log.Printf("ToDataMap: dataMap: %v\n", dataMap)

	return dataMap, nil
}

func MergeDataInto(dest, src interface{}) error {
	return mergo.Merge(&dest, src, mergo.WithOverride)
}
