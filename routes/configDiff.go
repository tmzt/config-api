package routes

import (
	"context"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/tmzt/config-api/config"
	"github.com/tmzt/config-api/util"
)

type ConfigDiffRoute struct {
	logger        util.SetRequestLogger
	configService *config.ConfigService
}

func NewConfigDiffRoute(resource *config.ConfigService) *ConfigDiffRoute {
	logger := util.NewLogger("ConfigDiffRoute", 0)

	return &ConfigDiffRoute{
		logger:        logger,
		configService: resource,
	}
}

// // Get the values for a config key
// func (r *ConfigDiffRoute) getKeyedConfigValues(req *restful.Request, res *restful.Response) {
// 	accountId := util.GetValidatedRequestAccountId(req)
// 	if accountId == nil {
// 		res.WriteErrorString(http.StatusForbidden, "Invalid account id")
// 		return
// 	}

// 	userId := util.GetRequestUserIdAttribute(req)

// 	configCollectionKeyParam := req.PathParameter("configCollectionKey")
// 	if configCollectionKeyParam == "" {
// 		res.WriteErrorString(http.StatusBadRequest, "Invalid config key")
// 		return
// 	}

// 	configCollectionKey := util.ConfigKey(configCollectionKeyParam)

// 	configValues, err := r.resource.GetKeyedConfigValues(*accountId, userId, configCollectionKey)
// 	if err != nil {
// 		res.WriteErrorString(http.StatusInternalServerError, "Failed to read config values")
// 		return
// 	}

// 	res.WriteEntity(configValues)
// }

// func (r *ConfigDiffRoute) getDiffsBetweenHashesWithParams(req *restful.Request, res *restful.Response, params *config.ConfigDiffParams) {
// 	scope, accountId, userId := util.GetRequestScopeAndIds(req)
// 	if scope == util.ScopeKindInvalid {
// 		res.WriteErrorString(http.StatusForbidden, "Invalid request scope")
// 		return
// 	}

// 	if params == nil {
// 		params = &config.ConfigDiffParams{}
// 	}

// 	fromHashParam := req.PathParameter("fromHash")
// 	toHashParam := req.PathParameter("toHash")

// 	var fromHash *util.ConfigVersionHash
// 	if fromHashParam != "" {
// 		fromHash = util.ConfigVersionHashPtr(util.ConfigVersionHash(fromHashParam))
// 		params.From = &config.ConfigDiffSpec{
// 			Version: &config.ConfigVersionRef{
// 				ConfigVersionHash: *fromHash,
// 			},
// 		}
// 	}

// 	var toHash *util.ConfigVersionHash
// 	if toHashParam != "" {
// 		toHash = util.ConfigVersionHashPtr(util.ConfigVersionHash(toHashParam))
// 		params.To = &config.ConfigDiffSpec{
// 			Version: &config.ConfigVersionRef{
// 				ConfigVersionHash: *toHash,
// 			},
// 		}
// 	}

// 	versions, err := r.configService.GetDiffsWithParams(context.Background(), nil, scope, accountId, userId, params)
// 	if err != nil {
// 		res.WriteErrorString(http.StatusInternalServerError, "Failed to get versions")
// 		return
// 	}

// 	res.WriteEntity(versions)
// }

// func (r *ConfigDiffRoute) getDiffsBetweenHashesWithRecordQuery(req *restful.Request, res *restful.Response, recordQuery *config.ConfigRecordQuery) {

// 	diffParams := &config.ConfigDiffParams{
// 		IncludeObject:         true,
// 		IncludePreviousObject: true,
// 		IncludePatch:          true,
// 	}

// 	r.getDiffsBetweenHashesWithParams(req, res, diffParams)
// }

// func (r *ConfigDiffRoute) getVersionsBetweenHashes(req *restful.Request, res *restful.Response) {
// 	r.getDiffsBetweenHashesWithParams(req, res, nil)
// }

// func (r *ConfigDiffRoute) getDiffsBetweenHashes(req *restful.Request, res *restful.Response) {
// 	diffParams := &config.ConfigDiffParams{
// 		IncludeObject:         true,
// 		IncludePreviousObject: true,
// 		IncludePatch:          true,
// 	}

// 	r.getDiffsBetweenHashesWithParams(req, res, diffParams)
// }

func (r *ConfigDiffRoute) getKeyedConfigDiffs(req *restful.Request, res *restful.Response) {

	scope, accountId, userId := util.GetRequestScopeAndIds(req)
	if scope == util.ScopeKindInvalid {
		res.WriteErrorString(http.StatusForbidden, "Unauthorized request")
		return
	}

	configCollectionKeyStr := req.PathParameter("configCollectionKey")
	if configCollectionKeyStr == "" {
		res.WriteErrorString(http.StatusBadRequest, "Invalid collection key")
		return
	}

	// resourceConds := config.ConfigQueryValues{
	// 	"config_key": util.ConfigKey(configCollectionKeyStr),
	// }

	recordQuery := &config.ConfigRecordQuery{
		CollectionKey: util.ConfigCollectionKeyPtr(configCollectionKeyStr),
	}

	// r.getDiffsBetweenHashesWithRecordQuery(req, res, recordQuery)
	// r.configService.GetConfigDiffService().GetVersionChain()

	diffService := r.configService.GetConfigDiffService()

	diffParams := &config.ConfigDiffParams{
		ConfigRecordQuery:          recordQuery,
		IncludeRecordContentsPatch: true,
		IncludeObject:              true,
		OnlyMatching:               true,
	}

	versions, err := diffService.GetVersionChain(context.Background(), nil, scope, accountId, userId, diffParams)
	if err != nil {
		res.WriteErrorString(http.StatusInternalServerError, "Failed to get versions")
		return
	}

	res.WriteEntity(versions)
}

