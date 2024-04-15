package resources

import (
	"errors"
	"log"
	"time"

	"github.com/tmzt/config-api/models"
	"github.com/tmzt/config-api/util"
	"gorm.io/gorm"
)

type UserResource struct {
	logger             util.SetRequestLogger
	db                 *gorm.DB
	accountPermissions *AccountPermissionsResource
}

func NewUserResource(db *gorm.DB, accountPermissions *AccountPermissionsResource) *UserResource {
	logger := util.NewLogger("UserResource", 0)

	return &UserResource{
		logger:             logger,
		db:                 db,
		accountPermissions: accountPermissions,
	}
}

// CreateUser creates a new user
func (u *UserResource) CreateUser(accountId util.AccountId, newUser *models.NewUser) (*models.UserDetail, error) {

	userId := util.UserId(util.NewUUID())

	log.Println("Creating user with account_id: ", accountId)
	log.Println("Creating user with user_id: ", userId)
	log.Println("Creating user with email: ", newUser.LoginEmail)

	emailId := util.EmailId(util.NewUUID())

	ormEmail := models.EmailORM{
		AccountId: string(accountId),
		Id:        string(emailId),
		Email:     newUser.LoginEmail,
		UserId:    string(userId),
	}

	ormUser := models.UserORM{
		AccountId:    string(accountId),
		Id:           string(userId),
		LoginEmail:   newUser.LoginEmail,
		LoginEmailId: string(emailId),
		FirstName:    newUser.FirstName,
		LastName:     newUser.LastName,
		MiddleName:   newUser.MiddleName,
	}

	err := u.db.Transaction(func(tx *gorm.DB) error {

		ormUser.Emails = append(ormUser.Emails, &ormEmail)
		ormUser.LoginEmailId = ormEmail.Id
		ormUser.LoginEmail = ormEmail.Email

		// Create user first
		log.Println("Creating user in transaction")
		if err := tx.Create(&ormUser).Error; err != nil {
			return err
		}

		// // Create email
		// if err := tx.Create(&ormEmail).Error; err != nil {
		// 	return err
		// }

		// Create permissions
		log.Printf("Creating user permissions for user %s in transaction\n", userId)
		err := u.accountPermissions.UpsertAccountPermissionsForRole(tx, accountId, userId, newUser.RoleKind)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return ormUser.Detail(), nil
}

func (u *UserResource) GetUser(accountId util.AccountId, userId util.UserId) (*models.UserDetail, error) {
	userORM := &models.UserORM{}

	querier := u.db.
		Preload("Emails").
		Preload("RootUserPermissions").
		Preload("AccountPermissions")

	// if err := querier.First(userORM, string(userId)).Error; err != nil {
	// 	return nil, err
	// }

	log.Println("Getting user: account_id: ", accountId)
	log.Println("Getting user: user_id: ", userId)

	if err := querier.Where("account_id = ? AND id = ?", string(accountId), string(userId)).First(userORM).Error; err != nil {
		return nil, err
	}

	userDetail := userORM.Detail()

	return userDetail, nil
}

func (u *UserResource) GetUserByEmail(email string) (*models.UserDetail, error) {
	userORM := &models.UserORM{}

	querier := u.db.
		Preload("Emails").
		Preload("RootUserPermissions").
		Preload("AccountPermissions")

	if err := querier.Where(" LOWER(login_email) = LOWER(?)", email).First(userORM).Error; err != nil {
		u.logger.Printf("GetUserByEmail: Error getting user by email %s: %v\n", email, err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	userDetail := userORM.Detail()

	return userDetail, nil
}

func (u *UserResource) UpdateUser(id string, updateUser *models.UpdateUser) (*models.UserDetail, error) {
	userORM := &models.UserORM{}

	// if err := u.db.Preload("BillingAddress").Preload("ShippingAddress").Preload("Emails").First(userORM, id).Error; err != nil {
	// 	return err
	// }

	err := u.db.Transaction(func(tx *gorm.DB) error {

		querier := tx.
			Preload("Emails").
			Preload("RootUserPermissions").
			Preload("AccountPermissions")

		if err := querier.Where("id = ?", id).First(userORM).Error; err != nil {
			return err
		}

		if updateUser.FirstName != nil {
			userORM.FirstName = *updateUser.FirstName
		}
		if updateUser.MiddleName != nil {
			userORM.MiddleName = *updateUser.MiddleName
		}
		if updateUser.LastName != nil {
			userORM.LastName = *updateUser.LastName
		}

		if updateUser.AccountUserPrimaryRole != nil {
			userORM.SetAccountUserPrimaryRole(*updateUser.AccountUserPrimaryRole)
		}

		// TODO: Restore addresses

		// if updateUser.BillingAddress != (models.Address{}) {
		// 	if userORM.BillingAddress == nil {
		// 		userORM.BillingAddress = &models.AddressORM{}
		// 	}

		// 	userORM.BillingAddress.City = updateUser.BillingAddress.City
		// 	userORM.BillingAddress.Country = updateUser.BillingAddress.Country
		// 	userORM.BillingAddress.Line1 = updateUser.BillingAddress.Line1
		// 	userORM.BillingAddress.Line2 = updateUser.BillingAddress.Line2
		// 	userORM.BillingAddress.PostalCode = updateUser.BillingAddress.PostalCode
		// 	userORM.BillingAddress.State = updateUser.BillingAddress.State
		// }

		// if updateUser.ShippingAddress != (models.Address{}) {
		// 	if userORM.ShippingAddress == nil {
		// 		userORM.ShippingAddress = &models.AddressORM{}
		// 	}

		// 	userORM.ShippingAddress.City = updateUser.ShippingAddress.City
		// 	userORM.ShippingAddress.Country = updateUser.ShippingAddress.Country
		// 	userORM.ShippingAddress.Line1 = updateUser.ShippingAddress.Line1
		// 	userORM.ShippingAddress.Line2 = updateUser.ShippingAddress.Line2
		// 	userORM.ShippingAddress.PostalCode = updateUser.ShippingAddress.PostalCode
		// 	userORM.ShippingAddress.State = updateUser.ShippingAddress.State
		// }

		if err := tx.Save(userORM).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Printf("UserResource: Error updating user: %s\n", err)
		return nil, err
	}

	return userORM.Detail(), nil
}

func (u *UserResource) DeleteUser(id string) (*models.DeleteUserResponse, error) {
	userORM := &models.UserORM{}

	if err := u.db.Where("id = ?", id).First(userORM).Error; err != nil {
		return nil, err
	}

	ts := time.Now()

	userORM.DeletedAt = &ts

	if err := u.db.Save(userORM).Error; err != nil {
		return nil, err
	}

	deleted := &models.DeleteUserResponse{
		AccountId: userORM.AccountId,
		Id:        id,
	}

	return deleted, nil
}

func (u *UserResource) ListUsers(accountId *util.AccountId) ([]*models.UserDetail, error) {
	userORMs := make([]*models.UserORM, 0)

	// if err := u.db.Preload("BillingAddress").Preload("ShippingAddress").Preload("Emails").Find(&userORMs).Error; err != nil {
	// 	return nil, err
	// }

	querier := u.db.
		Preload("Emails").
		Preload("AccountPermissions")

	if accountId != nil {
		querier = querier.Where("account_id = ?", accountId)
	}

	// Exclude deleted users
	querier = querier.Where("deleted_at IS NULL")

	if err := querier.Find(&userORMs).Error; err != nil {
		return nil, err
	}

	userDetails := make([]*models.UserDetail, 0)

	for _, userORM := range userORMs {
		userDetail := userORM.Detail()

		userDetails = append(userDetails, userDetail)
	}

	return userDetails, nil
}
