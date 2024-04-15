package util

import "time"

type MutableEmbed struct {
	AccountId AccountId  `json:"account_id" gorm:"type:text;not null"`
	UserId    *UserId    `json:"user_id" gorm:"type:text"`
	CreatedAt time.Time  `json:"created_at" gorm:"type:timestamp with time zone;not null"`
	CreatedBy *UserId    `json:"created_by" gorm:"type:text"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"type:timestamp with time zone;not null"`
	UpdatedBy *UserId    `json:"updated_by" gorm:"type:text"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"type:timestamp with time zone"`
}

type ImmutableEmbed struct {
	Scope     ScopeKind  `json:"scope" gorm:"type:text;not null"`
	AccountId AccountId  `json:"account_id" gorm:"type:text;not null"`
	UserId    *UserId    `json:"user_id" gorm:"type:text"`
	CreatedAt time.Time  `json:"created_at" gorm:"type:timestamp with time zone;not null"`
	CreatedBy UserId     `json:"created_by" gorm:"type:text;not null"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"type:timestamp with time zone"`
}
