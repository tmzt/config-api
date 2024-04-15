package util

import (
	"log"

	restful "github.com/emicklei/go-restful/v3"
)

func RequestBoolAttribute(request *restful.Request, key string) bool {
	v := request.Attribute(key)
	if v != nil {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func RequestStringAttribute(request *restful.Request, key string) string {
	v := request.Attribute(key)
	if v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func RequestAccountIdAttribute(request *restful.Request, key string) *AccountId {
	v := request.Attribute(key)
	if v != nil {
		if a, ok := v.(*AccountId); ok {
			return a
		}
		if a, ok := v.(AccountId); ok {
			return &a
		}
		if s, ok := v.(string); ok {
			if s == ROOT_ACCOUNT_ID {
				return AccountIdPtr(ROOT_ACCOUNT_ID)
			}
			return AccountIdPtr(s)
		}
	}
	return nil
}

func RequestUserIdAttribute(request *restful.Request, key string) *UserId {
	v := request.Attribute(key)
	if v != nil {
		if a, ok := v.(*UserId); ok {
			return a
		}
		if a, ok := v.(UserId); ok {
			return &a
		}
		if s, ok := v.(string); ok {
			return UserIdPtr(s)
		}
	}
	return nil
}

func RequestCheckoutTransactionIdAttribute(request *restful.Request, key string) *CheckoutTransactionId {
	v := request.Attribute(key)
	if v != nil {
		if a, ok := v.(*CheckoutTransactionId); ok {
			return a
		}
		if a, ok := v.(CheckoutTransactionId); ok {
			return &a
		}
		if s, ok := v.(string); ok {
			return CheckoutTransactionIdPtr(s)
		}
	}
	return nil
}

func GetRequestParentAccountIdAttribute(request *restful.Request) *AccountId {
	return RequestAccountIdAttribute(request, "parentAccountId")
}

func GetRequestAccountIdAttribute(request *restful.Request) *AccountId {
	return RequestAccountIdAttribute(request, "accountId")
}

func GetRequestUserIdAttribute(request *restful.Request) *UserId {
	if !IsRequestAuthorized(request) {
		log.Printf("Request is not authorized, cannot get userId attribute")
		return nil
	}

	return RequestUserIdAttribute(request, "userId")
}
