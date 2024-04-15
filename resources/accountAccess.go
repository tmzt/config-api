package resources

import (
	"log"
	"net/http"

	restful "github.com/emicklei/go-restful/v3"

	"github.com/tmzt/config-api/util"
)

type AccountAccessValidator struct {
	logger              util.SetRequestLogger
	accountPermissions  *AccountPermissionsResource
	platformPermissions *PlatformPermissionsResource
}

func NewAccountAccessValidator(accountPermissions *AccountPermissionsResource, platformPermissions *PlatformPermissionsResource) *AccountAccessValidator {
	logger := util.NewLogger("AccountAccessValidator", 0)

	return &AccountAccessValidator{
		logger:              logger,
		accountPermissions:  accountPermissions,
		platformPermissions: platformPermissions,
	}
}

// Only called with parentAccountId if a parent account was provided in the URL
// otherwise, parentAccountId must be nil
func (v *AccountAccessValidator) ValidateAccountAccess(userId util.UserId, parentAccountId *util.AccountId, accountId *util.AccountId) bool {

	if parentAccountId != nil {
		// Ensure the account is a child of the parent account
		if !v.accountPermissions.IsAccountParent(*parentAccountId, *accountId) {
			v.logger.Println("Unauthorized account request, not a parent of the account")
			return false
		}
	}

	if accountId == nil {
		v.logger.Println("The accountId is nil, cannot validate access")
		return false
	}

	if *accountId == util.ROOT_ACCOUNT_ID {
		v.logger.Println("Platform account request, validating admin access")
		if !v.platformPermissions.IsPlatformAdmin(userId) {
			v.logger.Println("Unauthorized platform account request")
			return false
		}
		v.logger.Println("Platform account request authorized")
		return true
	}

	if !v.accountPermissions.HasAccountUserAccess(*accountId, userId) {
		v.logger.Println("Unauthorized account request")
		return false
	}

	v.logger.Println("Account request authorized")
	return true
}

func (v *AccountAccessValidator) ValidateAccess(request *restful.Request, response *restful.Response) (*util.UserId, *util.AccountId, *util.AccountId, bool) {
	userId := util.GetRequestUserId(request)
	if userId == nil {
		log.Println("Missing userId")
		response.WriteErrorString(http.StatusUnauthorized, "Unauthorized")
		return nil, nil, nil, false
	}

	parentAccountId := util.GetRequestParentAccountIdPathParam(request)
	accountId := util.GetRequestAccountIdPathParam(request)

	// TODO: Make sure the params match the attributes

	if accountId == nil {
		response.WriteErrorString(http.StatusUnauthorized, "Unauthorized")
		return nil, nil, nil, false
	}

	if parentAccountId != nil {
		if !v.ValidateAccountAccess(*userId, parentAccountId, accountId) {
			v.logger.Printf("Unauthorized account request, user %s has no access to parent account %s\n", *userId, *parentAccountId)
			response.WriteErrorString(http.StatusUnauthorized, "Unauthorized")
			return nil, nil, nil, false
		}

		return userId, parentAccountId, accountId, true
	}

	validated := v.ValidateAccountAccess(*userId, nil, accountId)
	if !validated {
		response.WriteErrorString(http.StatusUnauthorized, "Unauthorized")
		return nil, nil, nil, false
	}

	return userId, nil, accountId, true
}
