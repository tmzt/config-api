package util

import (
	"database/sql/driver"
	"strconv"
	"time"
)

func BoolPtr(b bool) *bool {
	return &b
}

func ParseBoolPtrOrNil(s string) *bool {
	if s == "" {
		return nil
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return nil
	}
	return &b
}

func StrPtr(s string) *string {
	return &s
}

func StrPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func StrPtrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func DebugStr(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func Int64Ptr(i int64) *int64 {
	return &i
}

func ParseInt64PtrOrNil(s string) *int64 {
	if s == "" {
		return nil
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil
	}
	return &i
}

func Int64Or(i *int64, or int64) int64 {
	if i == nil {
		return or
	}
	return *i
}

func TimePtr(t time.Time) *time.Time {
	return &t
}

func ParseTimePtrOrNil(s string, pattern string) *time.Time {
	if s == "" {
		return nil
	}
	if pattern == "" {
		pattern = time.RFC3339
	}
	t, err := time.Parse(pattern, s)
	if err != nil {
		return nil
	}
	return &t
}

func DurationPtr(d time.Duration) *time.Duration {
	return &d
}

func TimeOrNow(t *time.Time) time.Time {
	if t == nil {
		return time.Now()
	}
	return *t
}

func AccountIdPtr(s string) *AccountId {
	id := AccountId(s)
	return &id
}

func AccountIdAsPtr(s AccountId) *AccountId {
	return &s
}

func AccountIdPtrStr(s *AccountId) string {
	if s == nil {
		return ""
	}
	return string(*s)
}

func ConfigVersionHashPtr(s ConfigVersionHash) *ConfigVersionHash {
	return &s
}

func ConfigKeyPtr(s string) *ConfigKey {
	id := ConfigKey(s)
	return &id
}

func ConfigRecordKeyPtr(s string) *ConfigRecordKey {
	id := ConfigRecordKey(s)
	return &id
}

func ConfigRecordKeyAsPtr(s ConfigRecordKey) *ConfigRecordKey {
	return &s
}

func ConfigRecordKeyStr(s *ConfigRecordKey) string {
	if s == nil {
		return ""
	}
	return string(*s)
}

func ConfigCollectionKeyPtr(s string) *ConfigCollectionKey {
	id := ConfigCollectionKey(s)
	return &id
}

func ConfigCollectionKeyStr(s *ConfigCollectionKey) string {
	if s == nil {
		return ""
	}
	return string(*s)
}

func ConfigItemKeyPtr(s string) *ConfigItemKey {
	id := ConfigItemKey(s)
	return &id
}

func ConfigItemKeyStr(s *ConfigItemKey) string {
	if s == nil {
		return ""
	}
	return string(*s)
}

func UserIdPtr(s string) *UserId {
	id := UserId(s)
	return &id
}

func UserIdPtrStr(s *UserId) string {
	if s == nil {
		return ""
	}
	return string(*s)
}

func UserIdStrOrNil(s *UserId) *string {
	if s == nil {
		return nil
	}
	return StrPtr(string(*s))
}

func CheckoutTransactionIdPtr(s string) *CheckoutTransactionId {
	id := CheckoutTransactionId(s)
	return &id
}

func CheckoutTokenIdPtr(s string) *CheckoutTokenId {
	id := CheckoutTokenId(s)
	return &id
}

func CheckoutTokenIdPtrStr(s *CheckoutTokenId) string {
	if s == nil {
		return ""
	}
	return string(*s)
}

func CheckoutTransactionIdPtrStr(s *CheckoutTransactionId) string {
	if s == nil {
		return ""
	}
	return string(*s)
}

func StripeTransactionIdPtr(s string) *StripeTransactionId {
	id := StripeTransactionId(s)
	return &id
}

func StripeTransactionIdPtrStr(s *StripeTransactionId) string {
	if s == nil {
		return ""
	}
	return string(*s)
}

type AccountId string
type UserId string

type ConfigId string
type ConfigVersionId string
type ConfigVersionHash string

// SHA256 of an empty object, used as the hash value of an "empty" version
// (i.e. a version with no data such as a root)
const EmptyHash ConfigVersionHash = "44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a"

type ConfigTagId string

type ConfigKey string
type ConfigDocumentKey string
type ConfigDocumentId string

type ConfigRecordKey string
type ConfigRecordId string
type ConfigCollectionKey string
type ConfigItemKey string

type ConfigSchemaContents Data
type ConfigSchemaHash string
type ConfigSchemaName string
type ConfigSchemaIdValue string

type ConfigSchemaListHash string

func (v *ConfigRecordKey) String() string {
	return ConfigRecordKeyStr(v)
}
func (v *ConfigCollectionKey) String() string {
	return ConfigCollectionKeyStr(v)
}

func (v *ConfigItemKey) String() string {
	return ConfigItemKeyStr(v)
}

type EmailId string
type PlatformSignupId string
type StripeConnectionId string
type StripeAccountId string

type CustomerAccountId string
type CustomerUserId string

type ProductId string
type SubscriptionOfferingDataId string

type PurchaseId string
type PurchaseItemId string

type StripeTransactionId string

type CheckoutTokenId string
type CheckoutTransactionId string

type AccountRoleKind string

const (
	AccountRoleKindOwner AccountRoleKind = "owner"
	AccountRoleKindAdmin AccountRoleKind = "admin"
	AccountRoleKindUser  AccountRoleKind = "user"
	AccountRoleKindBuyer AccountRoleKind = "buyer"
)

func (v AccountId) Value() (driver.Value, error) {
	s := string(v)
	// log.Printf("AccountId.Value() called: s: << %v >>\n", s)
	if s == "" {
		// log.Printf("AccountId.Value() called: returning nil\n")
		return nil, nil
	}

	// log.Printf("AccountId.Value() called: v: << %v >>\n", v)
	res := []byte(string(v))
	// log.Printf("AccountId.Value() called: %v\n", res)
	return res, nil
}

// func (v *AccountId) Value() (driver.Value, error) {
// 	return AccountIdPtrStr(v), nil
// }
