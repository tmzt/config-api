package config

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/tmzt/config-api/util"
	"github.com/urfave/cli/v2"
	"gorm.io/gorm"
)

//go:embed schemas/*.schema.json
var embedConfigSchemas embed.FS

func getCmdAddConfigSchemaParams(c *cli.Context) *cmdAddConfigSchemaParams {
	params := &cmdAddConfigSchemaParams{}

	scope := util.ScopeKindAsPtr(util.ScopeKindGlobal)

	scopeStr := c.String("scope")
	accountId := c.String("account")
	userId := c.String("user")

	if userId != "" {
		if scopeStr != "" && scopeStr != "user" {
			log.Fatal("User id requires scope to be user")
			return nil
		}
		scope = util.ScopeKindAsPtr(util.ScopeKindUser)
	} else if accountId != "" {
		if scopeStr != "" && scopeStr != "account" {
			log.Fatal("Account id requires scope to be account")
		}
		scope = util.ScopeKindAsPtr(util.ScopeKindAccount)
	}

	if *scope != util.ScopeKindGlobal {
		if accountId == "" {
			log.Fatal("Account id required when scope is account or user")
			return nil
		}
		params.AccountId = util.AccountIdPtr(accountId)

		if *scope == util.ScopeKindUser {
			if userId == "" {
				log.Fatal("User id required when scope is user")
				return nil
			}
			params.UserId = util.UserIdPtr(userId)
		}
	}

	if c.Bool("all") {
		params.All = true
	} else {
		name := c.String("name")
		path := c.String("path")
		if name != "" && path != "" {
			log.Fatal("Only one of name or path can be specified")
			return nil
		} else if name == "" && path == "" {
			log.Fatal("One of name or path must be specified")
			return nil
		} else if name != "" {
			params.SchemaName = &name
		} else if path != "" {
			params.SchemaPath = &path
		}
	}

	params.Scope = scope
	return params
}

type cmdAddConfigSchemaParams struct {
	Scope     *util.ScopeKind
	AccountId *util.AccountId
	UserId    *util.UserId

	Fs         bool
	SchemaName *string
	SchemaPath *string
	All        bool
}

func makeAddConfigSchemaFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "scope",
			Usage: "Scope (global, account, user)",
		},
		&cli.StringFlag{
			Name:  "account",
			Usage: "Account id",
		},
		&cli.StringFlag{
			Name:  "user",
			Usage: "User id",
		},
		&cli.StringFlag{
			Name:  "name",
			Usage: "Schema name (loads from compiled-in schemas)",
		},
		&cli.StringFlag{
			Name:  "path",
			Usage: "Schema path (loads from system filesystem, relative to current directory. Using a directory will load all schemas in the directory.)",
		},
		&cli.BoolFlag{
			Name:  "all",
			Usage: "All (adds all compiled-in schemas)",
		},
	}
}

type ConfigSchemaMigrations struct {
	logger util.SetRequestLogger
	db     *gorm.DB
}

func (m *ConfigSchemaMigrations) AddSchema(ctx context.Context, params *cmdAddConfigSchemaParams) error {
	var _fs fs.FS = &osFS{}

	m.logger.Printf("Adding schema: %+v\n", params)

	schemaPath := ""

	// If a schema name was provided, use the embedded filesystem
	if params.SchemaName != nil {
		m.logger.Printf("Using embedded schemas filesystem")
		_fs = embedConfigSchemas

		schemaPath = path.Join("schemas", *params.SchemaName+".schema.json")
	} else if params.SchemaPath == nil {
		m.logger.Fatalf("Schema path is required if schema name not provided.\n")
		return fmt.Errorf("schema path is required if schema name not provided")
	} else {
		schemaPath = filepath.FromSlash(*params.SchemaPath)
	}

	m.logger.Printf("Schema path: %s\n", schemaPath)

	// Collect schemas
	var schemas []string
	if strings.HasSuffix(schemaPath, ".schema.json") {
		m.logger.Printf("Adding single schema: %s\n", schemaPath)
		schemas = []string{
			filepath.FromSlash(schemaPath),
		}
	} else {
		globPath := "*.schema.json"
		if !params.All {
			globPath = path.Join(schemaPath, "*.schema.json")
		}

		var err error
		schemas, err = fs.Glob(_fs, globPath)
		if err != nil {
			m.logger.Fatalf("Unable to load schemas from %s: %+v\n", globPath, err)
		}
	}

	// Load schemas
	m.logger.Printf("Loading %d schemas...\n", len(schemas))

	for _, schemaPath := range schemas {
		m.logger.Printf("Loading schema: %s (nop)\n", schemaPath)
	}

	return nil
}

func (m *ConfigSchemaMigrations) AddSchemas(ctx context.Context, params *cmdAddConfigSchemaParams) error {
	return nil
}

func NewConfigSchemaMigrations(db *gorm.DB) *ConfigSchemaMigrations {
	logger := util.NewLogger("ConfigSchemaMigrations", 0)
	return &ConfigSchemaMigrations{
		logger: logger,
		db:     db,
	}
}

func CreateConfigSchemaAddCommand(db *gorm.DB) *cli.Command {
	migrations := NewConfigSchemaMigrations(db)
	return &cli.Command{
		Name:  "add",
		Usage: "Add schema to scope (inserts new DAG node)",
		// Hidden: true,
		Action: func(c *cli.Context) error {
			params := getCmdAddConfigSchemaParams(c)

			if params.All {
				return migrations.AddSchemas(c.Context, params)
			}

			return migrations.AddSchema(c.Context, params)
		},
		Flags: makeAddConfigSchemaFlags(),
	}
}

func CreateConfigSchemaCommand(db *gorm.DB) *cli.Command {
	subcommands := []*cli.Command{
		CreateConfigSchemaAddCommand(db),
	}

	return &cli.Command{
		Name:        "schema",
		Usage:       "Manage config system (DAG) JSON schemas",
		Subcommands: subcommands,
	}
}

func CreateConfigCommand(db *gorm.DB) *cli.Command {
	subcommands := []*cli.Command{
		CreateConfigSchemaCommand(db),
	}

	return &cli.Command{
		Name:        "config",
		Usage:       "Manage config system (DAG)",
		Subcommands: subcommands,
	}
}

type osFS struct{}

func (osFS) Open(name string) (fs.File, error) { return os.Open(filepath.FromSlash(name)) }

func (osFS) ReadDir(name string) ([]fs.DirEntry, error) { return os.ReadDir(filepath.FromSlash(name)) }

func (osFS) Stat(name string) (fs.FileInfo, error) { return os.Stat(filepath.FromSlash(name)) }

func (osFS) ReadFile(name string) ([]byte, error) { return os.ReadFile(filepath.FromSlash(name)) }

func (osFS) Glob(pattern string) ([]string, error) { return filepath.Glob(filepath.FromSlash(pattern)) }
