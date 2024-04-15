package resources

import (
	"context"
	"fmt"
	"log"
	"net/http"

	restful "github.com/emicklei/go-restful/v3"
	redis "github.com/go-redis/redis/v8"

	"github.com/tmzt/config-api/models"
	"github.com/tmzt/config-api/util"
	"gorm.io/gorm"
)

type AccountPermissionsResource struct {
	logger *log.Logger
	rdb    *redis.Client
	db     *gorm.DB
}

func NewAccountPermissionsResource(rdb *redis.Client, db *gorm.DB) *AccountPermissionsResource {
	logger := log.New(log.Writer(), "AccountPermissionsResource: ", log.LstdFlags|log.Lshortfile)
	return &AccountPermissionsResource{logger, rdb, db}
}

func (r *AccountPermissionsResource) GetAccountPermissions(accountId util.AccountId, userId util.UserId) (*models.AccountPermissionsDetail, error) {
	// Check for cached version
	cacheObj := &models.AccountPermissionsCache{}
	cacheKey := fmt.Sprintf("account_user_permissions:%s:%s", accountId, userId)
	if err := util.GetCache(context.Background(), r.rdb, cacheKey, *cacheObj); err == nil {
		log.Println("Returning cached account user permissions detail")
		return cacheObj.Detail, nil
	}

	ormAccountPermissions := &models.AccountPermissionsORM{}
	err := r.db.First(ormAccountPermissions, "account_id = ? AND user_id = ?", string(accountId), string(userId)).Error
	if err != nil {
		log.Println("Returning empty account user permissions detail")
		return &models.AccountPermissionsDetail{}, nil
	}

	// Cache the result
	util.SetCache(r.rdb, ormAccountPermissions.Cache())

	detail := ormAccountPermissions.Detail()

	return detail, nil
}

func setWithRole(detail *models.AccountPermissionsDetail, role util.AccountRoleKind) {
	switch role {
	case util.AccountRoleKindAdmin:
		detail.AccountAdmin = true
	case util.AccountRoleKindBuyer:
		detail.AccountBuyer = true
	case util.AccountRoleKindOwner:
		detail.AccountOwner = true
	case util.AccountRoleKindUser:
		detail.AccountUser = true
	}
}

func (r *AccountPermissionsResource) UpsertAccountPermissions(tx *gorm.DB, accountId util.AccountId, userId util.UserId, updatePermissions *models.AccountPermissionsDetail) error {
	err := util.WithTransaction(r.db, tx, func(tx *gorm.DB) error {
		ormAccountPermissions := models.AccountPermissionsORM{}
		res := tx.
			Where("account_id = ? AND user_id = ?", string(accountId), string(userId)).
			// Only used if record does not exist
			Attrs(&models.AccountPermissionsORM{
				Id: util.NewUUID(),
			}).
			// Always updated
			Assign(&models.AccountPermissionsORM{
				AccountId:              string(accountId),
				UserId:                 string(userId),
				AccountAdmin:           updatePermissions.AccountAdmin,
				AccountBuyer:           updatePermissions.AccountBuyer,
				AccountOwner:           updatePermissions.AccountOwner,
				AccountUser:            updatePermissions.AccountUser,
				AccountEmailListReader: updatePermissions.AccountEmailListReader,
			}).
			FirstOrCreate(&ormAccountPermissions)

		if res.Error != nil {
			return res.Error
		}

		// Cache
		util.SetCache(r.rdb, ormAccountPermissions.Cache())

		return nil
	})

	if err != nil {
		log.Println("Error updating account user permissions: ", err)
		return err
	}

	return nil
}

func (r *AccountPermissionsResource) UpsertAccountPermissionsForRole(tx *gorm.DB, accountId util.AccountId, userId util.UserId, role util.AccountRoleKind) error {
	assign := &models.AccountPermissionsDetail{}

	if role == "" {
		role = util.AccountRoleKindUser
	}

	setWithRole(assign, role)

	return r.UpsertAccountPermissions(tx, accountId, userId, assign)
}

