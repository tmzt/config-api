package filters

import (
	"net/http"
	"strings"

	restful "github.com/emicklei/go-restful/v3"

	"github.com/tmzt/config-api/util"
)

type ParentAccountFilter struct {
}

func NewParentAccountFilter() *ParentAccountFilter {
	return &ParentAccountFilter{}
}

func (f *ParentAccountFilter) validatePlatformUser(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	accountId := req.PathParameter("accountId")
	if accountId == util.ROOT_ACCOUNT_ID {
		chain.ProcessFilter(req, resp)
		return
	}

	// If the user is not a platform user, they must be authorized
	if !util.IsRequestAuthorized(req) {
		resp.WriteErrorString(http.StatusForbidden, "Forbidden")
		return
	}

	chain.ProcessFilter(req, resp)
}

func (f *ParentAccountFilter) Filter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	pathname := req.Request.URL.Path

	isRootUser := util.RequestBoolAttribute(req, "isRootUser")

	// If this is a platform route, it's the same as /accounts/{ROOT_ACCOUNT_ID}
	if isRootUser && strings.HasPrefix(pathname, "/platform/") {
		req.SetAttribute("platformAccount", true)
		req.SetAttribute("parentAccountId", util.ROOT_ACCOUNT_ID)
		chain.ProcessFilter(req, resp)
		return
	}

	parentAccountId := req.PathParameter("parentAccountId")
	if parentAccountId != "" {
		req.SetAttribute("actualParentAccountId", parentAccountId)
		req.SetAttribute("parentAccountId", parentAccountId)
		chain.ProcessFilter(req, resp)
		return
	}

	chain.ProcessFilter(req, resp)
}
