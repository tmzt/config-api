package routes

import (
	"net/http"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/tmzt/config-api/models"
	"github.com/tmzt/config-api/resources"
	"github.com/tmzt/config-api/util"
)

type AuthRoute struct {
	logger       util.SetRequestLogger
	authResource *resources.AuthResource
}

func NewAuthRoute(authResource *resources.AuthResource) *AuthRoute {
	logger := util.NewLogger("AuthRoute", 0)

	return &AuthRoute{
		logger:       logger,
		authResource: authResource,
	}
}

func (r *AuthRoute) createToken(request *restful.Request, response *restful.Response) {
	r.logger.SetRequest(request)
	r.logger.Println("Called createToken")

	newToken := new(models.NewToken)

	err := request.ReadEntity(newToken)

	if err != nil {
		response.WriteErrorString(http.StatusUnprocessableEntity, err.Error())
		return
	}

	tokenDetail, err := r.authResource.CreateToken(newToken)
	if err != nil {
		response.WriteErrorString(http.StatusInternalServerError, err.Error())
		return
	} else if tokenDetail == nil {
		response.WriteHeader(http.StatusNotFound)
		return
	}

	response.WriteHeaderAndEntity(http.StatusCreated, tokenDetail)
}

func (r *AuthRoute) Register(container *restful.Container) {
	ws := new(restful.WebService)

	ws.Path("/auth").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.POST("/tokens").
		To(r.createToken).
		Doc("Create a new token").
		Reads(models.NewToken{}).
		Writes(models.TokenDetail{}))

	container.Add(ws)
}

func (r *AuthRoute) Prefixed(ws *restful.WebService, prefix string) {
	ws.Route(ws.POST(prefix + "/tokens").
		To(r.createToken).
		Doc("Create a new token").
		Reads(models.NewToken{}).
		Writes(models.TokenDetail{}))
}
