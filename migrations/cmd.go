package migrations

import (
	"context"
	"embed"
	"fmt"
	"log"

	"github.com/pressly/goose/v3"
	"github.com/tmzt/config-api/util"
	"github.com/urfave/cli/v2"
	"gorm.io/gorm"
)

//go:embed *.sql
var embedMigrations embed.FS

type SchemaMigrations struct {
	logger util.SetRequestLogger
	db     *gorm.DB
}

func NewSchemaMigrations(db *gorm.DB) *SchemaMigrations {
	logger := util.NewLogger("SchemaMigrations", 0)
	return &SchemaMigrations{
		logger: logger,
		db:     db,
	}
}

func (m *SchemaMigrations) Run(ctx context.Context, db *gorm.DB, command string, args ...string) error {

	sqlDb, err := db.DB()
	if err != nil {
		m.logger.Fatalf("db.DB() error: %v\n", err)
	}

	goose.SetLogger(m.logger)
	goose.SetBaseFS(embedMigrations)

	// args = append([]string{command}, args...)

	// m.logger.Printf("Running schema migrations with args: %s\n", fmt.Sprint([]any(args)...))

	err = goose.RunContext(ctx, command, sqlDb, ".", args...)
	if err != nil {
		m.logger.Fatalf("goose.RunContext() error: %v\n", err)
	}
	return err
}

func createSchemaGooseCommand(db *gorm.DB, command string) *cli.Command {
	migrations := NewSchemaMigrations(db)
	return &cli.Command{
		Name:  command,
		Usage: fmt.Sprintf("Run %s command with goose", command),
		// Hidden: true,
		Action: func(c *cli.Context) error {
			args := c.Args().Slice()

			log.Printf("Running schema migrations: args: %+v\n", args)

			return migrations.Run(c.Context, db, command, args...)
		},
	}
}

var gooseCommands = []string{
	"up",
	"down",
	"redo",
	"version",
}

func CreateSchemaCommand(db *gorm.DB) *cli.Command {
	subcommands := []*cli.Command{}
	for _, command := range gooseCommands {
		subcommands = append(subcommands, createSchemaGooseCommand(db, command))
	}

	return &cli.Command{
		Name:        "schema",
		Usage:       "Manage database schema with goose",
		Subcommands: subcommands,
	}
}