// Returns true if the account is an immediate child of the parent account, false otherwise
func (r *AccountPermissionsResource) IsAccountParent(parentAccountId util.AccountId, accountId util.AccountId) bool {
	ormAccount := &models.AccountORM{}
	err := r.db.First(ormAccount, "parent_account_id = ? AND id = ?", string(parentAccountId), string(accountId)).Error
	if err != nil {
		log.Println("Error getting account: ", err)
		return false
	}

	// Root account is parent of all accounts that have NULL parent
	if ormAccount.ParentAccountId == nil {
		return string(parentAccountId) == util.ROOT_ACCOUNT_ID
	}

	return *ormAccount.ParentAccountId == string(parentAccountId)
}

func (r *AccountPermissionsResource) HasAccountUserAccess(accountId util.AccountId, userId util.UserId) bool {
	detail, err := r.GetAccountPermissions(accountId, userId)
	if err != nil {
		log.Println("Error getting account user permissions: ", err)
		return false
	}

	userAccess := detail.AccountOwner || detail.AccountAdmin || detail.AccountBuyer || detail.AccountUser

	return userAccess
}

func (r *AccountPermissionsResource) HasAccountAdminAccess(accountId util.AccountId, userId util.UserId) bool {

	detail, err := r.GetAccountPermissions(accountId, userId)
	if err != nil {
		log.Println("Error getting account user permissions: ", err)
		return false
	}

	// TBD: should buyer only configure admin accounts
	adminAccess := detail.AccountAdmin || detail.AccountOwner || detail.AccountBuyer

	return adminAccess
}

func (r *AccountPermissionsResource) HasAccountAccess(userId util.UserId, parentAccountId *util.AccountId, accountId util.AccountId) bool {
	// TODO: Move this to the platformPermissions resource
	// if accountId == util.ROOT_ACCOUNT_ID {
	// 	// For now, only platform admins will have access
	// 	// TBD: should we allow platform users to have access?

	// 	if !r.platformPermissions.IsPlatformAdmin(userId) {
	// 		r.logger.Printf("User %s does not have platform admin permission, denying access to account %s\n", userId, accountId)
	// 		return false
	// 	}

	// 	return true
	// }

	if accountId == util.ROOT_ACCOUNT_ID {
		r.logger.Println("Use the platformPermissions resource to check access to the platform account")
		return false
	}

	// If we are accessing an account, we need to check the account user permissions
	r.logger.Printf("Checking account user permissions for user %s to account %s\n", userId, accountId)

	if parentAccountId != nil && !r.IsAccountParent(*parentAccountId, accountId) {
		r.logger.Printf("Account %s is not a child of account %s", accountId, *parentAccountId)
		return false
	}

	if !r.HasAccountUserAccess(accountId, userId) {
		r.logger.Printf("User %s does not have account permission (owner, admin, buyer, or user) for account %s", userId, accountId)
		return false
	}

	// accountUserPermissions, err := accountPermissions.GetAccountPermissions(accountId, userId)
	// if err != nil {
	// 	r.logger.Printf("Error getting account user permissions while accessing account: %v", err)
	// 	resp.WriteHeader(http.StatusUnauthorized)
	// 	return
	// }

	// hasAccountPermission := accountUserPermissions.AccountOwner || accountUserPermissions.AccountAdmin || accountUserPermissions.AccountUser
	// if !hasAccountPermission {
	// 	r.logger.Printf("User does not have account permission (owner, admin, or user) for account %s", accountId)
	// 	resp.WriteHeader(http.StatusUnauthorized)
	// 	return
	// }

	return true
}

func (r *AccountPermissionsResource) CheckAccessToAccount(parentAccountId *util.AccountId, accountId util.AccountId, req *restful.Request, resp *restful.Response) bool {
	// Use this one because it does not check authorized attribute
	userId := util.RequestUserIdAttribute(req, "userId")
	if userId == nil {
		r.logger.Printf("No user id found in request")
		resp.WriteErrorString(http.StatusUnauthorized, "Unauthorized")
		return false
	}

	if !r.HasAccountAccess(*userId, parentAccountId, accountId) {
		r.logger.Printf("User %s does not have access to account %s", *userId, accountId)
		resp.WriteErrorString(http.StatusUnauthorized, "Unauthorized")
		return false
	}

	// Just requires a signed token and passing the previous tests
	req.SetAttribute("authorized", true)

	r.logger.Printf("User %s has access to account %s\n", *userId, accountId)

	return true
}
