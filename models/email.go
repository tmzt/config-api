package models

import (
	"time"
)

type EmailORM struct {
	AccountId  string      `gorm:"uniqueIndex:idx_account_email"`
	Account    *AccountORM `gorm:"foreignKey:AccountId;references:Id"`
	CreatedAt  *time.Time
	DeletedAt  *time.Time
	Email      string `gorm:"uniqueIndex:idx_account_email"`
	Id         string `gorm:"primary_key"`
	Subscribed bool
	UpdatedAt  *time.Time
	UserId     string
	User       *UserORM `gorm:"foreignKey:UserId"`
	Verified   bool
}

func (e *EmailORM) TableName() string {
	return "emails"
}

func (e *EmailORM) Detail() *EmailDetail {
	return &EmailDetail{
		Id:         e.Id,
		Email:      e.Email,
		Subscribed: e.Subscribed,
		Verified:   e.Verified,
	}
}

type EmailDetail struct {
	Id         string `json:"id"`
	Email      string `json:"email"`
	Subscribed bool   `json:"subscribed"`
	Verified   bool   `json:"verified"`
}

type NewEmail struct {
	Email      string `json:"email"`
	Subscribed bool   `json:"subscribed"`
	Verified   bool   `json:"verified"`
}

type UpdateEmail struct {
	Subscribed bool `json:"subscribed"`
	Verified   bool `json:"verified"`
}
