package routes

import (
	"crypto/sha256"
	"fmt"
	"net/http"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/tmzt/config-api/config"
	"github.com/tmzt/config-api/util"
)

type ConfigSchemaRoute struct {
	logger        util.SetRequestLogger
	configService *config.ConfigService
	apiBaseUrl    string
}

func NewConfigSchemaRoute(resource *config.ConfigService) *ConfigSchemaRoute {
	logger := util.NewLogger("ConfigSchemaRoute", 0)

	apiBaseUrl := util.MustGetPublicApiBaseUrl()

	return &ConfigSchemaRoute{
		logger:        logger,
		configService: resource,
		apiBaseUrl:    apiBaseUrl,
	}
}

// func (route *ConfigSchemaRoute) GetConfigSchemaVersions(ctx context.Context, tx *gorm.DB, req *restful.Request, res *restful.Response, scope util.ScopeKind, accountId util.AccountId, userId util.UserId) (*[]config.ConfigSchemaRecord, error) {
// 	route.logger.SetRequest(req)

// 	configSchemaList, err := route.configService.GetConfigSchemaList()
// 	if err != nil {
// 		route.logger.Error("GetConfigSchemaList", err)
// 		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	ctx.JSON(http.StatusOK, configSchemaList)
// }

type configSchemaInput struct {
	RecordMetadata *config.ConfigRecordMetadata `json:"record_metadata"`
	Schema         *config.ConfigSchemaRecord   `json:"schema"`
}

type configSchemaOutput struct {
	Id             *util.ConfigVersionHash      `json:"id"`
	RecordMetadata *config.ConfigRecordMetadata `json:"record_metadata"`
	Schema         *config.ConfigSchemaRecord   `json:"schema"`
}

func (r *ConfigSchemaRoute) postConfigSchema(req *restful.Request, res *restful.Response) {
	// route.logger.SetRequest(req)
	r.logger.Printf("postConfigSchema called\n")

	scope, accountId, userId := util.GetRequestScopeAndIds(req)
	if scope == util.ScopeKindInvalid {
		res.WriteErrorString(http.StatusForbidden, "Invalid account id")
		return
	}

	input := &configSchemaInput{}
	if err := req.ReadEntity(input); err != nil {
		r.logger.Printf("Error reading entity: %v\n", err)
		res.WriteErrorString(http.StatusBadRequest, "Invalid request")
		return
	}

	ctx := req.Request.Context()
	node, err := r.configService.InsertRecord(ctx, nil, scope, accountId, userId, input.RecordMetadata, input.Schema)
	if err != nil {
		res.WriteErrorString(http.StatusInternalServerError, err.Error())
		return
	}

	schemaHash := node.VersionRef.ConfigVersionHash

	schemaOut := &configSchemaOutput{
		Id:     &schemaHash,
		Schema: input.Schema,
	}

	schemaUrl := fmt.Sprintf("%s/global/config/schemas/%s", r.apiBaseUrl, schemaHash)

	res.Header().Add("Location", schemaUrl)

	res.WriteHeaderAndEntity(http.StatusCreated, schemaOut)
}

