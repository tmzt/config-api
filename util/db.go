package util

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"

	"gorm.io/gorm"
)

func WithTransaction(db *gorm.DB, tx *gorm.DB, fn func(tx *gorm.DB) error) error {
	if tx != nil {
		return fn(tx)
	} else {
		return db.Transaction(fn)
	}
}

var sqlParamRegex = regexp.MustCompile(`(\$[0-9]|\?)`)

func FormatDebugQuery(query string, args ...interface{}) string {
	argsAny := make([]interface{}, len(args))
	for i, arg := range args {
		// typ := reflect.TypeOf(arg)

		if arg == nil {
			argsAny[i] = "NULL"
		} else if n, ok := arg.(int); ok {
			argsAny[i] = n
		} else if n, ok := arg.(int64); ok {
			argsAny[i] = n
		} else if n, ok := arg.(uint); ok {
			argsAny[i] = n
		} else if n, ok := arg.(uint64); ok {
			argsAny[i] = n
		} else if n, ok := arg.(float64); ok {
			argsAny[i] = n
		} else if n, ok := arg.(bool); ok {
			argsAny[i] = n
		} else if n, ok := arg.(string); ok {
			argsAny[i] = fmt.Sprintf("'%v'", n)
		} else if n, ok := arg.(*string); ok {
			if n == nil {
				argsAny[i] = "NULL"
			} else {
				argsAny[i] = fmt.Sprintf("'%v'", *n)
			}
		} else if reflect.TypeOf(arg).Kind() == reflect.String {
			argsAny[i] = fmt.Sprintf("'%v'", arg)
		} else if reflect.TypeOf(arg).Kind() == reflect.Slice {
			argsAny[i] = fmt.Sprintf("'%v'::JSONB", ToJson(arg))
		} else if reflect.TypeOf(arg).Kind() == reflect.Map {
			argsAny[i] = fmt.Sprintf("'%v'::JSONB", ToJson(arg))
		} else {
			argsAny[i] = fmt.Sprintf("'%v'::JSONB", ToJson(arg))
		}
	}
	// queryFmt := strings.ReplaceAll(query, "?", "%s")
	queryFmt := sqlParamRegex.ReplaceAllString(query, "%v")
	return fmt.Sprintf("\n\n"+queryFmt+";\n\n", argsAny...)
}

func ScanSingleValueInto(rs *gorm.DB, dest interface{}) error {
	logger := NewLogger("ScanValueInto", 0)

	if rs.Error != nil {
		logger.Printf("util.ScanValueInto: rs.Error: %v", rs.Error)
		return rs.Error
	}

	// if rs.RowsAffected == 0 {
	// 	logger.Printf("util.ScanValueInto: no rows affected\n")
	// 	return gorm.ErrRecordNotFound
	// }

	rows, err := rs.Rows()
	if err != nil {
		logger.Printf("util.ScanValueInto: rs.Rows() error: %v", err)
		return err
	}

	defer rows.Close()

	if !rows.Next() {
		logger.Printf("util.ScanValueInto: unable to get next row\n")
		return gorm.ErrRecordNotFound
	}

	buf := []byte{}

	if err := rows.Scan(&buf); err != nil {
		logger.Printf("util.ScanValueInto: rs.Scan error: %v", err)
		return err
	}

	logger.Printf("util.ScanValueInto: buf(str): %s\n", string(buf))

	if err := json.Unmarshal(buf, dest); err != nil {
		logger.Printf("util.ScanValueInto: json.Unmarshal error: %v", err)
		return err
	}

	return nil
}

func RawGetJsonValue(ctx context.Context, db *gorm.DB, tx *gorm.DB, dest interface{}, query string, args ...interface{}) error {
	logger := NewLogger("RawGetJsonValue", 0)

	if db == nil {
		logger.Printf("util.RawGetJsonValue: db is nil\n")
		return fmt.Errorf("db is nil")
	}

	logger.Printf("util.RawGetJsonValue: query: %s\n", FormatDebugQuery(query, args...))

	return WithTransaction(db, tx, func(tx *gorm.DB) error {

		rs := db.Raw(query, args...)
		if rs.Error != nil {
			logger.Printf("util.RawGetJsonValue: rs.Error: %v", rs.Error)
			return rs.Error
		}

		return ScanSingleValueInto(rs, dest)
	})
}
