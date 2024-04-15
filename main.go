package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/tmzt/config-api/commands"
	"github.com/tmzt/config-api/config"
	"github.com/tmzt/config-api/connections"
	"github.com/tmzt/config-api/util"
	cli "github.com/urfave/cli/v2"
)

const apiAddr = ":8001"

func main() {
	log.Default().SetFlags(log.LstdFlags | log.Lshortfile)
	logger := log.New(os.Stdout, "main: ", log.LstdFlags|log.Lshortfile)

	if err := godotenv.Load(); err != nil {
		logger.Fatal("Error loading .env file")
	}

	postgresUrl := util.MustGetPostgresURL()

	// Create a new database connection
	db := connections.MustNewDB(postgresUrl)
	rdb := connections.MustNewRedis()

	app := &cli.App{
		Name:  "config-api",
		Usage: "config-api server-side cli",
		Commands: []*cli.Command{
			connections.CreateAutoMigrateCommand(db),
			// migrations.CreateSchemaCommand(db),
			config.CreateConfigCommand(db),
			commands.MakeServerCommand(apiAddr, db, rdb),
		},
	}

	app.Run(os.Args)
}
