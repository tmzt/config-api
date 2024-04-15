package resources

import (
	"context"
	"fmt"
	"log"

	redis "github.com/go-redis/redis/v8"

	"github.com/tmzt/config-api/models"
	"github.com/tmzt/config-api/util"
	"gorm.io/gorm"
)

type PlatformPermissionsResource struct {
	logger *log.Logger
	rdb    *redis.Client
	db     *gorm.DB
}

func NewPlatformPermissionsResource(rdb *redis.Client, db *gorm.DB) *PlatformPermissionsResource {
	logger := log.New(log.Writer(), "RootUserPermissionsResource: ", log.LstdFlags|log.Lshortfile)
	return &PlatformPermissionsResource{logger, rdb, db}
}

func (r *PlatformPermissionsResource) GetRootUserPermissions(userId util.UserId) (*models.PlatformPermissionsDetail, error) {
	ormRootPermissions := &models.PlatformPermissionsORM{}

	// Check for cached version
	cacheObj := &models.PlatformPermissionsCache{}
	cacheKey := fmt.Sprintf("platform_user_permissions:%s", userId)
	if err := util.GetCache(context.Background(), r.rdb, cacheKey, *cacheObj); err == nil {
		log.Println("Returning cached platform user permissions detail")
		return cacheObj.Detail, nil
	}

	res := r.db.
		Where("user_id = ?", string(userId)).
		Find(ormRootPermissions)

	if res.Error != nil {
		log.Println("Returning empty platform user permissions detail")
		return &models.PlatformPermissionsDetail{}, nil
	}

	detail := ormRootPermissions.Detail()

	// Cache
	r.rdb.Set(context.Background(), "platform_user_permissions:"+string(userId), detail, 0)

	return detail, nil
}

func (r *PlatformPermissionsResource) UpsertRootUserPermissions(userId util.UserId, updatePermissions *models.PlatformPermissionsDetail) error {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		ormRootUserPermissions := models.PlatformPermissionsORM{}

		accountId := util.ROOT_ACCOUNT_ID

		res := r.db.
			Where("account_id = ? AND user_id = ?", accountId, string(userId)).
			// Only used if record does not exist
			Attrs(&models.PlatformPermissionsORM{
				Id: util.NewUUID(),
			}).
			// Always updated
			Assign(&models.PlatformPermissionsORM{
				AccountId:    string(accountId),
				UserId:       string(userId),
				RootUser:     updatePermissions.PlatformUser,
				InternalUser: updatePermissions.InternalUser,
			}).
			FirstOrCreate(&ormRootUserPermissions)

		if res.Error != nil {
			return res.Error
		}

		// Cache the result
		util.SetCache(r.rdb, ormRootUserPermissions.Cache())

		return nil
	})

	if err != nil {
		log.Println("Error updating platform user permissions: ", err)
		return err
	}

	return nil
}
func (r *PlatformPermissionsResource) IsPlatformAdmin(userId util.UserId) bool {
	detail, err := r.GetRootUserPermissions(userId)
	if err != nil {
		log.Println("Error getting platform user permissions: ", err)
		return false
	}

	return detail.PlatformAdmin
}

func (r *PlatformPermissionsResource) IsPlatformUser(userId util.UserId) bool {
	detail, err := r.GetRootUserPermissions(userId)
	if err != nil {
		log.Println("Error getting platform user permissions: ", err)
		return false
	}

	return detail.PlatformUser
}

func (r *PlatformPermissionsResource) IsInternalUser(userId util.UserId) bool {
	detail, err := r.GetRootUserPermissions(userId)

	if err != nil {
		log.Println("Error getting platform user permissions: ", err)
		return false
	}

	return detail.InternalUser
}
