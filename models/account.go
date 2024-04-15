package models

import (
	"time"
)

type AccountState string

const (
	AccountStatePlatformSignup AccountState = "signup"
	AccountStateActive         AccountState = "active"
	AccountStateSuspended      AccountState = "suspended"
	AccountStateDeleted        AccountState = "deleted"
)

type AccountORM struct {
	ParentAccountId *string `gorm:"index"`
	Id              string  `gorm:"primary_key"`
	CreatedAt       *time.Time
	DeletedAt       *time.Time
	UpdatedAt       *time.Time
	State           AccountState
	// ApiDomain       string             `gorm:"uniqueIndex"`

	// Related entities
	PlatformAccountData *PlatformAccountDataORM `gorm:"foreignKey:AccountId;references:Id"`
}

func (a *AccountORM) TableName() string {
	return "accounts"
}

func (a *AccountORM) Detail() *AccountDetail {
	res := &AccountDetail{
		Id:        a.Id,
		CreatedAt: a.CreatedAt,
	}

	if a.PlatformAccountData != nil {
		res.ApiDomain = a.PlatformAccountData.ApiDomain
	}

	return res
}

type NewAccount struct {
	ParentAccountId *string `json:"parent_account_id"`
	ApiDomain       string  `json:"api_domain"`
}

type UpdateAccount struct {
	ApiDomain    *string       `json:"api_domain"`
	AccountState *AccountState `json:"state"`
}

type AccountDetail struct {
	Id        string     `json:"id"`
	CreatedAt *time.Time `json:"created_at"`
	ApiDomain *string    `json:"api_domain"`
}
