package bootstrap

import (
	"context"
	"fmt"
	"log"
	"slices"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/db"
	"github.com/justblue/samsa/internal/infras/postgres"
	"github.com/justblue/samsa/tools/module"
	"github.com/justblue/samsa/tools/sqlc"
	"github.com/urfave/cli/v3"
)

func Serve(version string) *cli.Command {
	cmd := &cli.Command{
		Name:      "samsa",
		Version:   version,
		Usage:     "cli for the simple writing app of samsa",
		ArgsUsage: "[args and such]",
		Commands: []*cli.Command{
			GetServeCmd(version),
			GetToolsCmd(),
		},
	}

	return cmd
}

//	@title			Samsa
//	@version		0.1.0
//	@description	Samsa is a simple backend for a writing platform.
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	https://github.com/justblue/samsa
//	@contact.email	trao0312@gmail.com

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@host		localhost:8000
//	@BasePath	/api/v1

// @externalDocs.description	OpenAPI
// @externalDocs.url			https://swagger.io/resources/open-api/
func GetServeCmd(version string) *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "serve the samsa server",
		Action: func(ctx context.Context, c *cli.Command) error {
			cfg, err := config.New()
			if err != nil {
				log.Fatal("config:", err)
			}

			app, err := Init(version, cfg)
			if err != nil {
				log.Fatal("init:", err)
			}

			if err := Run(app); err != nil {
				log.Fatal("run:", err)
			}
			return nil
		},
	}
}

func GetToolsCmd() *cli.Command {
	return &cli.Command{
		Name:  "tools",
		Usage: "Tools commands",
		Commands: []*cli.Command{
			{
				Name:  "sqlc",
				Usage: "transform sqlc models",
				Action: func(ctx context.Context, c *cli.Command) error {
					if success, _ := sqlc.TransformModels(); !success {
						return fmt.Errorf("failed to transform SQLc models")
					}
					return nil
				},
			},
			{
				Name:  "create-module",
				Usage: "create a new module",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name:      "module",
						UsageText: "Name of the module to create",
					},
				},
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:    "ignore-files",
						Usage:   "Files to ignore (Valid values: repository, http_handler, usecase, register, models)",
						Aliases: []string{"i"},
						Validator: func(s []string) error {
							for _, name := range s {
								if !slices.Contains(module.FilesToCreate, name) {
									return fmt.Errorf("invalid input: %s", name)
								}
							}
							return nil
						},
					},
					&cli.BoolFlag{
						Name:    "force",
						Usage:   "Allow to override the exist folder",
						Aliases: []string{"f"},
						Value:   false,
					},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					moduleName := c.StringArg("module")
					ignoreFiles := c.StringSlice("ignore-files")
					force := c.Bool("force")
					return module.CreateModule(moduleName, ignoreFiles, force)
				},
			},
			{
				Name:  "migrate",
				Usage: "run database migrations",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name:      "rev",
						UsageText: "revision to migrate to (up/down)",
					},
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "test",
						Usage:   "apply migrations for test database",
						Aliases: []string{"t"},
						Value:   false,
					},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					rev := c.StringArg("rev")
					isTest := c.Bool("test")

					cfg, err := config.New()
					if err != nil {
						log.Fatal("config:", err)
					}

					var pool *pgxpool.Pool
					if isTest {
						pool, err = postgres.NewTestDatabase(ctx, cfg)
						if err != nil {
							log.Fatal("initialize postgres failed:", err)
						}
					} else {
						pool, err = postgres.New(ctx, cfg)
						if err != nil {
							log.Fatal("initialize postgres failed:", err)
						}
					}

					migrator := db.Migrator(pool)
					switch rev {
					case "up":
						return migrator.Up()
					case "down":
						return migrator.Down()
					default:
						return fmt.Errorf("invalid revision: %s", rev)
					}
				},
			},
		},
	}
}
