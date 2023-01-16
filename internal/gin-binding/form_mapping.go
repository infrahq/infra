// Copyright 2014 Manu Martinez-Almeida. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package binding

import (
	"encoding"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

func BindURI(m map[string][]string, obj any) error {
	return decode(obj, m, "uri")
}

func BindQuery(req *http.Request, obj any) error {
	values := req.URL.Query()
	return decode(obj, values, "form")
}

func decode(target any, source map[string][]string, tag string) error {
	return decodeStruct(reflect.ValueOf(target), source, tag)
}

type formSource map[string][]string

func decodeStruct(value reflect.Value, source formSource, tag string) error {
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	tValue := value.Type()
	for i := 0; i < value.NumField(); i++ {
		sf := tValue.Field(i)
		if sf.PkgPath != "" && !sf.Anonymous { // unexported
			continue
		}

		if _, ok := value.Field(i).Addr().Interface().(encoding.TextUnmarshaler); ok {
			err := tryToSetValue(value.Field(i), sf, source, tag)
			if err != nil {
				return err
			}
			continue
		}

		var err error
		if sf.Type.Kind() == reflect.Struct {
			err = decodeStruct(value.Field(i), source, tag)
		} else {
			err = tryToSetValue(value.Field(i), sf, source, tag)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func tryToSetValue(value reflect.Value, field reflect.StructField, source formSource, tag string) error {
	tagValue, _, _ := strings.Cut(field.Tag.Get(tag), ",")
	if tagValue == "-" {
		return nil
	}

	// TODO: remove
	if tagValue == "" { // default value is FieldName
		tagValue = field.Name
	}
	if tagValue == "" { // when field is "emptyField" variable
		return nil
	}

	vs, ok := source[tagValue]
	if !ok || len(vs) == 0 {
		return nil
	}
	val := vs[0]

	// TODO: remove?
	if u, ok := value.Addr().Interface().(encoding.TextUnmarshaler); ok {
		return u.UnmarshalText([]byte(val))
	}

	// nolint:exhaustive
	switch value.Kind() {
	case reflect.Slice:
		return setSlice(vs, value)
	case reflect.Pointer:
		// TODO: can we reduce this at all?
		var isNew bool
		vPtr := value
		if value.IsNil() {
			isNew = true
			vPtr = reflect.New(value.Type().Elem())
		}
		err := setValue(val, vPtr.Elem())
		if err != nil {
			return err
		}
		if isNew {
			value.Set(vPtr)
		}
		return nil
	default:
		return setValue(val, value)
	}
}

func setValue(val string, value reflect.Value) error {
	if val == "" {
		return nil
	}

	if u, ok := value.Addr().Interface().(encoding.TextUnmarshaler); ok {
		return u.UnmarshalText([]byte(val))
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
		return fmt.Errorf("type %v is not supported by decode", value.Type())
	}
	return nil
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
