package util

import (
	"encoding/json"
	"time"

	"github.com/jpincas/gouuidv6"
	uuid "github.com/satori/go.uuid"
)

func ToJson(v interface{}) string {
	j, _ := json.Marshal(v)
	return string(j)
}

func ToJsonPretty(v interface{}) string {
	j, _ := json.MarshalIndent(v, "", "  ")
	return string(j)
}

func NewUUID() string {
	return uuid.NewV4().String()
}

func NewUUIDV6B64() string {
	return gouuidv6.NewB64().String()
}

func NewUUIDV6B64FromTime(t time.Time) string {
	return gouuidv6.NewB64FromTime(t).String()
}
