package decode

import (
	"reflect"
)

type Decoder func(target interface{}, source interface{}) error

type PrepareForDecoder interface {
	PrepareForDecode(data interface{}) error
}

// HookPrepareForDecode is a mapstructure.DecodeHookFuncValue that enables decoding
// of any type that implements the PrepareForDecoder interface.
//
// Types that implement PrepareForDecoder can use the passed in data to set
// concrete types on any polymorphic fields, which will allow mapstructure.Decode
// to properly decode the config into the expected type.
func HookPrepareForDecode(from reflect.Value, to reflect.Value) (interface{}, error) {
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
