package env

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrNotAStructPtr is returned if you pass something that is not a pointer to a
	// Struct to Parse
	ErrNotAStructPtr = errors.New("Expected a pointer to a Struct")
	// ErrUnsupportedType if the struct field type is not supported by env
	ErrUnsupportedType = errors.New("Type is not supported")
	// ErrMismatchType if the env value can't be assigned. For instance when try to assign an env value
	// to a struct (The env values MUST be assigned to the struct fields).
	ErrMismatchType = errors.New("Data Type mismatch. Mismatching left and right hand types")
	// ErrUnsupportedSliceType if the slice element type is not supported by env
	ErrUnsupportedSliceType = errors.New("Unsupported slice type")
	// Friendly names for reflect types
	sliceOfInts     = reflect.TypeOf([]int(nil))
	sliceOfInt64s   = reflect.TypeOf([]int64(nil))
	sliceOfStrings  = reflect.TypeOf([]string(nil))
	sliceOfBools    = reflect.TypeOf([]bool(nil))
	sliceOfFloat32s = reflect.TypeOf([]float32(nil))
	sliceOfFloat64s = reflect.TypeOf([]float64(nil))
)

// CustomParsers is a friendly name for the type that `ParseWithFuncs()` accepts
type CustomParsers map[reflect.Type]ParserFunc

// ParserFunc defines the signature of a function that can be used within `CustomParsers`
type ParserFunc func(v string) (interface{}, error)

// Parse parses a struct containing `env` tags and loads its values from
// environment variables.
func Parse(v interface{}) error {
	ptrRef := reflect.ValueOf(v)
	if ptrRef.Kind() != reflect.Ptr {
		return ErrNotAStructPtr
	}
	ref := ptrRef.Elem()
	if ref.Kind() != reflect.Struct {
		return ErrNotAStructPtr
	}
	//return doParse(ref, make(map[reflect.Type]ParserFunc, 0))
	return doParse(ref)

}

// ParseWithFuncs is the same as `Parse` except it also allows the user to pass
// in custom parsers.
// func ParseWithFuncs(v interface{}, funcMap CustomParsers) error {
// 	ptrRef := reflect.ValueOf(v)
// 	if ptrRef.Kind() != reflect.Ptr {
// 		return ErrNotAStructPtr
// 	}
// 	ref := ptrRef.Elem()
// 	if ref.Kind() != reflect.Struct {
// 		return ErrNotAStructPtr
// 	}
// 	return doParse(ref, funcMap)
// }

func doParse(ref reflect.Value) error {
	refType := ref.Type()
	var errorList []string

	for i := 0; i < refType.NumField(); i++ {
		if reflect.Ptr == ref.Field(i).Kind() && !ref.Field(i).IsNil() && ref.Field(i).CanSet() {
			err := Parse(ref.Field(i).Interface())
			if nil != err {
				return err
			}
			continue
		}

		// skip unexported fields
		if !ref.Field(i).CanSet() && ref.Field(i).Kind() != reflect.Struct {
			continue
		}

		value, err := get(refType.Field(i))
		// if the field is a struct then doParse(field)
		if ref.Field(i).Kind() == reflect.Struct && value == "" {
			doParse(ref.Field(i))
			continue
		}

		// if the field is a pointer to a struct, then doParse(field.Elem())
		if ref.Field(i).Kind() == reflect.Ptr && ref.Field(i).Elem().Kind() == reflect.Struct && value == "" {
			doParse(ref.Field(i).Elem())
			continue
		}

		if err != nil {
			errorList = append(errorList, err.Error())
			continue
		}
		if value == "" {
			continue
		}
		if err := set(ref.Field(i), refType.Field(i), value); err != nil {
			errorList = append(errorList, err.Error())
			continue
		}
	}
	if len(errorList) == 0 {
		return nil
	}
	return errors.New(strings.Join(errorList, ". "))
}

func get(field reflect.StructField) (string, error) {
	var (
		val string
		err error
	)

	key, opts := parseKeyForOption(field.Tag.Get("env"))

	defaultValue := field.Tag.Get("envDefault")
	val = getOr(key, defaultValue)

	if len(opts) > 0 {
		for _, opt := range opts {
			// The only option supported is "required".
			switch opt {
			case "":
				break
			case "required":
				val, err = getRequired(key)
			default:
				err = errors.New("Env tag option " + opt + " not supported.")
			}
		}
	}

	return val, err
}

