package routes

import (
	"context"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/tmzt/config-api/config"
	"github.com/tmzt/config-api/util"
)

type ConfigRoute struct {
	logger        util.SetRequestLogger
	configService *config.ConfigService
}

func NewConfigRoute(resource *config.ConfigService) *ConfigRoute {
	logger := util.NewLogger("ConfigRoute", 0)

	return &ConfigRoute{
		logger:        logger,
		configService: resource,
	}
}

func getRecordQuery(req *restful.Request, res *restful.Response, requireCollectionKey bool, requireItemKey bool, requireVersionHash bool) *config.ConfigRecordQuery {
	logger := util.NewLogger("config.getRecordQuery", 0)
	logger.SetRequest(req)

	scope, accountId, userId := util.GetRequestScopeAndIds(req)
	if scope == util.ScopeKindInvalid {
		res.WriteErrorString(http.StatusForbidden, "Invalid account id")
		return nil
	}

	var collectionKey *util.ConfigCollectionKey
	if v := req.PathParameter("collectionKey"); true {
		if v != "" {
			collectionKey = util.ConfigCollectionKeyPtr(v)
		} else if requireCollectionKey {
			logger.Printf("Invalid config collection key\n")
			res.WriteErrorString(http.StatusBadRequest, "Invalid config collection key")
			return nil
		}
	}

	var itemKey *util.ConfigItemKey
	if v := req.PathParameter("itemKey"); true {
		if v != "" {
			itemKey = util.ConfigItemKeyPtr(v)
		} else if requireItemKey {
			logger.Printf("Invalid config item key\n")
			res.WriteErrorString(http.StatusBadRequest, "Invalid config item key")
			return nil
		}
	}

	var versionHash *util.ConfigVersionHash
	if v := req.QueryParameter("configVersionHash"); true {
		if v != "" {
			hash := util.ConfigVersionHash(v)
			versionHash = &hash
		} else if requireVersionHash {
			res.WriteErrorString(http.StatusBadRequest, "Invalid config version hash")
			return nil
		}
	}

	query := &config.ConfigRecordQuery{
		Scope:             &scope,
		AccountId:         &accountId,
		UserId:            &userId,
		CollectionKey:     collectionKey,
		ItemKey:           itemKey,
		ConfigVersionHash: versionHash,
	}

	return query
}

func (r *ConfigRoute) getRecordList(req *restful.Request, res *restful.Response) {

	scope, accountId, userId := util.GetRequestScopeAndIds(req)
	if scope == util.ScopeKindInvalid {
		res.WriteErrorString(http.StatusForbidden, "Invalid account id")
		return
	}

	// TBD: should we support other parameters for the record query?

	recordList, err := r.configService.ListConfigs(context.Background(), nil, scope, accountId, userId, nil)
	if err != nil {
		r.logger.Printf("Failed to list configs: %v\n", err)
		res.WriteErrorString(http.StatusInternalServerError, "Failed to list configs")
		return
	}

	r.logger.Printf("Record list: %v\n", recordList)

	// Add Content-Range header
	res.Header().Set("Content-Range", fmt.Sprintf("configs 0-%d/%d", len(recordList), len(recordList)))

	res.WriteEntity(recordList)
}

type configRecordValues struct {
	Data *util.Data `json:"data"`
}

type configDocumentInput struct {
	Data *configRecordValues `json:"data"`
}

type uiMetadata interface{}

type configRecordCreateInput struct {
	Data           *configRecordValues          `json:"data"`
	RecordMetadata *config.ConfigRecordMetadata `json:"record_metadata"`
	UiMetadata     *uiMetadata                  `json:"ui_metadata"`
}

type configRecordResponse struct {
	Id             string                                  `json:"id"`
	Data           *configRecordValues                     `json:"data"`
	RecordHistory  []*config.ConfigDiffVersionHistoryEntry `json:"record_history"`
	RecordMetadata *config.ConfigRecordMetadata            `json:"record_metadata"`
	NodeMetadata   *config.ConfigNodeMetadata              `json:"node_metadata"`
}

