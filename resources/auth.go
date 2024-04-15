package resources

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/tmzt/config-api/models"
	"github.com/tmzt/config-api/services"
	"github.com/tmzt/config-api/util"
	"gorm.io/gorm"

	redis "github.com/go-redis/redis/v8"
)

type AuthResource struct {
	rdb                 *redis.Client
	db                  *gorm.DB
	userResource        *UserResource
	platformPermissions *PlatformPermissionsResource
	accountPermissions  *AccountPermissionsResource
	jwtService          *services.JwtService
}

func NewAuthResource(
	rdb *redis.Client,
	db *gorm.DB,
	userResource *UserResource,
	platformPermissions *PlatformPermissionsResource,
	accoutPermissions *AccountPermissionsResource,
	jwtService *services.JwtService,
) *AuthResource {
	return &AuthResource{rdb, db, userResource, platformPermissions, accoutPermissions, jwtService}
}

func (r *AuthResource) CreateTokenInternal(newToken *models.NewTokenInternal) (*models.TokenDetail, error) {

	// TODO: do we need to check accountId here?

	// Get user
	user, err := r.userResource.GetUserByEmail(newToken.Email)
	if err != nil {
		log.Printf("CreateTokenInternal: Failed to get user for email %s: %v\n", newToken.Email, err)
		return nil, err
	}

	id := util.NewUUID()
	userId := user.Id
	accountId := user.AccountId

	aud := util.MustGetRootTokenAudience()
	sub := fmt.Sprintf("account:%s:user:%s", accountId, userId)

	ts := time.Now()

	stdClaims := r.jwtService.CreateStandardClaims(ts, id, nil, aud, sub)

	claims := models.AuthTokenClaims{
		AccountId: accountId,
		UserId:    userId,
		Name:      user.FriendlyName,
		Email:     user.LoginEmail,
		Initials:  user.Initials,
	}

	claims.StandardClaims = stdClaims

	tokenString, err := r.jwtService.CreateSignedToken(claims)
	if err != nil {
		log.Printf("Failed to sign token: %v", err)
		return nil, err
	}

	isPlatformUser := r.platformPermissions.IsPlatformUser(util.UserId(userId))

	tokenDetail := &models.TokenDetail{
		AccountId:    accountId,
		UserId:       userId,
		Id:           id,
		Token:        tokenString,
		Initals:      user.Initials,
		Issuer:       stdClaims.Issuer,
		IssuedAt:     time.Unix(stdClaims.IssuedAt, 0),
		Email:        user.LoginEmail,
		ExpiresAt:    time.Unix(stdClaims.ExpiresAt, 0),
		FriendlyName: user.FriendlyName,
		PlatformUser: isPlatformUser,
	}

	tokenCache := tokenDetail.ToCache()

	// Cache in Redis
	err = r.rdb.Set(context.Background(), tokenCache.CacheKey(), tokenCache, tokenCache.Ttl()).Err()
	if err != nil {
		log.Printf("Failed to cache token: %v", err)
		return nil, err
	}

	return tokenDetail, nil
}

func (r *AuthResource) CreateToken(newToken *models.NewToken) (*models.TokenDetail, error) {

	// TODO: Research strong password validation mechanism

	user, err := r.userResource.GetUserByEmail(newToken.Email)
	if err != nil {
		log.Printf("CreateToken: Failed to get user for email %s: %v", newToken.Email, err)
		return nil, err
	} else if user == nil {
		return nil, nil
	}

	accountId := util.AccountId(user.AccountId)
	userId := util.UserId(user.Id)

	newTokenInternal := &models.NewTokenInternal{
		AccountId: accountId,
		UserId:    userId,
		Email:     newToken.Email,
	}

	return r.CreateTokenInternal(newTokenInternal)
}
