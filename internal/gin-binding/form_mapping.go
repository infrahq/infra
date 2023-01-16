// Copyright 2014 Manu Martinez-Almeida. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package binding

import (
	"encoding"
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"unsafe"
)

func BindURI(m map[string][]string, obj any) error {
	return decode(obj, m, "uri")
}

func BindQuery(req *http.Request, obj any) error {
	values := req.URL.Query()
	return decode(obj, values, "form")
}

var errUnknownType = errors.New("unknown type")

var emptyField = reflect.StructField{}

func decode(target any, source map[string][]string, tag string) error {
	_, err := mapping(reflect.ValueOf(target), emptyField, source, tag)
	return err
}

type formSource map[string][]string

func mapping(value reflect.Value, field reflect.StructField, source formSource, tag string) (bool, error) {
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	vKind := value.Kind()
	if vKind != reflect.Struct || !field.Anonymous {
		ok, err := tryToSetValue(value, field, source, tag)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}

	if vKind == reflect.Struct {
		tValue := value.Type()

		var isSet bool
		for i := 0; i < value.NumField(); i++ {
			sf := tValue.Field(i)
			if sf.PkgPath != "" && !sf.Anonymous { // unexported
				continue
			}

			var err error
			var ok bool
			if sf.Type.Kind() == reflect.Struct {
				ok, err = mapping(value.Field(i), sf, source, tag)
			} else {
				ok, err = tryToSetValue(value.Field(i), sf, source, tag)
			}
			if err != nil {
				return false, err
			}
			isSet = isSet || ok
		}
		return isSet, nil
	}
	return false, nil
}

func tryToSetValue(value reflect.Value, field reflect.StructField, source formSource, tag string) (bool, error) {
	tagValue, _, _ := strings.Cut(field.Tag.Get(tag), ",")
	if tagValue == "-" {
		return false, nil
	}

	// TODO: remove
	if tagValue == "" { // default value is FieldName
		tagValue = field.Name
	}
	if tagValue == "" { // when field is "emptyField" variable
		return false, nil
	}

	vs, ok := source[tagValue]
	if !ok || len(vs) == 0 {
		return false, nil
	}
	val := vs[0]

	if u, ok := value.Addr().Interface().(encoding.TextUnmarshaler); ok {
		return true, u.UnmarshalText(stringToBytes(val))
	}

	// nolint:exhaustive
	switch value.Kind() {
	case reflect.Slice:
		return true, setSlice(vs, value)
	case reflect.Pointer:
		// TODO: can we reduce this at all?
		var isNew bool
		vPtr := value
		if value.IsNil() {
			isNew = true
			vPtr = reflect.New(value.Type().Elem())
		}
		isSet, err := mapping(vPtr.Elem(), field, source, tag)
		if err != nil {
			return false, err
		}
		if isNew && isSet {
			value.Set(vPtr)
		}
		return isSet, nil
	default:
		return true, setValue(val, value)
	}
}

func setValue(val string, value reflect.Value) error {
	if val == "" {
		return nil
	}

	if u, ok := value.Addr().Interface().(encoding.TextUnmarshaler); ok {
		return u.UnmarshalText(stringToBytes(val))
	}

	// nolint:exhaustive
	switch value.Kind() {
	case reflect.Int:
		return setIntField(val, 0, value)
	case reflect.Int8:
		return setIntField(val, 8, value)
	case reflect.Int16:
		return setIntField(val, 16, value)
	case reflect.Int32:
		return setIntField(val, 32, value)
	case reflect.Int64:
		return setIntField(val, 64, value)
	case reflect.Uint:
		return setUintField(val, 0, value)
	case reflect.Uint8:
		return setUintField(val, 8, value)
	case reflect.Uint16:
		return setUintField(val, 16, value)
	case reflect.Uint32:
		return setUintField(val, 32, value)
	case reflect.Uint64:
		return setUintField(val, 64, value)
	case reflect.Bool:
		return setBoolField(val, value)
	case reflect.Float32:
		return setFloatField(val, 32, value)
	case reflect.Float64:
		return setFloatField(val, 64, value)
	case reflect.String:
		value.SetString(val)
	default:
		return errUnknownType
	}
	return nil
}

// TODO: remove this optimization. Maybe change to []byte everywhere?
// stringToBytes converts string to byte slice without a memory allocation.
func stringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}

func setIntField(val string, bitSize int, field reflect.Value) error {
	intVal, err := strconv.ParseInt(val, 10, bitSize)
	if err == nil {
		field.SetInt(intVal)
	}
	return err
}

func setUintField(val string, bitSize int, field reflect.Value) error {
	uintVal, err := strconv.ParseUint(val, 10, bitSize)
	if err == nil {
		field.SetUint(uintVal)
	}
	return err
}

func setBoolField(val string, field reflect.Value) error {
	boolVal, err := strconv.ParseBool(val)
	if err == nil {
		field.SetBool(boolVal)
	}
	return err
}

func setFloatField(val string, bitSize int, field reflect.Value) error {
	floatVal, err := strconv.ParseFloat(val, bitSize)
	if err == nil {
		field.SetFloat(floatVal)
	}
	return err
}

func setSlice(source []string, value reflect.Value) error {
	slice := reflect.MakeSlice(value.Type(), len(source), len(source))
	for i, s := range source {
		if err := setValue(s, slice.Index(i)); err != nil {
			return err
		}
	}
	value.Set(slice)
	return nil
}
