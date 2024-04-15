package util

import (
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strings"

	restful "github.com/emicklei/go-restful/v3"
)

type QueryFilter interface{}

// func boolOrNil(s string) *bool {
// 	v, err := strconv.ParseBool(s)
// 	if err != nil {
// 		return nil
// 	}
// 	return &v
// }

var allowedChars *regexp.Regexp = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func validateQueryParam(k string, v string) error {
	log.Printf("validateQueryParam: %s = << %s >>\n", k, v)
	if !allowedChars.MatchString(v) {
		return restful.NewError(400, fmt.Sprintf("Invalid query parameter %s", k))
	}
	return nil
}

func toQueryFilterStruct(filter interface{}) (QueryFilter, error) {
	filterType := reflect.TypeOf(filter)
	filterKind := filterType.Kind()

	// Check for the pointer type first
	switch filterKind {
	case reflect.Ptr:
		// Convert to *QueryFilter
		filterPtr := reflect.ValueOf(filter).Elem()
		return toQueryFilterStruct(filterPtr.Interface())
	case reflect.Struct:
		return filter, nil
	default:
		return nil, nil
	}
}

func PopulateQueryFilter(request *restful.Request, filter interface{}) error {
	fv := reflect.ValueOf(filter)
	if fv.Kind() != reflect.Ptr {
		log.Printf("PopulateQueryFilter: filter is not a pointer\n")
		return fmt.Errorf("filter is not a pointer")
	}

	// Get the value that the pointer points to
	fv = fv.Elem()

	if fv.Kind() != reflect.Struct {
		log.Printf("PopulateQueryFilter: filter is not a struct\n")
		return fmt.Errorf("filter is not a struct")
	}

	filterType := fv.Type()

	// Loop over the fields of the struct
	for i := 0; i < filterType.NumField(); i++ {
		field := filterType.Field(i)
		fieldName := field.Name
		fieldValue := fv.Field(i)

		// Skip fields with empty names, such as embedded structs/interfaces
		if fieldName == "" {
			continue
		}

		paramName := fieldName

		// Get the json tag of the field
		jsonTag := fv.Type().Field(i).Tag.Get("json")
		if parts := strings.Split(jsonTag, ","); len(parts) > 0 && len(parts[0]) > 0 {
			paramName = parts[0]
		}

		paramValue := request.QueryParameter(paramName)
		if paramValue == "" {
			continue
		}
		// TODO: validate

		log.Printf("PopulateQueryFilter: field %s: fieldValue.Kind(): %v fieldValue.Type(): %v\n", fieldName, fieldValue.Kind(), fieldValue.Type())

		// Set the field value
		if fieldValue.Kind() == reflect.Ptr {
			fieldType := fieldValue.Type()
			elem := fieldType.Elem()

			// This allows us to assign a string to a *string (or a newtype of string)
			if elem.Kind() == reflect.String {
				log.Printf("PopulateQueryFilter: setting string %s: %s = %s\n", fieldName, paramName, paramValue)
				v := reflect.ValueOf(&paramValue)
				if v.Type().AssignableTo(fieldType) {
					fieldValue.Set(v)
				} else if v.Type().ConvertibleTo(fieldType) {
					fieldValue.Set(v.Convert(fieldType))
				} else {
					log.Printf("PopulateQueryFilter: skipping %s: %s = %s (unable to assign or convert %v to %v)\n", fieldName, paramName, paramValue, v, fieldType)
				}
			} else {
				log.Printf("PopulateQueryFilter: skipping %s: %s = %s (unsupported type %v)\n", fieldName, paramName, paramValue, fieldType)
			}
		}

		// Set the field value
		// if fieldValue.Kind() == reflect.Ptr {
		// 	targetValue := fieldValue.Elem()
		// 	log.Printf("PopulateQueryFilter: field %s: targetValue: %v\n", fieldName, targetValue)

		// 	switch targetValue.Kind() {
		// 	case reflect.String:
		// 		log.Printf("PopulateQueryFilter: setting string %s: %s = %s\n", fieldName, paramName, paramValue)

		// 		elemType := fieldValue.Type().Elem()

		// 		v := reflect.ValueOf(paramValue)
		// 		if v.Type().AssignableTo(elemType) {
		// 			fieldValue.Set(v)
		// 			if v.Type().ConvertibleTo(elemType) {
		// 				fieldValue.Set(v.Convert(fieldValue.Type().Elem()))
		// 			}
		// 		}
		// 	default:
		// 		log.Printf("PopulateQueryFilter: skipping %s: %s = %s (unsupported type %v)\n", fieldName, paramName, paramValue, fieldValue.Type())
		// 	}
		// }

	}

	log.Printf("PopulateQueryFilter: final filter: %v\n", filter)

	return nil
}

