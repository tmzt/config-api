package util

import (
	"log"

	"github.com/emicklei/go-restful/v3"
)

func IsRequestAuthorized(request *restful.Request) bool {
	return RequestBoolAttribute(request, "authorized")
}

func GetRequestParentAccountIdPathParam(req *restful.Request) *AccountId {
	if v := req.PathParameter("parentAccountId"); v != "" {
		if v == "platform" {
			v = ROOT_ACCOUNT_ID
		}
		log.Printf("Using PathParameter parentAccountId: %s\n", v)
		return AccountIdPtr(v)
	}
	return nil
}

func GetRequestAccountIdPathParam(req *restful.Request) *AccountId {
	if v := req.PathParameter("accountId"); v != "" {
		if v == "platform" {
			v = ROOT_ACCOUNT_ID
		}
		log.Printf("Using PathParameter accountId: %s\n", v)
		return AccountIdPtr(v)
	}
	return nil
}

func GetRequestAccountId(req *restful.Request) *AccountId {
	if !IsRequestAuthorized(req) {
		log.Printf("Request is not authorized, cannot get accountId")
		return nil
	}

	if v := req.PathParameter("accountId"); v != "" {
		log.Printf("Using PathParameter accountId: %s\n", v)
		return AccountIdPtr(v)
	}

	return GetRequestAccountIdAttribute(req)
}

func GetValidatedRequestParentAccountId(req *restful.Request) *AccountId {
	param := req.PathParameter("parentAccountId")
	if param == "" {
		log.Printf("PathParameter parentAccountId is empty, invalid account request")
		return nil
	}

	attr := GetRequestParentAccountIdAttribute(req)
	if attr == nil {
		log.Printf("ParentAccountId attribute is nil, cannot validate")
		return nil
	}

	if param != string(*attr) {
		log.Printf("PathParameter parentAccountId %s does not match attribute %s", param, string(*attr))
		return nil
	}

	return attr
}

func GetValidatedRequestAccountId(req *restful.Request) *AccountId {
	accountParam := req.PathParameter("accountId")
	if accountParam == "" {
		log.Printf("PathParameter accountId is empty, invalid account request")
		return nil
	}

	accountIdAttr := GetRequestAccountIdAttribute(req)
	if accountIdAttr == nil {
		log.Printf("AccountId attribute is nil, cannot validate")
		return nil
	}

	accountIdParam := req.PathParameter("accountId")
	if accountIdParam != string(*accountIdAttr) {
		log.Printf("PathParameter accountId %s does not match attribute %s", accountIdParam, string(*accountIdAttr))
		return nil
	}

	return accountIdAttr
}

func GetValidatedRequestUserId(req *restful.Request) *UserId {
	if !IsRequestAuthorized(req) {
		log.Printf("Request is not authorized, cannot get userId attribute")
		return nil
	}

	userIdAttr := GetRequestUserIdAttribute(req)
	if userIdAttr == nil {
		log.Printf("UserId attribute is nil, cannot validate")
		return nil
	}

	userIdParam := req.PathParameter("userId")
	if userIdParam != string(*userIdAttr) {
		log.Printf("PathParameter userId %s does not match attribute %s", userIdParam, string(*userIdAttr))
		return nil
	}

	return userIdAttr
}

func GetRequestUserId(request *restful.Request) *UserId {
	authorized := request.Attribute("authorized")
	if authorized == nil {
		return nil
	}

	if v := request.PathParameter("userId"); v != "" {
		log.Printf("Using PathParameter userId: %s\n", v)
		res := UserId(v)
		return &res
	}

	userIdAttr := request.Attribute("userId")
	if userIdAttr != nil {
		log.Printf("Using Attribute userId: %v\n", userIdAttr)
		if v, ok := userIdAttr.(string); ok {
			res := UserId(v)
			return &res
		}
		if v, ok := userIdAttr.(UserId); ok {
			return &v
		}
	}

	return nil
}

func GetRequestCustomerUserId(request *restful.Request) *CustomerUserId {
	authorized := request.Attribute("authorized")
	if authorized == nil {
		return nil
	}

	// TODO: Validate customerAccount from attributes

	if v := request.PathParameter("customerUserId"); v != "" {
		log.Printf("Using PathParameter customerUserId: %s\n", v)
		res := CustomerUserId(v)
		return &res
	}

	return nil
}

func GetRequestCheckoutTokenIdPathParam(req *restful.Request) *CheckoutTokenId {
	if v := req.PathParameter("checkoutTokenId"); v != "" {
		log.Printf("Using PathParameter checkoutTokenId: %s\n", v)
		res := CheckoutTokenId(v)
		return &res
	}
	return nil
}

// Returns the scope determined by whether account_id and user_id are path parameters. Always returns both
// account_id and user_id, even if they are not path parameters. Returns ScopeKindInvaild in
// other cases.
func GetRequestScopeAndIds(request *restful.Request) (ScopeKind, AccountId, UserId) {
	accountId := GetValidatedRequestAccountId(request)
	if accountId == nil {
		return ScopeKindInvalid, "", ""
	}

	userId := GetRequestUserIdAttribute(request)
	if userId == nil {
		return ScopeKindInvalid, "", ""
	}

	userIdParam := request.PathParameter("userId")
	if userIdParam != "" {
		validatedUserId := GetValidatedRequestUserId(request)
		if validatedUserId != nil {
			return ScopeKindUser, *accountId, *validatedUserId
		}
	}

	return ScopeKindAccount, *accountId, *userId
}
