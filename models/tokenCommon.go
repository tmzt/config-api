package models

import (
	"log"

	"github.com/golang-jwt/jwt"
	"github.com/tmzt/config-api/util"

	restful "github.com/emicklei/go-restful/v3"
)

type ApiReadPermission string

func RequestApiReadPermissionsAttribute(request *restful.Request, key string) *[]ApiReadPermission {
	v := request.Attribute(key)
	if v != nil {
		if a, ok := v.(*[]ApiReadPermission); ok {
			return a
		}
		if a, ok := v.([]ApiReadPermission); ok {
			return &a
		}
	}
	return nil
}

const (
	ApiReadPermissionCurrentCheckoutTransaction ApiReadPermission = "cur_cktrx"
	ApiReadPermissionAccountPurchases           ApiReadPermission = "acct_purchases"
)

type ApiFullPermissions string

func RequestApiFullPermissionsAttribute(request *restful.Request, key string) *[]ApiFullPermissions {
	v := request.Attribute(key)
	if v != nil {
		if a, ok := v.(*[]ApiFullPermissions); ok {
			return a
		}
		if a, ok := v.([]ApiFullPermissions); ok {
			return &a
		}
	}
	return nil

}

const (
	ApiFullPermissionsAccountAdmin  ApiFullPermissions = "acct_adm"
	ApiFullPermissionsUser          ApiFullPermissions = "user"
	ApiFullPermissionsPlatformAdmin ApiFullPermissions = "plat_adm"
	ApiFullPermissionsPlatformUser  ApiFullPermissions = "plat_user"
)

type CommonTokenClaims struct {
	AccountId             *util.AccountId             `json:"aid,omitempty"`
	UserId                *util.UserId                `json:"uid,omitempty"`
	CheckoutTransactionId *util.CheckoutTransactionId `json:"c_xid,omitempty"`
	ApiReadPermissions    []ApiReadPermission         `json:"api_rd,omitempty"`
	ApiFullPermissions    []ApiFullPermissions        `json:"api_full,omitempty"`
	jwt.StandardClaims
}

func (c *CommonTokenClaims) HasApiReadPermission(permission ApiReadPermission) bool {
	for _, p := range c.ApiReadPermissions {
		if p == permission {
			return true
		}
	}
	return false
}

func (c *CommonTokenClaims) HasApiFullPermission(permission ApiFullPermissions) bool {
	for _, p := range c.ApiFullPermissions {
		if p == permission {
			return true
		}
	}
	return false
}

func (c *CommonTokenClaims) CommonClaims() *CommonTokenClaims {
	return c
}

type TokenWithCommonClaims interface {
	CommonClaims() *CommonTokenClaims
	jwt.Claims
}

func GetCommonClaims(token *jwt.Token) *CommonTokenClaims {
	// claims, ok := token.Claims.(jwt.MapClaims)
	// if !ok {
	// 	log.Println("TokenCommon: GetCommonClaims: nil getting map claims")
	// 	return nil
	// }

	claimsPtr, ok := token.Claims.(*jwt.MapClaims)
	if !ok || claimsPtr == nil {
		log.Println("TokenCommon: GetCommonClaims: nil getting map claims")
		return nil
	}

	claims := *claimsPtr

	common := &CommonTokenClaims{}

	if accountId, ok := claims["aid"].(string); ok {
		common.AccountId = util.AccountIdPtr((accountId))
	}
	if userId, ok := claims["uid"].(string); ok {
		common.UserId = util.UserIdPtr(userId)
	}
	if trxId, ok := claims["c_xid"].(string); ok {
		common.CheckoutTransactionId = util.CheckoutTransactionIdPtr(trxId)
	}
	if apiRead, ok := claims["api_rd"].([]interface{}); ok {
		for _, p := range apiRead {
			common.ApiReadPermissions = append(common.ApiReadPermissions, ApiReadPermission(p.(string)))
		}
	}
	if apiFull, ok := claims["api_full"].([]interface{}); ok {
		for _, p := range apiFull {
			common.ApiFullPermissions = append(common.ApiFullPermissions, ApiFullPermissions(p.(string)))
		}
	}
	if issuer, ok := claims["iss"].(string); ok {
		common.Issuer = issuer
	}
	if issuedAt, ok := claims["iat"].(int64); ok {
		common.IssuedAt = issuedAt
	}
	if expiresAt, ok := claims["exp"].(int64); ok {
		common.ExpiresAt = expiresAt
	}
	if id, ok := claims["jti"].(string); ok {
		common.Id = id
	}
	if subject, ok := claims["sub"].(string); ok {
		common.Subject = subject
	}
	if audience, ok := claims["aud"].(string); ok {
		common.Audience = audience
	}

	return common
}

func RequestHasApiFullPermission(req *restful.Request, permission ApiFullPermissions) bool {
	attr := req.Attribute("claims")
	if attr == nil {
		return false
	}

	token, ok := attr.(TokenWithCommonClaims)
	if !ok {
		return false
	}

	return token.CommonClaims().HasApiFullPermission(permission)
}

func RequestHasApiReadPermission(req *restful.Request, permission ApiReadPermission) bool {
	attr := req.Attribute("claims")
	if attr == nil {
		return false
	}

	token, ok := attr.(TokenWithCommonClaims)
	if !ok {
		return false
	}

	return token.CommonClaims().HasApiReadPermission(permission)
}
