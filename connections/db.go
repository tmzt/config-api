package connections

import (
	"log"
	"os"
	"time"

	"github.com/tmzt/config-api/config"
	"github.com/tmzt/config-api/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const enableLogger = true

func MustNewDB(postgresUrl string) *gorm.DB {

	config := &gorm.Config{}

	if enableLogger {
		dbLogger := logger.New(
			log.New(os.Stderr, "\r\n", log.LstdFlags), // io writer
			logger.Config{
				SlowThreshold: time.Second, // Slow SQL threshold
				LogLevel:      logger.Info, // Log level
				Colorful:      true,        // Disable color
			},
		)
		config.Logger = dbLogger
	}

	// Create a new database connection
	db, err := gorm.Open(postgres.Open(postgresUrl), config)
	if err != nil {
		log.Fatalf("failed to connect to database: %v\n", err)
	}

	return db
}

func AutoMigrate(db *gorm.DB) {
	// Migrate the schema
	err := db.AutoMigrate(
		&models.AccountORM{},
		&models.PlatformAccountDataORM{},
		&models.UserORM{},
		&models.EmailORM{},
		&models.AddressORM{},
		&models.UserAddressORM{},
		&models.PlatformPermissionsORM{},
		&models.AccountPermissionsORM{},

		// TODO: Move account-related models
		// to a schema for each account

		&config.ConfigORM{},
		&config.ConfigVersionORM{},
		&config.ConfigReferenceORM{},
		&config.ConfigTagORM{},

		// &config.ConfigRecordORM{},
		&config.ConfigNodeORM{},

		&config.ConfigKeyedDataORM{},
		&config.ConfigDocumentORM{},
	)

	if err != nil {
		log.Fatalf("failed to migrate schema: %v\n", err)
	}
}
