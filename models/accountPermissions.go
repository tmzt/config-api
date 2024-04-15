package models

import (
	"fmt"
	"time"

	"github.com/tmzt/config-api/util"
)

type AccountPermissionsORM struct {
	AccountId string
	Account   *AccountORM `gorm:"foreignKey:AccountId;references:Id"`
	Id        string      `gorm:"uniqueIndex"`
	UserId    string      `gorm:"primaryKey"`
	User      *UserORM    `gorm:"foreignKey:UserId;references:Id"`

	AccountAdmin           bool
	AccountBuyer           bool
	AccountOwner           bool
	AccountUser            bool
	AccountEmailListReader bool
}

func (a *AccountPermissionsORM) TableName() string {
	return "account_user_permissions"
}

func (a *AccountPermissionsORM) SetPrimaryRole(role util.AccountRoleKind) {
	a.AccountAdmin = false
	a.AccountBuyer = false
	a.AccountOwner = false
	a.AccountUser = false
	switch role {
	case util.AccountRoleKindAdmin:
		a.AccountAdmin = true
	case util.AccountRoleKindBuyer:
		a.AccountBuyer = true
	case util.AccountRoleKindOwner:
		a.AccountOwner = true
	case util.AccountRoleKindUser:
		a.AccountUser = true
	}
}

func (a *AccountPermissionsORM) Cache() *AccountPermissionsCache {
	return &AccountPermissionsCache{
		AccountId: a.AccountId,
		UserId:    a.UserId,
		Detail:    a.Detail(),
	}
}

func (a *AccountPermissionsORM) Detail() *AccountPermissionsDetail {
	return &AccountPermissionsDetail{
		AccountAdmin:           a.AccountAdmin,
		AccountBuyer:           a.AccountBuyer,
		AccountOwner:           a.AccountOwner,
		AccountUser:            a.AccountUser,
		AccountEmailListReader: a.AccountEmailListReader,
	}
}

func (a *AccountPermissionsORM) Update(detail *AccountPermissionsDetail) {
	a.AccountAdmin = detail.AccountAdmin
	a.AccountBuyer = detail.AccountBuyer
	a.AccountOwner = detail.AccountOwner
	a.AccountUser = detail.AccountUser
	a.AccountEmailListReader = detail.AccountEmailListReader
}

type AccountUserPrimaryRole string

func (a *AccountPermissionsORM) GetPrimaryRole() *util.AccountRoleKind {
	if a.AccountOwner {
		role := util.AccountRoleKindOwner
		return &role
	}
	if a.AccountAdmin {
		role := util.AccountRoleKindAdmin
		return &role
	}
	if a.AccountBuyer {
		role := util.AccountRoleKindBuyer
		return &role
	}
	if a.AccountUser {
		role := util.AccountRoleKindUser
		return &role
	}
	return nil
}

func (a *AccountPermissionsORM) AccountPermissionsList() []string {
	permissions := []string{}
	if a.AccountAdmin {
		permissions = append(permissions, "acct_admin")
	}
	if a.AccountBuyer {
		permissions = append(permissions, "acct_buyer")
	}
	if a.AccountOwner {
		permissions = append(permissions, "acct_owner")
	}
	if a.AccountUser {
		permissions = append(permissions, "acct_user")
	}
	if a.AccountEmailListReader {
		permissions = append(permissions, "acct_eml_lst_rdr")
	}
	return permissions
}

type AccountPermissionsDetail struct {
	AccountAdmin           bool `json:"acct_admin"`
	AccountBuyer           bool `json:"acct_buyer"`
	AccountOwner           bool `json:"acct_owner"`
	AccountUser            bool `json:"acct_user"`
	AccountEmailListReader bool `json:"acct_eml_lst_rdr"`
}

type AccountPermissionsCache struct {
	AccountId string
	UserId    string
	Detail    *AccountPermissionsDetail
}

func (a AccountPermissionsCache) CacheKey() string {
	return fmt.Sprintf("account_user_permissions:%s:%s", a.AccountId, a.UserId)
}

func (a AccountPermissionsCache) Ttl() time.Duration {
	return time.Duration(0)
}