func (r *ConfigRoute) getRecordValues(req *restful.Request, res *restful.Response, withCollectionKey bool, withItemKey bool, withConfigVersion bool) {

	scope, accountId, userId := util.GetRequestScopeAndIds(req)
	if scope == util.ScopeKindInvalid {
		res.WriteErrorString(http.StatusForbidden, "Invalid account id")
		return
	}

	recordQuery := getRecordQuery(req, res, withCollectionKey, withItemKey, withConfigVersion)
	if recordQuery == nil {
		return
	}

	// TODO: If version hash is provided, use it to get the record

	version, err := r.configService.GetLatestRecord(context.Background(), nil, scope, accountId, userId, nil, nil, recordQuery)
	if err != nil {
		r.logger.Printf("Failed to get latest record: %v\n", err)
		res.WriteErrorString(http.StatusInternalServerError, "Failed to get latest record")
		return
	} else if version == nil {
		r.logger.Printf("Record not found\n")
		res.WriteErrorString(http.StatusNotFound, "Record not found")
		return
	}

	// values := node.GetRecordContents()
	// if values == nil {
	// 	r.logger.Printf("Failed to get record contents\n")
	// 	res.WriteErrorString(http.StatusInternalServerError, "Failed to get record contents")
	// 	return
	// }

	values := version.RecordContents

	// Add headers
	// version := node.GetVersionRef()
	// res.Header().Set("X-Config-Record-Id", string(version.ConfigRecordId))
	// res.Header().Set("X-Config-Version-Id", string(version.ConfigVersionId))
	// res.Header().Set("X-Config-Version-Hash", string(version.ConfigVersionHash))

	res.Header().Set("X-Config-Record-Hash", string(version.RecordMetadata.RecordId))
	res.Header().Set("X-Config-Version-Hash", string(version.ToVersion.ConfigVersionHash))

	r.logger.Printf("Record metadata: %+v\n", version.RecordMetadata)

	recordKey := string(version.RecordMetadata.CollectionKey)
	if version.RecordMetadata.ItemKey != nil {
		recordKey += "/" + string(*version.RecordMetadata.ItemKey)
	}

	r.logger.Printf("Record key: %s\n", recordKey)

	doc := &configRecordResponse{
		Id:             recordKey,
		Data:           &configRecordValues{Data: values},
		RecordHistory:  version.RecordHistory,
		RecordMetadata: version.RecordMetadata,
	}

	res.WriteEntity(doc)
}

func (r *ConfigRoute) setRecordValuesByPath(req *restful.Request, res *restful.Response, withCollectionKey bool, withItemKey bool) {

	scope, _, _ := util.GetRequestScopeAndIds(req)
	if scope == util.ScopeKindInvalid {
		res.WriteErrorString(http.StatusForbidden, "Invalid account id")
		return
	}

	recordQuery := getRecordQuery(req, res, withCollectionKey, withItemKey, false)
	if recordQuery == nil {
		return
	} else if recordQuery.ConfigVersionHash != nil {
		res.WriteErrorString(http.StatusBadRequest, "Cannot set values by version hash")
		return
	}

	recordMetadata := recordQuery.AsMetadata()
	if recordMetadata == nil {
		res.WriteErrorString(http.StatusInternalServerError, "Internal server error")
		return
	}

	// Read Data from request
	// configInput := &util.Data{}
	configInput := &configDocumentInput{}

	if err := req.ReadEntity(configInput); err != nil {
		res.WriteErrorString(http.StatusBadRequest, "Invalid config values")
		return
	}
}

func (r *ConfigRoute) setRecordValuesFromPost(req *restful.Request, res *restful.Response, kind config.ConfigRecordKind) {
	// r.setRecordValuesByPath(req, res, true)

	scope, accountId, userId := util.GetRequestScopeAndIds(req)
	if scope == util.ScopeKindInvalid {
		res.WriteErrorString(http.StatusForbidden, "Invalid account id")
		return
	}

	input := &configRecordCreateInput{}
	if err := req.ReadEntity(input); err != nil {
		res.WriteErrorString(http.StatusBadRequest, "Invalid request")
		return
	}

	r.logger.Printf("Input data: %s\n", util.ToJsonPretty(input.Data))

	if input.Data == nil || input.Data.Data == nil {
		res.WriteErrorString(http.StatusBadRequest, "Invalid data (data.data)")
		return
	}

	if input.RecordMetadata == nil {
		res.WriteErrorString(http.StatusBadRequest, "Invalid record metadata (record_metadata)")
		return
	}

	if input.RecordMetadata.CollectionKey == "" {
		res.WriteErrorString(http.StatusBadRequest, "Invalid record metadata (record_metadata.collection_key is required)")
		return
	}

	if kind == config.ConfigRecordKindDocument && input.RecordMetadata.ItemKey == nil {
		res.WriteErrorString(http.StatusBadRequest, "Invalid record metadata (record_metadata.item_key is required for document records)")
		return
	}

	recordMetadata := input.RecordMetadata

	r.setRecordValues(req, res, accountId, userId, input.Data.Data, recordMetadata)
}

