package cliopts

import (
	"reflect"
)

type flagValueSlice interface {
	GetSlice() []string
}

// hookFlagValueSlice allows for decoding from pflag.SliceValue types into a
// slice in the target.
func hookFlagValueSlice(from reflect.Value, to reflect.Value) (interface{}, error) {
	source := from.Interface()
	v, ok := source.(flagValueSlice)
	if !ok {
		return source, nil
	}
	return v.GetSlice(), nil
}

type PrepareForDecoder interface {
	PrepareForDecode(data interface{}) error
}

// hookPrepareForDecode is a mapstructure.DecodeHookFuncValue that enables decoding
// of any type that implements the PrepareForDecoder interface.
//
// Types that implement PrepareForDecoder can use the passed in data to set
// concrete types on any polymorphic fields, which will allow mapstructure.Decode
// to properly decode the config into the expected type.
func hookPrepareForDecode(from reflect.Value, to reflect.Value) (interface{}, error) {
	source := from.Interface()
	unmapper, ok := to.Interface().(PrepareForDecoder)
	if !ok {
		if to.CanAddr() {
			unmapper, ok = to.Addr().Interface().(PrepareForDecoder)
		}
		if !ok {
			return source, nil
		}
	}

	err := unmapper.PrepareForDecode(source)
	return source, err
}

type FromString interface {
	Set(string) error
}

// hookSetFromString allows any complex type that implements FromString to
// set its value from a string.
//
// This same interface is accepted by spf13/pflag, which allows us to use the
// same type for command line flags, env vars, and config files.
func hookSetFromString(from reflect.Value, to reflect.Value) (interface{}, error) {
	source := from.Interface()
	v, ok := source.(string)
	if !ok {
		return source, nil
	}

	fromString, ok := to.Interface().(FromString)
	if !ok {
		if to.CanAddr() {
			fromString, ok = to.Addr().Interface().(FromString)
		}
		if !ok {
			return source, nil
		}
	}

	err := fromString.Set(v)
	return to, err
}
