package config

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"

	"github.com/tmzt/config-api/util"
)

func hashObject(src interface{}) (*string, error) {
	dataMap, err := toDataMapInternal(src)
	if err != nil {
		return nil, err
	}

	log.Printf("hashObject: dataMap: %v\n", dataMap)

	return hashConfigDataMapInternal(dataMap)
}

func hashConfigDataMapInternal(data innerData) (*string, error) {
	// For now, we will use the SHA256 hash of the JSON representation of the data
	// For this to work, we need to ensure the keys are sorted
	// TODO: Take a SHA256 hasher from the service

	hasher := sha256.New()

	err := json.NewEncoder(hasher).Encode(data)
	if err != nil {
		return nil, NewConfigObjectEncodingFailed(err)
	}

	hash := hasher.Sum(nil)

	// Convert to hex
	// TODO: Use a more efficient method
	s := fmt.Sprintf("%x", hash)

	return &s, nil
}

func toDataMapInternal(src interface{}) (innerData, error) {
	dataMap, err := util.ToDataMap(src)
	if err != nil {
		// TODO: Remove from this critical path
		log.Printf("toDataMapInternal: error: %v\n", err)
		return nil, NewInvalidConfigRecordType(err)
	}

	return innerData(dataMap), nil
}