func (r *ConfigSchemaRoute) getAllSchemas(req *restful.Request, res *restful.Response) {
	r.logger.SetRequest(req)

	scope, accountId, userId := util.GetRequestScopeAndIds(req)
	if scope == util.ScopeKindInvalid {
		res.WriteErrorString(http.StatusForbidden, "Invalid account id")
		return
	}

	hasher := sha256.New()

	recordQuery := &config.ConfigRecordQuery{}
	recordQuery.RecordKind = config.ConfigRecordKindAsPtr(config.ConfigRecordKindConfigSchema)

	ctx := req.Request.Context()

	diffService := r.configService.GetConfigDiffService()
	diffParams := &config.ConfigDiffParams{
		ConfigRecordQuery: recordQuery,
		IncludeObject:     true,
		IncludeRecord:     true,
		OnlyMatching:      true,
	}
	versions, err := diffService.GetVersionChain(ctx, nil, scope, accountId, userId, diffParams)
	if err != nil {
		r.logger.Printf("getAllSchemas: Error getting record versions: %v\n", err)
		res.WriteErrorString(http.StatusInternalServerError, err.Error())
		return
	}

	schemas := []*config.ConfigSchemaRecord{}

	if versions != nil {
		for _, version := range versions.Versions {
			// r.logger.Printf("getAllSchemas: version: %s\n", util.ToJsonPretty(version))
			// r.logger.Printf("getAllSchemas: node contents: %s\n", util.ToJsonPretty(version.NodeContents))
			r.logger.Printf("getAllSchemas: record contents: %s\n", util.ToJsonPretty(version.RecordContents))
			schema := &config.ConfigSchemaRecord{}
			if err := util.FromDataMap(version.RecordContents, schema); err != nil {
				r.logger.Printf("getAllSchemas: Error decoding schema: %v\n", err)
				continue
			}
			r.logger.Printf("getAllSchemas: versions.ToVersion: %s\n", util.ToJsonPretty(versions.ToVersion))
			r.logger.Printf("getAllSchemas: versions.FromVersion: %s\n", util.ToJsonPretty(versions.FromVersion))
			if v := version.ToVersion; v != nil {
				schema.SchemaHash = &v.ConfigVersionHash

				// Update the overall hash
				fmt.Fprintf(hasher, "%s", v.ConfigVersionHash)
			}
			r.logger.Printf("getAllSchemas: schema: %s\n", util.ToJsonPretty(schema))
			schemas = append(schemas, schema)

		}
	}

	listHash := fmt.Sprintf("%x", hasher.Sum(nil))
	r.logger.Printf("getAllSchemas: listHash: %s\n", listHash)
	res.Header().Add("X-Content-Hash", listHash)

	res.Header().Add("Content-Type", "application/json")
	res.Header().Set("Content-Range", fmt.Sprintf("schemas 0-%d/%d", len(schemas)-1, len(schemas)))
	res.WriteHeaderAndEntity(http.StatusOK, schemas)
}

func (r *ConfigSchemaRoute) getSchemaQuery(req *restful.Request, res *restful.Response, requireCollectionKey bool, requireItemKey bool, requireSchemaHash bool) *config.ConfigRecordQuery {
	recordQuery := getRecordQuery(req, res, requireCollectionKey, requireItemKey, requireSchemaHash)
	if recordQuery == nil {
		r.logger.Printf("Invalid record query")
		return nil
	}
	recordQuery.RecordKind = config.ConfigRecordKindAsPtr(config.ConfigRecordKindConfigSchema)

	return recordQuery
}

func (r *ConfigSchemaRoute) getSchema(req *restful.Request, res *restful.Response, requireCollectionKey bool, requireItemKey bool, requireSchemaHash bool) {
	r.logger.SetRequest(req)

	query := r.getSchemaQuery(req, res, requireCollectionKey, requireItemKey, requireSchemaHash)
	if query == nil {
		r.logger.Printf("Invalid record query\n")
		res.WriteErrorString(http.StatusBadRequest, "Invalid record query")
		return
	}

	ctx := req.Request.Context()
	// schema, err := r.configSchemaService.GetSchema(ctx, nil, query)
	var err error
	var schema *config.ConfigSchemaRecord

	schemaService := r.configService.GetConfigSchemaService()

	if query.ConfigVersionHash == nil {
		r.logger.Printf("Getting latest schema version\n")
		schema, err = schemaService.GetLatestSchemaVersion(ctx, nil, query)
	} else {
		r.logger.Printf("Getting schema by hash\n")
		schema, err = schemaService.GetSchema(ctx, nil, query)
	}

	if err != nil {
		r.logger.Printf("Error getting schema: %v\n", err)
		res.WriteErrorString(http.StatusInternalServerError, "Internal server error getting schema")
		return
	} else if schema == nil {
		r.logger.Printf("Schema not found\n")
		res.WriteErrorString(http.StatusNotFound, "Schema not found")
		return
	}

	r.logger.Printf("Schema name: %s\n", util.ToJsonPretty(schema.SchemaName))
	r.logger.Printf("Schema hash: %s\n", util.ToJsonPretty(schema.SchemaHash))
	r.logger.Printf("Schema id: %s\n", util.ToJsonPretty(schema.SchemaIdValue))

	schemaOut := &configSchemaOutput{
		Id:     schema.SchemaHash,
		Schema: schema,
	}

	// TODO: Handle scopes
	if schemaOut.Id != nil {
		schemaUrl := fmt.Sprintf("%s/global/config/schemas/%s", r.apiBaseUrl, *schemaOut.Id)
		res.Header().Add("Location", schemaUrl)
	}

	res.WriteHeaderAndEntity(http.StatusOK, schemaOut)
}

