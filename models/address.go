package models

import (
	"time"
)

type AddressKind string

const (
	BillingAddress  AddressKind = "billing"
	ShippingAddress AddressKind = "shipping"
)

type AddressORM struct {
	AccountId  string
	Account    *AccountORM `gorm:"foreignKey:AccountId;references:Id"`
	City       string
	Country    string
	CreatedAt  *time.Time
	DeletedAt  *time.Time
	Id         string `gorm:"primary_key"`
	Line1      string
	Line2      string
	PostalCode string
	State      string
	UpdatedAt  *time.Time
}

func (a *AddressORM) TableName() string {
	return "addresses"
}

type Address struct {
	City       string `json:"city"`
	Country    string `json:"country"`
	Line1      string `json:"line1"`
	Line2      string `json:"line2"`
	PostalCode string `json:"postal_code"`
	State      string `json:"state"`
}

type UserAddressORM struct {
	UserId      string      `gorm:"primary_key;column:user_id"`
	AddressId   string      `gorm:"primary_key;column:address_id"`
	AddressKind AddressKind `gorm:"primary_key;column:address_kind"`
}

func (u *UserAddressORM) TableName() string {
	return "user_addresses"
}
