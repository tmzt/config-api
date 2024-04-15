package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/tmzt/config-api/util"
)

type NewToken struct {
	AccountId           string `json:"account_id"`
	Email               string `json:"email"`
	Password            string `json:"password"`
	PlatformSignupToken string `json:"signup_token"`
}

type NewTokenInternal struct {
	AccountId util.AccountId `json:"account_id"`
	UserId    util.UserId    `json:"user_id"`
	Email     string         `json:"email"`
}

type TokenDetail struct {
	AccountId    string               `json:"account_id"`
	UserId       string               `json:"user_id"`
	Email        string               `json:"email"`
	Issuer       string               `json:"issuer"`
	IssuedAt     time.Time            `json:"issued_at"`
	ExpiresAt    time.Time            `json:"expires_at"`
	Id           string               `json:"id"`
	Token        string               `json:"token"`
	FriendlyName string               `json:"friendly_name"`
	Initals      string               `json:"initials"`
	PlatformUser bool                 `json:"platform_account_user"`
	Role         util.AccountRoleKind `json:"role"`
}

func (t *TokenDetail) ToCache() *TokenCache {
	return &TokenCache{
		AccountId: t.AccountId,
		UserId:    t.UserId,
		Email:     t.Email,
		IssuedAt:  t.IssuedAt.Unix(),
		ExpiresAt: t.ExpiresAt.Unix(),
		Id:        t.Id,
		Token:     t.Token,
		Role:      t.Role,
	}
}

type TokenCache struct {
	AccountId string               `json:"account_id"`
	UserId    string               `json:"user_id"`
	Email     string               `json:"email"`
	IssuedAt  int64                `json:"issued_at"`
	ExpiresAt int64                `json:"expires_at"`
	Id        string               `json:"id"`
	Token     string               `json:"token"`
	Role      util.AccountRoleKind `json:"role"`
}

func (t *TokenCache) MarshalBinary() ([]byte, error) {
	return json.Marshal(t)
}

func (t *TokenCache) CacheKey() string {
	return fmt.Sprintf("token:%s:%s", t.AccountId, t.Email)
}

func (t *TokenCache) Ttl() time.Duration {
	return time.Duration(t.ExpiresAt-t.IssuedAt) * time.Second
}

type AuthTokenClaims struct {
	AccountId string `json:"aid"`
	UserId    string `json:"uid"`
	Name      string `json:"name,omitempty"`
	Email     string `json:"email,omitempty"`
	Initials  string `json:"initials,omitempty"`
	CommonTokenClaims
}

func (a *AuthTokenClaims) CommonClaims() *CommonTokenClaims {
	return &a.CommonTokenClaims
}