// func PopulateQueryFilter(request *restful.Request, filter interface{}) error {
// 	// filter.Limit = request.QueryParameter("limit")
// 	// filter.Offset = request.QueryParameter("offset")
// 	// filter.Sort = request.QueryParameter("sort")
// 	// filter.Order = request.QueryParameter("order")
// 	// filter.Filter = request.QueryParameter("filter")

// 	// Use reflect to get the fields of the filter struct and set them from the request

// 	filterStruct, err := toQueryFilterStruct(filter)
// 	if err != nil {
// 		log.Printf("PopulateQueryFilter: toQueryFilterStruct error: %s\n", err)
// 		return err
// 	}

// 	filterType := reflect.TypeOf(filterStruct)
// 	log.Printf("PopulateQueryFilter: filterType: %v\n", filterType)

// 	if filterType.Kind() != reflect.Struct {
// 		log.Printf("PopulateQueryFilter: filterType is not a struct\n")
// 		return fmt.Errorf("filterType is not a struct")
// 	}

// 	var queryError error

// 	for i := 0; i < filterType.NumField(); i++ {
// 		field := filterType.Field(i)
// 		fieldName := field.Name

// 		log.Printf("PopulateQueryFilter: field %d: %s\n", i, fieldName)
// 		paramName := fieldName

// 		// Use the json tag if it exists
// 		jsonTag := field.Tag.Get("json")
// 		if parts := strings.Split(jsonTag, ","); len(parts) > 0 && len(parts[0]) > 0 {
// 			paramName = parts[0]
// 		}

// 		paramValue := request.QueryParameter(paramName)
// 		log.Printf("PopulateQueryFilter: field %d: %s: %s = %s\n", i, fieldName, paramName, paramValue)

// 		if paramName == "" {
// 			log.Printf("PopulateQueryFilter: field %d: %s: skipping (no json tag, empty name)\n", i, fieldName)
// 			continue
// 		} else if paramValue == "" {
// 			log.Printf("PopulateQueryFilter: field %d: %s: skipping (no value)\n", i, fieldName)
// 			continue
// 		}

// 		if err := validateQueryParam(paramName, paramValue); err != nil {
// 			queryError = err
// 			break
// 		}

// 		log.Printf("PopulateQueryFilter: setting %s: %s = %s\n", fieldName, paramName, paramValue)
// 		// reflect.ValueOf(filterStruct).FieldByName(fieldName).SetString(paramValue)
// 		// Set field on struct
// 		// reflect.ValueOf(filterStruct).

// 		fieldValue := reflect.ValueOf(filterStruct).FieldByName(fieldName)
// 		fieldValueType := fieldValue.Type()
// 		log.Printf("PopulateQueryFilter: field %d: %s: fieldValue: %v fieldValueType: %v\n", i, fieldName, fieldValue, fieldValueType)

// 		log.Printf("PopulateQueryFilter: field %d: %s: IsValid() = %v CanSet() = %v CanAddr() = %v IsPtr() = %v\n", i, fieldName, fieldValue.IsValid(), fieldValue.CanSet(), fieldValue.CanAddr(), fieldValue.Kind() == reflect.Ptr)

// 		isPtr := fieldValue.CanAddr() && fieldValue.Kind() == reflect.Ptr
// 		isStringPtr := isPtr && fieldValue.Elem().Kind() == reflect.String

// 		log.Printf("PopulateQueryFilter: field %s: kind: %v type: %v\n", fieldName, fieldValue.Kind(), fieldValue.Type())
// 		if fieldValue.Kind() == reflect.Ptr {
// 			elem := fieldValue.Elem()
// 			if elem != reflect.Zero(elem.Type()) {
// 				log.Printf("PopulateQueryFilter: field %s is pointer to: kind: %v type: %v\n", fieldName, elem.Kind(), elem.Type())
// 			}
// 		}

// 		if isStringPtr {
// 			log.Printf("Setting string pointer for %s: %s\n", fieldName, paramValue)
// 			elem := fieldValue.Elem()
// 			v := paramValue
// 			elem.Set(reflect.ValueOf(&v))
// 		}

// 		// if fieldValue.IsValid() && fieldValue.CanSet() {
// 		// 	fieldValue.SetString(paramValue)
// 		// } else if fieldValue.CanAddr() && fieldValue.Kind() == reflect.Ptr {
// 		// 	addr := fieldValue.Addr()
// 		// 	v := paramValue
// 		// 	addr.Elem().Set(reflect.ValueOf(&v))
// 		// } else {
// 		// 	log.Printf("PopulateQueryFilter: field %d: %s: skipping (invalid or cannot set)\n", i, fieldName)
// 		// }
// 	}

// 	if queryError != nil {
// 		return queryError
// 	}

// 	return nil
// }
