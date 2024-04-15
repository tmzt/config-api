package routes

import (
	"github.com/tmzt/config-api/config"
	"github.com/tmzt/config-api/util"

	restful "github.com/emicklei/go-restful/v3"
)

type AccountRoute struct {
	logger util.SetRequestLogger
	props  *NewAccountProps
}

type NewAccountProps struct {
	ConfigService *config.ConfigService
}

func NewAccountRoute(
	props *NewAccountProps,
) *AccountRoute {
	logger := util.NewLogger("AccountRoute", 0)

	return &AccountRoute{
		logger: logger,
		props:  props,
	}
}

func (r *AccountRoute) RegisterAccountRoute(path string, subAccount bool, container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path(path).
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	configService := r.props.ConfigService

	NewConfigRoute(configService).Prefixed(ws, "/")
	NewConfigDiffRoute(configService).Prefixed(ws, "/")
	NewConfigSchemaRoute(configService).Prefixed(ws, "/")

	container.Add(ws)
}
