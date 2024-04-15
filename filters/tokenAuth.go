package filters

import (
	"fmt"
	"net/http"
	"strings"

	restful "github.com/emicklei/go-restful/v3"

	"github.com/tmzt/config-api/models"
	"github.com/tmzt/config-api/resources"
	"github.com/tmzt/config-api/services"
	"github.com/tmzt/config-api/util"
)

type TokenAuthorizationFilter struct {
	logger              util.SetRequestLogger
	platformPermissions *resources.PlatformPermissionsResource
	accountPermissions  *resources.AccountPermissionsResource
	jwtService          *services.JwtService
}

func NewTokenAuthorizationFilter(platformPermissions *resources.PlatformPermissionsResource, accountUserPermissions *resources.AccountPermissionsResource, jwtService *services.JwtService) *TokenAuthorizationFilter {
	logger := util.NewLogger("TokenAuthorizationFilter", 0)

	return &TokenAuthorizationFilter{
		logger,
		platformPermissions,
		accountUserPermissions,
		jwtService,
	}
}

func (f TokenAuthorizationFilter) Filter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	f.logger.SetRequest(req)

	if util.RequestBoolAttribute(req, "bypassAuth") {
		f.logger.Println("Bypassing authorization")
		chain.ProcessFilter(req, resp)
		return
	}

	authValue := req.Request.Header.Get("Authorization")
	if authValue == "" {
		f.logger.Printf("No Authorization header")
		resp.WriteHeader(http.StatusUnauthorized)
		return
	}

	f.logger.Printf("Authorization: %s", authValue)
	parts := strings.Split(authValue, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		f.logger.Printf("Invalid Authorization header")
		resp.WriteHeader(http.StatusUnauthorized)
		return
	}

	token := parts[1]

	// Decode the token
	// TODO: Use jwtService and validate the token
	// tokenObj, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
	// 	return f.publicKey, nil
	// })
	_, claims, common, err := f.jwtService.ParseValidTokenWithCommonClaims(token)

	if err != nil {
		f.logger.Printf("Error parsing token: %v", err)
		resp.WriteHeader(http.StatusUnauthorized)
		return
	}

	req.SetAttribute("token", token)

	// claims := models.GetCommonClaims(tokenObj)
	// if claims == nil || !tokenObj.Valid {
	// 	f.logger.Printf("Invalid token")
	// 	resp.WriteHeader(http.StatusUnauthorized)
	// 	return
	// }

	f.logger.Printf("Claims: %v", claims)
	req.SetAttribute("claims", claims)

	// common := claims.CommonClaims()

	sub := common.Subject

	if common.AccountId == nil {
		f.logger.Printf("No account ID in token")
		resp.WriteHeader(http.StatusUnauthorized)
		chain.ProcessFilter(req, resp)
		return
	}
	accountId := *common.AccountId

	f.logger.Printf("Sub: %s\n", sub)
	f.logger.Printf("AccountId: %v\n", accountId)
	if common.UserId != nil {
		f.logger.Printf("UserId: %v\n", *common.UserId)
	} else {
		f.logger.Printf("UserId: nil\n")
	}
	f.logger.Printf("CurrentTrx: %v\n", common.CheckoutTransactionId)

	req.SetAttribute("actualAccountId", accountId)
	req.SetAttribute("accountId", common.UserId)

	if f.handleSpecialToken(req, resp, common, chain) {
		return
	}

	userId := *common.UserId

	req.SetAttribute("actualUserId", userId)
	req.SetAttribute("userId", userId)

	f.logger.Printf("**** UserId: %s\n", userId)

	actualAccountId := accountId
	actualUserId := userId

	// Optional claims
	currentTrx := common.CheckoutTransactionId

	// Set some attributes

	var parentAccountId *util.AccountId
	if v := req.PathParameter("parentAccountId"); v != "" {
		parentAccountId = util.AccountIdPtr(v)
		f.logger.Printf("ParentAccountId: %s\n", *parentAccountId)
		req.SetAttribute("actualParentAccountId", *parentAccountId)
		req.SetAttribute("parentAccountId", *parentAccountId)
	}

	req.SetAttribute("actualAccountId", actualAccountId)
	req.SetAttribute("actualUserId", actualUserId)
	req.SetAttribute("isImpersonating", false)

	req.SetAttribute("accountId", accountId)
	req.SetAttribute("userId", userId)
	req.SetAttribute("currentTrx", currentTrx)

	req.SetAttribute("apiFull", common.ApiFullPermissions)
	req.SetAttribute("apiRead", common.ApiReadPermissions)

	chain.ProcessFilter(req, resp)
}

func (f TokenAuthorizationFilter) handleSpecialToken(req *restful.Request, resp *restful.Response, common *models.CommonTokenClaims, chain *restful.FilterChain) bool {
	accountId := *common.AccountId
	sub := common.Subject

	isCheckoutToken := strings.HasPrefix(sub, fmt.Sprintf("account:%s:checkout_token:", accountId)) && common.CheckoutTransactionId != nil
	isDemoToken := sub == fmt.Sprintf("appsub:demo:token:account_id:%s", accountId)

	f.logger.Printf("IsCheckoutToken: %v\n", isCheckoutToken)
	f.logger.Printf("IsDemoToken: %v\n", isDemoToken)

	isSpecialToken := isCheckoutToken || isDemoToken

	f.logger.Printf("IsSpecialToken: %v\n", isSpecialToken)

	// User id is optional for checkout tokens
	if common.UserId == nil && !isSpecialToken {
		f.logger.Printf("No user ID in token (and not a valid special token)")
		resp.WriteHeader(http.StatusUnauthorized)
		return true
	}

	if isSpecialToken {
		req.SetAttribute("isSpecialToken", true)
		req.SetAttribute("specialTokenAccountId", accountId)

		// TODO(SECURITY): We should distinguish the token used to create a special token
		// from the account used to access the API
		// This may mean creating a temporary account and/or user
		// for the demo or checkout session.
		// Since we allow anyone to create a checkout token, this is
		// a potential security issue if the correct checks are not in place
		// on the actual API calls.

		if common.ApiFullPermissions != nil {
			req.SetAttribute("specialTokenApiFullPermissions", common.ApiFullPermissions)
		}
		if common.ApiReadPermissions != nil {
			req.SetAttribute("specialTokenApiReadPermissions", common.ApiReadPermissions)
		}
	}

	if isCheckoutToken {
		req.SetAttribute("checkoutToken", true)
		req.SetAttribute("checkoutTokenAccountId", accountId)
		req.SetAttribute("checkoutTokenCurrentTransactionId", common.CheckoutTransactionId)
		req.SetAttribute("accountId", accountId)
		f.logger.Printf("Passing to next filter for checkout token")
		chain.ProcessFilter(req, resp)
		return true
	} else if isDemoToken {
		req.SetAttribute("demoToken", true)
		req.SetAttribute("demoTokenAccountId", accountId)
		req.SetAttribute("accountId", accountId)
		f.logger.Printf("Passing to next filter for demo token")
		chain.ProcessFilter(req, resp)
		return true
	}
	// Handle other special tokens here

	return false
}
