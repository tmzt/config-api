package resources

import (
	"fmt"
	"log"

	"github.com/tmzt/config-api/models"
	"github.com/tmzt/config-api/services"
	"github.com/tmzt/config-api/util"

	"gorm.io/gorm"
)

type AccountResource struct {
	logger              util.SetRequestLogger
	platformPermissions *PlatformPermissionsResource
	accountPermissions  *AccountPermissionsResource
	jwtService          *services.JwtService
	db                  *gorm.DB
}

func NewAccountResource(
	db *gorm.DB,
	platformPermissions *PlatformPermissionsResource,
	accountPermissions *AccountPermissionsResource,
	jwtService *services.JwtService,
) *AccountResource {
	logger := util.NewLogger("AccountResource", 0)

	return &AccountResource{
		logger:              logger,
		db:                  db,
		platformPermissions: platformPermissions,
		accountPermissions:  accountPermissions,
		jwtService:          jwtService,
	}
}

func (r *AccountResource) CheckSubdomainAvailability(subdomain string) (*bool, error) {
	// var count int64

	// err := r.db.Table("accounts").
	// 	Joins("LEFT JOIN platform_account_data pad ON pad.account_id = accounts.id").
	// 	Where("LOWER(pad.api_domain) = LOWER(?)", subdomain).
	// 	Select(gorm.Expr("1")).
	// 	Limit(1).
	// 	Count(&count).Error

	query := `
		SELECT 1 FROM accounts a
			LEFT JOIN platform_account_data pad ON pad.account_id = a.id
		WHERE LOWER(pad.api_domain) = LOWER(?)
	`

	if err := r.db.Raw(query, subdomain).Error; err != nil {
		log.Printf("Error checking subdomain availability: %v", err)
		return nil, err
	}

	return util.BoolPtr(true), nil
}