// func (r *ConfigSchemaRoute) getSchemaByHash(req *restful.Request, res *restful.Response) {
// 	r.getSchema(req, res, false, false, true)
// }

// func (r *ConfigSchemaRoute) getLatestSchemaForPath(req *restful.Request, res *restful.Response) {
// 	r.getSchema(req, res, true, false, false)
// }

// func (r *ConfigSchemaRoute) getSchemaForPathAndHash(req *restful.Request, res *restful.Response) {
// 	r.getSchema(req, res, true, false, true)
// }

// Prefixed routes
func (r *ConfigSchemaRoute) Prefixed(ws *restful.WebService, prefix string) {

	ws.Route(ws.POST(prefix + "/schemas").To(r.postConfigSchema).
		Doc("Create a new config schema").
		Operation("postConfigSchema").
		Reads(configSchemaInput{}).
		Writes(config.ConfigSchemaRecord{}))

	ws.Route(ws.GET(prefix + "/schemas").To(r.getAllSchemas).
		Doc("Get all config schemas").
		Operation("getAllSchemas").
		Writes([]config.ConfigSchemaRecord{}))

	ws.Route(ws.GET(prefix + "/schemas/{collectionKey}/{itemKey}").To(
		func(req *restful.Request, res *restful.Response) {
			r.getSchema(req, res, true, true, false)
		}).
		Doc("Get the latest config schema for a path").
		Operation("getLatestSchemaForPath").
		Param(ws.PathParameter("collectionKey", "collection key").DataType("string")).
		Param(ws.PathParameter("itemKey", "item key").DataType("string")).
		Writes(config.ConfigSchemaRecord{}))

	ws.Route(ws.GET(prefix + "/schemas/{collectionKey}").To(
		func(req *restful.Request, res *restful.Response) {
			r.getSchema(req, res, true, false, false)
		}).
		Doc("Get the latest config schema for a path (keyed schema)").
		Operation("getLatestSchemaForPath").
		Param(ws.PathParameter("collectionKey", "collection key").DataType("string")).
		Writes(config.ConfigSchemaRecord{}))

	ws.Route(ws.GET(prefix + "/schemas/{collectionKey}/{itemKey}/{configVersionHash}").To(
		func(req *restful.Request, res *restful.Response) {
			r.getSchema(req, res, true, true, true)
		}).
		Doc("Get the config schema for a path and version hash").
		Operation("getLatestSchemaForPath").
		Param(ws.PathParameter("collectionKey", "collection key").DataType("string")).
		Param(ws.PathParameter("itemKey", "item key").DataType("string")).
		Writes(config.ConfigSchemaRecord{}))

	ws.Route(ws.GET(prefix + "/schemas/{collectionKey}/_/{configVersionHash}").To(
		func(req *restful.Request, res *restful.Response) {
			r.getSchema(req, res, true, false, false)
		}).
		Doc("Get the config schema for a path and version hash (keyed schema)").
		Operation("getLatestSchemaForPath").
		Param(ws.PathParameter("collectionKey", "collection key").DataType("string")).
		Writes(config.ConfigSchemaRecord{}))

	ws.Route(ws.GET(prefix + "/schema_version/{configVersionHash}").
		To(func(req *restful.Request, res *restful.Response) {
			r.getSchema(req, res, false, false, true)
		}).
		Doc("Get a config schema for a specific version hash").
		Operation("getSchema").
		Param(ws.PathParameter("configVersionHash", "identifier of the schema (the ConfigVersionHash of the record node)").DataType("string")).
		Writes(config.ConfigSchemaRecord{}))

}