// func (r *ConfigDiffRoute) getDocumentConfigDiffs(req *restful.Request, res *restful.Response) {
// 	configCollectionKeyStr := req.PathParameter("configCollectionKey")
// 	if configCollectionKeyStr == "" {
// 		res.WriteErrorString(http.StatusBadRequest, "Invalid config document key")
// 		return
// 	}

// 	configItemKeyStr := req.PathParameter("configItemKey")
// 	if configItemKeyStr == "" {
// 		res.WriteErrorString(http.StatusBadRequest, "Invalid config document id")
// 		return
// 	}

// 	// resourceConds := config.ConfigQueryValues{
// 	// 	"config_document_key": util.ConfigDocumentKey(configCollectionKeyStr),
// 	// 	"config_document_id":  util.ConfigDocumentId(configItemKeyStr),
// 	// }

// 	recordQuery := &config.ConfigRecordQuery{
// 		CollectionKey: util.ConfigCollectionKeyPtr(configCollectionKeyStr),
// 		ItemKey:       util.ConfigItemKeyPtr(configItemKeyStr),
// 	}

// 	r.getDiffsBetweenHashesWithRecordQuery(req, res, recordQuery)
// }

// Prefixed routes
func (r *ConfigDiffRoute) Prefixed(ws *restful.WebService, prefix string) {

	// ws.Route(ws.PUT(prefix + "/configs/{configCollectionKey}").
	// 	To(r.putKeyedConfigValues).
	// 	Doc("Set values for a keyed config (only has a collection key)").
	// 	Param(ws.PathParameter("configCollectionKey", "The config key").DataType("string")).
	// 	Reads(util.Data{}).
	// 	Writes(nil))

	// ws.Route(ws.PUT(prefix + "/configs/{configCollectionKey}/{configItemKey}").
	// 	To(r.putDocumentValues).
	// 	Doc("Set values for a config document (has both a collection key and an item key)").
	// 	Param(ws.PathParameter("configCollectionKey", "The config document key").DataType("string")).
	// 	Param(ws.PathParameter("configItemKey", "The config document id").DataType("string")).
	// 	Reads(util.Data{}).
	// 	Writes(nil))

	// ws.Route(ws.GET(prefix + "/{configCollectionKey}").
	// 	To(r.getKeyedConfigValues).
	// 	Doc("Get values for a config key").
	// 	Param(ws.PathParameter("configCollectionKey", "The config key").DataType("string")).
	// 	Writes(util.Data{}))

	// ws.Route(ws.GET(prefix + "/config/diff/versions/{fromHash}/{toHash}").
	// 	To(r.getVersionsBetweenHashes).
	// 	Doc("Get config versions between two hashes").
	// 	Param(ws.PathParameter("fromHash", "The from hash").DataType("string")).
	// 	Param(ws.PathParameter("toHash", "The to hash").DataType("string")).
	// 	Writes(config.ConfigDiffVersions{}))

	// ws.Route(ws.GET(prefix + "/config/diff/{resource}/{fromHash}/{toHash}").
	// 	To(r.getDiffsBetweenHashes).
	// 	Doc("Get config diffs between two hashes").
	// 	Param(ws.PathParameter("resource", "The resource").DataType("string")).
	// 	Param(ws.PathParameter("fromHash", "The from hash").DataType("string")).
	// 	Param(ws.PathParameter("toHash", "The to hash").DataType("string")).
	// 	Writes(config.ConfigDiffVersions{}))

	ws.Route(ws.GET(prefix + "/config/diff/configs/{configCollectionKey}/{fromHash}/{toHash}").
		To(r.getKeyedConfigDiffs).
		Doc("Get config diffs between two hashes for a config key").
		Param(ws.PathParameter("configCollectionKey", "The config key").DataType("string")).
		Param(ws.PathParameter("fromHash", "The from hash").DataType("string")).
		Param(ws.PathParameter("toHash", "The to hash").DataType("string")).
		Writes(config.ConfigDiffVersions{}))

	// ws.Route(ws.GET(prefix + "/config/diff/documents/{configCollectionKey}/{configItemKey}/{fromHash}/{toHash}").
	// 	To(r.getDocumentConfigDiffs).
	// 	Doc("Get config diffs between two hashes for a config document").
	// 	Param(ws.PathParameter("configCollectionKey", "The config document key").DataType("string")).
	// 	Param(ws.PathParameter("configItemKey", "The config document id").DataType("string")).
	// 	Param(ws.PathParameter("fromHash", "The from hash").DataType("string")).
	// 	Param(ws.PathParameter("toHash", "The to hash").DataType("string")).
	// 	Writes(config.ConfigDiffVersions{}))

}
