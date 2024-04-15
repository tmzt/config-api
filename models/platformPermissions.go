package models

import (
	"fmt"
	"time"
)

type PlatformPermissionsORM struct {
	AccountId string
	Account   *AccountORM `gorm:"foreignKey:AccountId;references:Id"`
	Id        string      `gorm:"uniqueIndex"`
	UserId    string      `gorm:"primaryKey"`
	User      *UserORM    `gorm:"foreignKey:UserId;references:Id"`

	RootUser     bool
	Admin        bool
	InternalUser bool
}

func (r *PlatformPermissionsORM) TableName() string {
	return "platform_permissions"
}

func (a *PlatformPermissionsORM) Cache() *PlatformPermissionsCache {
	return &PlatformPermissionsCache{
		UserID: a.UserId,
		Detail: a.Detail(),
	}
}

func (a *PlatformPermissionsORM) Detail() *PlatformPermissionsDetail {
	return &PlatformPermissionsDetail{
		PlatformUser:  a.RootUser,
		PlatformAdmin: a.Admin,
		InternalUser:  a.InternalUser,
	}
}

func (a *PlatformPermissionsORM) Update(detail *PlatformPermissionsDetail) {
	a.RootUser = detail.PlatformUser
	a.Admin = detail.PlatformAdmin
	a.InternalUser = detail.InternalUser
}

func (a *PlatformPermissionsORM) RootUserPermissionsList() []string {
	permissions := []string{}
	if a.RootUser {
		permissions = append(permissions, "platform_user")
	}

	if a.InternalUser {
		permissions = append(permissions, "internal_user")
	}
	return permissions
}

type PlatformPermissionsCache struct {
	UserID string
	Detail *PlatformPermissionsDetail
}

func (a PlatformPermissionsCache) CacheKey() string {
	return fmt.Sprintf("platform_permissions:%s", a.UserID)
}

func (a PlatformPermissionsCache) Ttl() time.Duration {
	return time.Duration(0)
}

type PlatformPermissionsDetail struct {
	PlatformUser  bool `json:"platform_user"`
	PlatformAdmin bool `json:"platform_admin"`
	InternalUser  bool `json:"internal_user"`
}

// TODO: Add Update struct
