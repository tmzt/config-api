package filters

import (
	"crypto/ecdsa"
	"net/http"
	"strings"

	restful "github.com/emicklei/go-restful/v3"

	"github.com/tmzt/config-api/models"
	"github.com/tmzt/config-api/resources"
	"github.com/tmzt/config-api/util"
)

type AuthorizationFilter struct {
	logger              util.SetRequestLogger
	platformPermissions *resources.PlatformPermissionsResource
	accountPermissions  *resources.AccountPermissionsResource
	publicKey           *ecdsa.PublicKey
}

func NewAuthorizationFilter(platformPermissions *resources.PlatformPermissionsResource, accountUserPermissions *resources.AccountPermissionsResource) *AuthorizationFilter {
	publicKey := util.MustGetRootTokenPublicKey()

	logger := util.NewLogger("AuthorizationFilter", 0)

	return &AuthorizationFilter{
		logger,
		platformPermissions,
		accountUserPermissions,
		publicKey,
	}
}

func (f AuthorizationFilter) Filter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	f.logger.SetRequest(req)

	if util.RequestBoolAttribute(req, "bypassAuth") {
		f.logger.Println("Bypassing authorization")
		chain.ProcessFilter(req, resp)
		return
	}

	actualUserId := util.RequestUserIdAttribute(req, "actualUserId")
	actualAccountId := util.RequestAccountIdAttribute(req, "actualAccountId")

	isSpecialToken := util.RequestBoolAttribute(req, "isSpecialToken")

	if actualUserId == nil && !isSpecialToken {
		f.logger.Println("No actualUserId (and not a special token)")
		resp.WriteHeader(http.StatusUnauthorized)
		return
	} else if actualAccountId == nil {
		f.logger.Println("No actualAccountId")
		resp.WriteHeader(http.StatusUnauthorized)
		return
	}

	if isSpecialToken {
		specialTokenAccountId := util.RequestAccountIdAttribute(req, "specialTokenAccountId")
		if specialTokenAccountId == nil {
			f.logger.Println("No specialTokenAccountId (using a special token)")
			resp.WriteHeader(http.StatusUnauthorized)
			return
		}

		req.SetAttribute("actualAccountId", *specialTokenAccountId)
		req.SetAttribute("accountId", *specialTokenAccountId)

		// Skip the rest of this filter, since we have a special token
		// the remaining checks will not pass.
		// TODO(SECURITY): Improve the security of special tokens

		req.SetAttribute("authorized", true)

		specialTokenApiReadPermissions := models.RequestApiReadPermissionsAttribute(req, "specialTokenApiReadPermissions")
		if specialTokenApiReadPermissions != nil {
			req.SetAttribute("apiReadPermissions", *specialTokenApiReadPermissions)
		}

		specialTokenApiFullPermissions := models.RequestApiFullPermissionsAttribute(req, "specialTokenApiFullPermissions")
		if specialTokenApiFullPermissions != nil {
			req.SetAttribute("apiFullPermissions", *specialTokenApiFullPermissions)
		}

		chain.ProcessFilter(req, resp)
		return
	}

	f.logger.Printf("**** Actual UserId: %s\n", actualUserId)

	// Optional claims
	// currentTrx := common.CheckoutTransactionId
	// currentTrx := util.RequestCheckoutTransactionIdAttribute(req, "currentTrx")

	// Set some attributes

	var parentAccountId *util.AccountId
	if v := req.PathParameter("parentAccountId"); v != "" {
		parentAccountId = util.AccountIdPtr(v)
		f.logger.Printf("ParentAccountId: %s\n", *parentAccountId)
		req.SetAttribute("actualParentAccountId", *parentAccountId)
		req.SetAttribute("parentAccountId", *parentAccountId)
	}

	// If the user is a platform admin, they have access
	isPlatformAdmin := f.platformPermissions.IsPlatformAdmin(*actualUserId)
	f.logger.Printf("AuthorizationFilter: IsPlatformAdmin: %v\n", isPlatformAdmin)

	userId := *actualUserId
	accountId := *actualAccountId

	req.SetAttribute("userId", userId)
	req.SetAttribute("accountId", accountId)

	if isPlatformAdmin {
		req.SetAttribute("isActualPlatformAdmin", true)

		// Handle impersonation, which will replace the non-Actual attributes
		impersonatedAccountId, impersonatedUserId, ret := f.handleImpersonation(req, resp, chain)
		if ret {
			return
		}

		if impersonatedAccountId != nil {
			req.SetAttribute("accountId", *impersonatedAccountId)
			accountId = *impersonatedAccountId
		}
		if impersonatedUserId != nil {
			req.SetAttribute("userId", *impersonatedUserId)
			userId = *impersonatedUserId
		}

		f.logger.Printf("Root account admins have super-admin access")
		req.SetAttribute("isPlatformAdmin", true)
		req.SetAttribute("authorized", true)
		chain.ProcessFilter(req, resp)
		return
	}

	f.logger.Printf("Checking account user permissions for user %s to account %s", userId, accountId)
	hasAccountAccess := f.accountPermissions.CheckAccessToAccount(parentAccountId, accountId, req, resp)
	if !hasAccountAccess {
		f.logger.Printf("User %s does not have access to account %s", userId, accountId)
		return
	}

	// Just requires a signed token and passing the previous tests
	req.SetAttribute("authorized", true)

	f.logger.Printf("User %s has access to account %s\n", userId, accountId)
	f.logger.Println("finished")
	chain.ProcessFilter(req, resp)
}

func (f AuthorizationFilter) handleImpersonation(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) (*util.AccountId, *util.UserId, bool) {
	impersonate := req.Request.Header.Get("X-Impersonate-User")
	if impersonate != "" {
		parts := strings.Split(impersonate, ":")
		if len(parts) == 2 {
			impersonateAccountId := util.AccountId(parts[0])
			impersonateUserId := util.UserId(parts[1])

			f.logger.Printf("IMPERSONATE: accountId: %s", impersonateAccountId)
			f.logger.Printf("IMPERSONATE: userId: %s", impersonateUserId)

			req.SetAttribute("authorized", true)

			req.SetAttribute("impersonateAccountId", impersonateAccountId)
			req.SetAttribute("impersonateUserId", impersonateUserId)
			req.SetAttribute("accountId", impersonateAccountId)
			req.SetAttribute("userId", impersonateUserId)
			req.SetAttribute("isImpersonating", true)

			// Pass to the next filter
			chain.ProcessFilter(req, resp)
			return &impersonateAccountId, &impersonateUserId, true
		} else {
			resp.WriteErrorString(http.StatusBadRequest, "Invalid X-Impersonate-User header")
			return nil, nil, true
		}
	}

	return nil, nil, false
}
