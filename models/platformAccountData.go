package models

import "time"

// config_api Platform Account Data, used to store platform specific data
// as an extension to the shared Account model
type PlatformAccountDataORM struct {
	AccountId string     `json:"account_id"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
	DeletedAt *time.Time
	ApiDomain *string `json:"api_domain"`
}

func (p *PlatformAccountDataORM) TableName() string {
	return "platform_account_data"
}

type PlatformAccountDataDetail struct {
	AccountId string     `json:"account_id"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
	ApiDomain *string    `json:"api_domain"`
}

func (p *PlatformAccountDataORM) Detail() *PlatformAccountDataDetail {
	return &PlatformAccountDataDetail{
		AccountId: p.AccountId,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
		DeletedAt: p.DeletedAt,
		ApiDomain: p.ApiDomain,
	}
}

type UpdatePlatformAccountData struct {
	DeletedAt *time.Time `json:"deleted_at"`
	ApiDomain *string    `json:"api_domain"`
}