// split the env tag's key into the expected key and desired option, if any.
func parseKeyForOption(key string) (string, []string) {
	opts := strings.Split(key, ",")
	return opts[0], opts[1:]
}

func getRequired(key string) (string, error) {
	if value, ok := os.LookupEnv(key); ok {
		return value, nil
	}
	// We do not use fmt.Errorf to avoid another import.
	return "", errors.New("Required environment variable " + key + " is not set")
}

func getOr(key, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if ok {
		return value
	}
	return defaultValue
}

func set(field reflect.Value, refType reflect.StructField, value string) error {
	switch field.Kind() {
	case reflect.Slice:
		separator := refType.Tag.Get("envSeparator")
		return handleSlice(field, value, separator)
	case reflect.String:
		field.SetString(value)
	case reflect.Bool:
		bvalue, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(bvalue)
	case reflect.Int:
		intValue, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return err
		}
		field.SetInt(intValue)
	case reflect.Uint:
		uintValue, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return err
		}
		field.SetUint(uintValue)
	case reflect.Float32:
		v, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return err
		}
		field.SetFloat(v)
	case reflect.Float64:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(v))
	case reflect.Int64:
		if refType.Type.String() == "time.Duration" {
			dValue, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(dValue))
		} else {
			intValue, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return err
			}
			field.SetInt(intValue)
		}
	case reflect.Struct:
		return ErrMismatchType
	default:
		return ErrUnsupportedType
	}
	return nil
}

func handleStruct(field reflect.Value, refType reflect.StructField, value string, funcMap CustomParsers) error {
	// Does the custom parser func map contain this type?
	parserFunc, ok := funcMap[field.Type()]
	if !ok {
		// Map does not contain a custom parser for this type
		return ErrUnsupportedType
	}

	// Call on the custom parser func
	data, err := parserFunc(value)
	if err != nil {
		return fmt.Errorf("Custom parser error: %v", err)
	}

	// Set the field to the data returned by the customer parser func
	rv := reflect.ValueOf(data)
	field.Set(rv)

	return nil
}

func handleSlice(field reflect.Value, value, separator string) error {
	if separator == "" {
		separator = ","
	}

	splitData := strings.Split(value, separator)

	switch field.Type() {
	case sliceOfStrings:
		field.Set(reflect.ValueOf(splitData))
	case sliceOfInts:
		intData, err := parseInts(splitData)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(intData))
	case sliceOfInt64s:
		int64Data, err := parseInt64s(splitData)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(int64Data))

	case sliceOfFloat32s:
		data, err := parseFloat32s(splitData)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(data))
	case sliceOfFloat64s:
		data, err := parseFloat64s(splitData)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(data))
	case sliceOfBools:
		boolData, err := parseBools(splitData)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(boolData))
	default:
		return ErrUnsupportedSliceType
	}
	return nil
}

func parseInts(data []string) ([]int, error) {
	var intSlice []int

	for _, v := range data {
		intValue, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return nil, err
		}
		intSlice = append(intSlice, int(intValue))
	}
	return intSlice, nil
}

func parseInt64s(data []string) ([]int64, error) {
	var intSlice []int64

	for _, v := range data {
		intValue, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}
		intSlice = append(intSlice, int64(intValue))
	}
	return intSlice, nil
}

func parseFloat32s(data []string) ([]float32, error) {
	var float32Slice []float32

	for _, v := range data {
		data, err := strconv.ParseFloat(v, 32)
		if err != nil {
			return nil, err
		}
		float32Slice = append(float32Slice, float32(data))
	}
	return float32Slice, nil
}

func parseFloat64s(data []string) ([]float64, error) {
	var float64Slice []float64

	for _, v := range data {
		data, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, err
		}
		float64Slice = append(float64Slice, float64(data))
	}
	return float64Slice, nil
}

func parseBools(data []string) ([]bool, error) {
	var boolSlice []bool

	for _, v := range data {
		bvalue, err := strconv.ParseBool(v)
		if err != nil {
			return nil, err
		}

		boolSlice = append(boolSlice, bvalue)
	}
	return boolSlice, nil
}