func (r *AccountResource) MustEnsurePlatform() {
	accountDomain := util.ROOT_ACCOUNT_DOMAIN
	accountId := util.ROOT_ACCOUNT_ID

	err := r.db.Transaction(func(tx *gorm.DB) error {
		ormAccount := models.AccountORM{}
		err := tx.
			// Where("api_domain = ?", accountDomain).
			Where("id = ?", accountId).
			Attrs(models.AccountORM{Id: accountId}).
			Assign(models.AccountORM{
				State: models.AccountStateActive,
				// ApiDomain: accountDomain,
			}).
			FirstOrCreate(&ormAccount).Error

		if err != nil {
			return err
		}

		ormPlatformAccountData := models.PlatformAccountDataORM{}
		err = tx.
			Where("account_id = ?", accountId).
			Attrs(models.PlatformAccountDataORM{AccountId: accountId}).
			Assign(models.PlatformAccountDataORM{
				ApiDomain: &accountDomain,
			}).
			FirstOrCreate(&ormPlatformAccountData).Error

		if err != nil {
			return err
		}

		ormEmail := models.EmailORM{
			AccountId:  accountId,
			Id:         util.NewUUID(),
			Email:      "platform@test.tld",
			Subscribed: true,
			Verified:   true,
		}

		ormUser := models.UserORM{}
		err = r.db.
			Where("account_id = ?", accountId).
			Attrs(&models.UserORM{
				AccountId:  accountId,
				Id:         util.NewUUID(),
				LoginEmail: "platform@test.tld",
				Emails:     []*models.EmailORM{&ormEmail},
				FirstName:  "Zach",
				MiddleName: "T",
				LastName:   "Zuniga",
			}).
			FirstOrCreate(&ormUser).Error

		if err != nil {
			return err
		}

		ormRootUserPermissions := models.PlatformPermissionsORM{}
		err = r.db.
			Where("user_id = ?", ormUser.Id).
			Attrs(&models.PlatformPermissionsORM{
				Id:        util.NewUUID(),
				AccountId: accountId,
				UserId:    ormUser.Id,
			}).
			Assign(&models.PlatformPermissionsORM{
				Admin:        true,
				RootUser:     true,
				InternalUser: true,
			}).
			FirstOrCreate(&ormRootUserPermissions).Error

		if err != nil {
			return err
		}

		ormAccountPermissions := models.AccountPermissionsORM{}
		err = r.db.
			Where("account_id = ? AND user_id = ?", accountId, ormUser.Id).
			Attrs(&models.AccountPermissionsORM{
				AccountId:    accountId,
				Id:           util.NewUUID(),
				UserId:       ormUser.Id,
				AccountOwner: true,
				AccountAdmin: true,
				AccountUser:  true,
			}).
			FirstOrCreate(&ormAccountPermissions).Error

		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Fatalf("Error ensuring platform account objects exist: %v", err)
	}
}

func (r *AccountResource) CreateAccount(newAccount *models.NewAccount) (string, error) {
	accountState := models.AccountStatePlatformSignup

	ormAccount := models.AccountORM{
		// ApiDomain: newAccount.ApiDomain,
		State: accountState,
	}

	ormAccount.PlatformAccountData = &models.PlatformAccountDataORM{
		ApiDomain: util.StrPtr(newAccount.ApiDomain),
	}

	err := r.db.Transaction(func(tx *gorm.DB) error {
		ormAccount.Id = util.NewUUID()

		if err := tx.Create(&ormAccount).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	accountId := ormAccount.Id

	return accountId, nil
}

func (r *AccountResource) GetAccountModel(id util.AccountId) (*models.AccountORM, error) {
	accountORM := &models.AccountORM{}

	querier := r.db.
		Preload("StripeConnection")

	err := querier.First(accountORM, "id = ?", string(id)).Error
	if err != nil {
		return nil, err
	}

	return accountORM, nil
}

func (r *AccountResource) GetAccount(id util.AccountId) (*models.AccountDetail, error) {
	accountORM, err := r.GetAccountModel(id)
	if err != nil {
		return nil, err
	}

	return accountORM.Detail(), nil
}

func (r *AccountResource) GetAccountByApiDomain(apiDomain string) (*models.AccountDetail, error) {
	accountORM := &models.AccountORM{}
	err := r.db.Preload("PlatformAccountData").
		Joins("LEFT JOIN platform_account_data pad ON pad.account_id = accounts.id").
		First(accountORM, "pad.api_domain = ?", apiDomain).Error
	if err != nil {
		return nil, err
	}

	return accountORM.Detail(), nil
}

func (r *AccountResource) UpdateAccount(id string, updateAccount *models.UpdateAccount) error {
	accountORM := &models.AccountORM{}
	err := r.db.Preload("PlatformAccountData").First(accountORM, "id = ?", id).Error
	if err != nil {
		return err
	}

	if updateAccount.ApiDomain != nil {
		if accountORM.PlatformAccountData != nil && accountORM.PlatformAccountData.ApiDomain != nil {
			// TODO: Implement account domain update
			return fmt.Errorf("cannot update account domain without specific API call")
		}
		if accountORM.PlatformAccountData == nil {
			accountORM.PlatformAccountData = &models.PlatformAccountDataORM{
				ApiDomain: updateAccount.ApiDomain,
			}
		} else {
			accountORM.PlatformAccountData.ApiDomain = updateAccount.ApiDomain
		}
	}
	if updateAccount.AccountState != nil {
		accountORM.State = *updateAccount.AccountState
	}

	err = r.db.Save(accountORM).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *AccountResource) DeleteAccount(id string) error {
	accountORM := &models.AccountORM{}
	err := r.db.First(accountORM, "id = ?", id).Error
	if err != nil {
		return err
	}

	err = r.db.Delete(accountORM).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *AccountResource) ListAccounts(parentAccountId util.AccountId) ([]*models.AccountDetail, error) {
	accountORMs := []*models.AccountORM{}

	querier := r.db.Preload("PlatformAccountData")

	condition := "parent_account_id = ?"
	if string(parentAccountId) == util.ROOT_ACCOUNT_ID {
		condition = "parent_account_id = ? OR parent_account_id IS NULL"
	}

	r.logger.Printf("ListAccounts: condition: %s\n", condition)
	r.logger.Printf("ListAccounts: parentAccountId: %s\n", parentAccountId)

	err := querier.Find(&accountORMs, condition, string(parentAccountId)).Error
	if err != nil {
		return nil, err
	}

	accountDetails := make([]*models.AccountDetail, len(accountORMs))
	for i, accountORM := range accountORMs {
		accountDetails[i] = accountORM.Detail()
	}

	return accountDetails, nil
}
