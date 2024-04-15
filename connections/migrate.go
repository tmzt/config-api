package connections

import (
	"log"

	"github.com/urfave/cli/v2"
	"gorm.io/gorm"
)

type Migrations struct {
	logger *log.Logger
	db     *gorm.DB
}

func (m *Migrations) Run(db *gorm.DB) {
	AutoMigrate(db)
	m.logger.Println("Migration complete")
}

func NewMigrations(db *gorm.DB) *Migrations {
	logger := log.New(log.Writer(), "Migrations: ", log.LstdFlags|log.Lshortfile)
	return &Migrations{
		logger: logger,
		db:     db,
	}
}

func CreateAutoMigrateCommand(db *gorm.DB) *cli.Command {
	migrations := NewMigrations(db)
	return &cli.Command{
		Name:  "automigrate",
		Usage: "Run Gorm database auto migrations",
		Action: func(c *cli.Context) error {
			migrations.Run(db)
			return nil
		},
	}
}