func (r *ConfigRoute) setRecordValues(req *restful.Request, res *restful.Response, accountId util.AccountId, userId util.UserId, data *util.Data, recordMetadata *config.ConfigRecordMetadata) {

	// TODO: See if there's a better context to use from the request
	ctx := context.Background()

	var inputValues *util.Data

	if data != nil {
		inputValues = data
	} else {
		r.logger.Printf("No data found in input (data.data)\n")
		res.WriteErrorString(http.StatusBadRequest, "No data found in input (data.data)")
		return
	}

	r.logger.Printf("Input values: %+v\n", inputValues)

	newNode, err := r.configService.SetRecordValues(ctx, nil, util.ScopeKindAccount, accountId, userId, config.ConfigRecordKindKeyed, recordMetadata, config.ValueSettingModeReplace, inputValues)
	if err != nil {
		r.logger.Printf("Failed to write config values: %v\n", err)
		res.WriteErrorString(http.StatusInternalServerError, "Failed to write config values")
		return
	}

	// newVersion := newNode.GetVersionRef()
	newVersion := newNode.VersionRef

	// TODO: Also set ETag
	// res.Header().Set("X-Config-Version-Id", string(newVersion.ConfigVersionId))
	res.Header().Set("X-Config-Version-Hash", string(newVersion.ConfigVersionHash))

	// res.WriteHeader(http.StatusNoContent)

	recordKey := string(recordMetadata.CollectionKey)
	if recordMetadata.ItemKey != nil {
		recordKey += "/" + string(*recordMetadata.ItemKey)
	}

	output := &configRecordResponse{
		Id:   recordKey,
		Data: &configRecordValues{Data: inputValues},
	}

	res.WriteHeaderAndEntity(http.StatusCreated, output)
}

func (r *ConfigRoute) postKeyedConfigValues(req *restful.Request, res *restful.Response) {
	r.setRecordValuesFromPost(req, res, config.ConfigRecordKindKeyed)
}

func (r *ConfigRoute) putKeyedConfigValues(req *restful.Request, res *restful.Response) {
	r.setRecordValuesByPath(req, res, true, false)
}

func (r *ConfigRoute) getKeyedConfigValues(req *restful.Request, res *restful.Response) {
	r.getRecordValues(req, res, true, false, false)
}

func (r *ConfigRoute) postDocumentValues(req *restful.Request, res *restful.Response) {
	r.setRecordValuesFromPost(req, res, config.ConfigRecordKindDocument)
}

func (r *ConfigRoute) putDocumentValues(req *restful.Request, res *restful.Response) {
	r.setRecordValuesByPath(req, res, true, true)
}

func (r *ConfigRoute) getDocumentValues(req *restful.Request, res *restful.Response) {
	r.getRecordValues(req, res, true, true, false)
}

// Prefixed routes
func (r *ConfigRoute) Prefixed(ws *restful.WebService, prefix string) {

	ws.Route(ws.GET(prefix + "/configs").
		To(r.getRecordList).
		Doc("List all config records").
		Writes([]config.ConfigListEntry{}))

	ws.Route(ws.POST(prefix + "/configs").
		To(r.postKeyedConfigValues).
		Doc("Create a new keyed config (has only a collection key)").
		Reads(configRecordCreateInput{}).
		Writes(configRecordResponse{}))

	ws.Route(ws.PUT(prefix + "/configs/{collectionKey}").
		To(r.putKeyedConfigValues).
		Doc("Set values for a keyed config (only has a collection key)").
		Param(ws.PathParameter("collectionKey", "The config key").DataType("string")).
		Reads(util.Data{}).
		Writes(nil))

	ws.Route(ws.GET(prefix + "/configs/{collectionKey}").
		To(r.getKeyedConfigValues).
		Doc("Get values for a keyed config (only has a collection key)").
		Param(ws.PathParameter("collectionKey", "The config key").DataType("string")).
		Writes(util.Data{}))

	ws.Route(ws.POST(prefix + "/configs/{collectionKey}").
		To(r.postDocumentValues).
		Doc("Create a new config document (has both a collection key and an item key)").
		Param(ws.PathParameter("collectionKey", "The config document key").DataType("string")).
		Reads(configRecordCreateInput{}).
		Writes(configRecordResponse{}))

	ws.Route(ws.PUT(prefix + "/configs/{collectionKey}/{itemKey}").
		To(r.putDocumentValues).
		Doc("Set values for a config document (has both a collection key and an item key)").
		Param(ws.PathParameter("collectionKey", "The config document key").DataType("string")).
		Param(ws.PathParameter("itemKey", "The config document id").DataType("string")).
		Reads(util.Data{}).
		Writes(nil))

	ws.Route(ws.GET(prefix + "/configs/{collectionKey}/{itemKey}").
		To(r.getDocumentValues).
		Doc("Get values for a config document (has both a collection key and an item key)").
		Param(ws.PathParameter("collectionKey", "The config document key").DataType("string")).
		Param(ws.PathParameter("itemKey", "The config document id").DataType("string")).
		Writes(util.Data{}))

}
