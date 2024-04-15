package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/tmzt/config-api/util"
)

type UserORM struct {
	AccountId string
	Account   *AccountORM `gorm:"foreignKey:AccountId;references:Id"`
	// BillingAddress  *AddressORM `gorm:"many2many:user_addresses"`
	CreatedAt    *time.Time
	DeletedAt    *time.Time
	Emails       []*EmailORM `gorm:"foreignKey:UserId;references:Id"`
	FirstName    string
	Id           string `gorm:"primary_key"`
	LastName     string
	LoginEmail   string
	LoginEmailId string
	// LoginEmailRef *EmailORM `gorm:"foreignKey:LoginEmailId;references:Id"`
	MiddleName string
	// ShippingAddress *AddressORM `gorm:"many2many:user_addresses"`
	UpdatedAt *time.Time

	// Related entities
	RootUserPermissions *PlatformPermissionsORM `gorm:"foreignKey:UserId;references:Id"`
	AccountPermissions  *AccountPermissionsORM  `gorm:"foreignKey:UserId;references:Id"`
}

func (u *UserORM) TableName() string {
	return "users"
}

func (u *UserORM) FriendlyName() string {
	firstName := u.FirstName
	lastName := u.LastName

	if firstName != "" && lastName != "" {
		return fmt.Sprintf("%s %s", firstName, lastName)
	} else if firstName == "" {
		return lastName
	} else if lastName == "" {
		return firstName
	} else {
		// First 3 characters of the email
		return u.LoginEmail[:3]
	}
}

func (u *UserORM) Initials() string {
	firstName := u.FirstName
	lastName := u.LastName
	middleName := u.MiddleName

	res := ""

	if firstName != "" {
		res += string(firstName[0])
	}

	if middleName != "" {
		res += string(middleName[0])
	}

	if lastName != "" {
		res += string(lastName[0])
	}

	if res == "" {
		// First 3 characters of the email
		res = u.LoginEmail[:3]
	}

	return strings.ToUpper(res)
}

func (u *UserORM) SetAccountUserPrimaryRole(roleKind util.AccountRoleKind) {
	if u.AccountPermissions == nil {
		u.AccountPermissions = &AccountPermissionsORM{
			AccountId: u.AccountId,
			Id:        util.NewUUID(),
			UserId:    u.Id,
		}
	}

	u.AccountPermissions.SetPrimaryRole(roleKind)
}

func (u *UserORM) Detail() *UserDetail {
	userDetail := &UserDetail{
		AccountId:    u.AccountId,
		Id:           u.Id,
		FirstName:    u.FirstName,
		FriendlyName: u.FriendlyName(),
		Initials:     u.Initials(),
		LastName:     u.LastName,
		LoginEmail:   u.LoginEmail,
		MiddleName:   u.MiddleName,
		Emails:       make([]*EmailDetail, 0),
	}

	for _, email := range u.Emails {
		userDetail.Emails = append(userDetail.Emails, &EmailDetail{
			Email:      email.Email,
			Subscribed: email.Subscribed,
			Verified:   email.Verified,
		})
	}

	if u.AccountPermissions != nil {
		userDetail.AccountPermissions = u.AccountPermissions.Detail()
		userDetail.AccountUserRole = u.AccountPermissions.GetPrimaryRole()
	}

	// if u.BillingAddress != nil {
	// 	userDetail.BillingAddress = Address{
	// 		City:       u.BillingAddress.City,
	// 		Country:    u.BillingAddress.Country,
	// 		Line1:      u.BillingAddress.Line1,
	// 		Line2:      u.BillingAddress.Line2,
	// 		PostalCode: u.BillingAddress.PostalCode,
	// 		State:      u.BillingAddress.State,
	// 	}
	// }

	// if u.ShippingAddress != nil {
	// 	userDetail.ShippingAddress = Address{
	// 		City:       u.ShippingAddress.City,
	// 		Country:    u.ShippingAddress.Country,
	// 		Line1:      u.ShippingAddress.Line1,
	// 		Line2:      u.ShippingAddress.Line2,
	// 		PostalCode: u.ShippingAddress.PostalCode,
	// 		State:      u.ShippingAddress.State,
	// 	}
	// }

	return userDetail
}

type UserDetail struct {
	AccountId       string         `json:"account_id"`
	Id              string         `json:"id"`
	Emails          []*EmailDetail `json:"emails"`
	FirstName       string         `json:"first_name"`
	FriendlyName    string         `json:"friendly_name"`
	Initials        string         `json:"initials"`
	LastName        string         `json:"last_name"`
	LoginEmail      string         `json:"login_email"`
	MiddleName      string         `json:"middle_name"`
	BillingAddress  Address        `json:"billing_address"`
	ShippingAddress Address        `json:"shipping_address"`

	// Computed
	AccountUserRole *util.AccountRoleKind `json:"account_user_role"`

	// Related entities
	AccountPermissions *AccountPermissionsDetail `json:"account_user_permissions"`
}

type NewUser struct {
	AccountId       string  `json:"account_id"`
	LoginEmail      string  `json:"login_email"`
	FirstName       string  `json:"first_name"`
	LastName        string  `json:"last_name"`
	MiddleName      string  `json:"middle_name"`
	BillingAddress  Address `json:"billing_address"`
	ShippingAddress Address `json:"shipping_address"`
	RoleKind        util.AccountRoleKind
}

type UpdateUser struct {
	LoginEmail      *string  `json:"login_email"`
	FirstName       *string  `json:"first_name"`
	LastName        *string  `json:"last_name"`
	MiddleName      *string  `json:"middle_name"`
	BillingAddress  *Address `json:"billing_address"`
	ShippingAddress *Address `json:"shipping_address"`

	// Convenience
	AccountUserPrimaryRole *util.AccountRoleKind `json:"account_user_role"`
}

// / Appears to be needed by frontend
type DeleteUserResponse struct {
	AccountId string `json:"account_id"`
	Id        string `json:"id"`
}
