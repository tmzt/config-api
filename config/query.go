package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tmzt/config-api/util"
	"gorm.io/gorm"
)

type ConfigQuerier interface {
	Where(cond interface{}, params ...interface{}) ConfigQuerier
}

type ConfigQueryFunc func(querier ConfigQuerier) ConfigQuerier

func ConfigRefQueryFunc(versionRef ConfigVersionRef) ConfigQueryFunc {
	return func(querier ConfigQuerier) ConfigQuerier {
		logger := util.NewLogger("ConfigRefQueryFunc", 0)
		logger.Printf("ConfigRefQueryFunc: versionRef.config_version_hash: %v\n", versionRef.ConfigVersionHash)
		return querier.Where("node_metadata->'version_ref'->>'config_version_hash = ?", versionRef.ConfigVersionHash)
	}
}

func ConfigRecordQueryFunc(recordQuery ConfigRecordQuery) ConfigQueryFunc {
	return func(querier ConfigQuerier) ConfigQuerier {
		logger := util.NewLogger("ConfigRecordQueryFunc", 0)
		logger.Printf("ConfigRecordQueryFunc: recordQuery: %v\n", recordQuery)

		if recordQuery.RecordId != nil {
			querier = querier.Where("(node_contents->'record_metadata'->>'record_id') = ?", *recordQuery.RecordId)
		}
		if recordQuery.CollectionKey != nil {
			querier = querier.Where("(node_contents->'record_metadata'->>'collection_key') = ?", *recordQuery.CollectionKey)
		}
		if recordQuery.ItemKey != nil {
			querier = querier.Where("(node_contents->'record_metadata'->>'item_key') = ?", *recordQuery.ItemKey)
		}
		return querier
	}
}

type ConfigGormQuerier struct {
	querier *gorm.DB
}

func (q *ConfigGormQuerier) Where(cond interface{}, params ...interface{}) ConfigQuerier {
	q.querier = q.querier.Where(cond, params...)
	return q
}

func (q *ConfigGormQuerier) Querier() *gorm.DB {
	return q.querier
}

type ConfigWhereQuery struct {
	conds []string
}

func (q *ConfigWhereQuery) formatCond(cond string, params ...interface{}) string {
	logger := util.NewLogger("ConfigWhereQuery.formatCond", 0)

	condFmt := strings.ReplaceAll(cond, "?", "%s")
	strParams := make([]interface{}, len(params))
	for i, param := range params {
		logger.Printf("ConfigWhereQuery.formatCond: param[%T]: %v\n", param, param)

		if intParam, ok := param.(int); ok {
			strParams[i] = strconv.Itoa(intParam)
		} else if int64Param, ok := param.(int64); ok {
			strParams[i] = strconv.FormatInt(int64Param, 10)
		} else if strParam, ok := param.(string); ok {
			strParams[i] = fmt.Sprintf("'%s'", strParam)
		} else if boolParam, ok := param.(bool); ok {
			if boolParam {
				strParams[i] = "true"
			} else {
				strParams[i] = "false"
			}
		} else {
			// Probably a string alias type
			strParams[i] = fmt.Sprintf("'%v'", param)
		}
	}
	return fmt.Sprintf(condFmt, strParams...)
}

func (q *ConfigWhereQuery) Where(cond interface{}, params ...interface{}) ConfigQuerier {
	whereCond := q.formatCond(cond.(string), params...)

	logger := util.NewLogger("ConfigWhereQuery.Where", 0)
	logger.Printf("ConfigWhereQuery.Where: whereCond: %v\n", whereCond)
	q.conds = append(q.conds, whereCond)
	logger.Printf("ConfigWhereQuery.Where: q.conds: %v\n", q.conds)

	return q
}

func (q *ConfigWhereQuery) SqlAlias(alias string) string {
	conds := make([]string, len(q.conds))
	for i, cond := range q.conds {
		// Handle a very simple expression
		if strings.HasPrefix(cond, "(") {
			conds[i] = strings.Replace(cond, "(", fmt.Sprintf("(%s.", alias), 1)
		} else {
			conds[i] = fmt.Sprintf("%s.%s", alias, cond)
		}
	}
	return strings.Join(conds, " AND ")
}

func (q *ConfigWhereQuery) Sql() string {
	return strings.Join(q.conds, " AND ")
}
