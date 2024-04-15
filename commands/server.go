package commands

import (
	"log"
	"net/http"
	"strings"

	"github.com/tmzt/config-api/config"
	"github.com/tmzt/config-api/filters"
	"github.com/tmzt/config-api/resources"
	"github.com/tmzt/config-api/routes"
	"github.com/tmzt/config-api/services"
	"github.com/tmzt/config-api/util"
	"gorm.io/gorm"

	restful "github.com/emicklei/go-restful/v3"
	redis "github.com/go-redis/redis/v8"
	cli "github.com/urfave/cli/v2"
)

type Server struct {
	logger util.SetRequestLogger
	addr   string
	db     *gorm.DB
	rdb    *redis.Client
}

func NewServer(addr string, db *gorm.DB, rdb *redis.Client) *Server {
	logger := util.NewLogger("Server", 0)

	return &Server{
		logger: logger,
		addr:   addr,
		db:     db,
		rdb:    rdb,
	}
}

func (s *Server) Run(c *cli.Context) error {

	container := restful.NewContainer()

	jwtSvc := services.NewJwtService()

	platformPermissions := resources.NewPlatformPermissionsResource(s.rdb, s.db)
	accountPermissions := resources.NewAccountPermissionsResource(s.rdb, s.db)

	// NOTE: for now, this is just used for config objects
	// we will probably want a separate service for config when
	// LRU is implemented. We may need to be more granular than that
	cacheService := util.NewCacheService(s.rdb)

	accountResource := resources.NewAccountResource(s.db, platformPermissions, accountPermissions, jwtSvc)
	accountResource.MustEnsurePlatform()

	userResource := resources.NewUserResource(s.db, accountPermissions)

	authResource := resources.NewAuthResource(s.rdb, s.db, userResource, platformPermissions, accountPermissions, jwtSvc)

	configService := config.NewConfigService(s.db, s.rdb, cacheService)

	authRoute := routes.NewAuthRoute(authResource)

	// Account hierarchy routes
	accountRoute := routes.NewAccountRoute(
		&routes.NewAccountProps{
			ConfigService: configService,
		},
	)

	// Register routes
	authRoute.Register(container)
	accountRoute.RegisterAccountRoute("/accounts/{accountId}", false, container)

	// Answer health checks immediately
	container.Filter(func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		if req.Request.URL.Path == "/health" && req.Request.Method == "GET" {
			resp.WriteHeader(http.StatusOK)
			return
		}
		chain.ProcessFilter(req, resp)
	})

	cors := restful.CrossOriginResourceSharing{
		// ExposeHeaders:  []string{"X-My-Header"},
		ExposeHeaders:  []string{"Range", "Content-Length", "Content-Range", "ETag", "X-Content-Hash"},
		AllowedHeaders: []string{"Content-Type", "Accept", "Accept-Language", "Authorization", "Referer", "User-Agent", "Origin", "Range", "If-None-Match"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedDomainFunc: func(origin string) bool {
			log.Printf("checking domain: %s", origin)

			// Allow localhost for development
			return strings.HasPrefix(origin, "http://localhost")
		},
		CookiesAllowed: true,
		Container:      container}
	container.Filter(cors.Filter)

	container.Filter(filters.NewBypassAuthFilter().Filter)

	container.Filter(filters.NewTokenAuthorizationFilter(platformPermissions, accountPermissions, jwtSvc).Filter)

	container.Filter(filters.NewAuthorizationFilter(platformPermissions, accountPermissions).Filter)

	// container.Filter(filters.NewConfigContextFilter(configService).Filter)

	container.Filter(func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		if req.Request.Method == "OPTIONS" {
			resp.WriteHeader(http.StatusOK)
			return
		}
		chain.ProcessFilter(req, resp)
	})

	// Start the server
	log.Fatal(http.ListenAndServe(s.addr, container.ServeMux))

	return nil
}

func MakeServerCommand(addr string, db *gorm.DB, rdb *redis.Client) *cli.Command {
	return &cli.Command{
		Name:  "server",
		Usage: "Start the server",
		Action: func(c *cli.Context) error {
			// If -addr was passed, use that
			if c.String("addr") != "" {
				addr = c.String("addr")
			}
			server := NewServer(addr, db, rdb)
			return server.Run(c)
		},
		ArgsUsage: "[-addr :8080]",
	}
}
