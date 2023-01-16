// Copyright 2019 Gin Core Team. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package binding

import (
	"reflect"
	"testing"

	"gotest.tools/v3/assert"
)

func TestMappingBaseTypes(t *testing.T) {
	for _, tt := range []struct {
		name   string
		value  any
		source string
		expect any
	}{
		{"base type", struct{ F int }{}, "9", int(9)},
		{"base type", struct{ F int8 }{}, "9", int8(9)},
		{"base type", struct{ F int16 }{}, "9", int16(9)},
		{"base type", struct{ F int32 }{}, "9", int32(9)},
		{"base type", struct{ F int64 }{}, "9", int64(9)},
		{"base type", struct{ F uint }{}, "9", uint(9)},
		{"base type", struct{ F uint8 }{}, "9", uint8(9)},
		{"base type", struct{ F uint16 }{}, "9", uint16(9)},
		{"base type", struct{ F uint32 }{}, "9", uint32(9)},
		{"base type", struct{ F uint64 }{}, "9", uint64(9)},
		{"base type", struct{ F bool }{}, "True", true},
		{"base type", struct{ F float32 }{}, "9.1", float32(9.1)},
		{"base type", struct{ F float64 }{}, "9.1", float64(9.1)},
		{"base type", struct{ F string }{}, "test", "test"},
		{"base type", struct{ F *int }{}, "9", ptr(9)},
		{"base type", struct{ F *string }{}, "9", ptr("9")},

		// zero values
		{"zero value", struct{ F int }{}, "", 0},
		{"zero value", struct{ F uint }{}, "", uint(0)},
		{"zero value", struct{ F bool }{}, "", false},
		{"zero value", struct{ F float32 }{}, "", float32(0)},
	} {
		tp := reflect.TypeOf(tt.value)
		testName := tt.name + ":" + tp.Field(0).Type.String()
		t.Run(testName, func(t *testing.T) {
			val := reflect.New(reflect.TypeOf(tt.value))
			val.Elem().Set(reflect.ValueOf(tt.value))

			_, err := mapping(val, emptyField, formSource{"F": {tt.source}}, "form")
			assert.NilError(t, err, testName)

			actual := val.Elem().Field(0).Interface()
			assert.DeepEqual(t, tt.expect, actual)
		})
	}
}

func ptr[T any](i T) *T {
	return &i
}

func TestMappingSkipField(t *testing.T) {
	var s struct {
		A int
	}
	err := decode(&s, formSource{}, "form")
	assert.NilError(t, err)

	assert.Equal(t, 0, s.A)
}

func TestMappingIgnoreField(t *testing.T) {
	var s struct {
		A int `form:"A"`
		B int `form:"-"`
	}
	err := decode(&s, formSource{"A": {"9"}, "B": {"9"}}, "form")
	assert.NilError(t, err)

	assert.Equal(t, 9, s.A)
	assert.Equal(t, 0, s.B)
}

func TestMappingUnexportedField(t *testing.T) {
	var s struct {
		A int `form:"a"`
		b int `form:"b"`
	}
	err := decode(&s, formSource{"a": {"9"}, "b": {"9"}}, "form")
	assert.NilError(t, err)

	assert.Equal(t, 9, s.A)
	assert.Equal(t, 0, s.b)
}

func TestMappingPrivateField(t *testing.T) {
	var s struct {
		f int `form:"field"`
	}
	err := decode(&s, formSource{"field": {"6"}}, "form")
	assert.NilError(t, err)
	assert.Equal(t, 0, s.f)
}

func TestMappingUnknownFieldType(t *testing.T) {
	var s struct {
		U uintptr
	}

	err := decode(&s, formSource{"U": {"unknown"}}, "form")
	assert.ErrorIs(t, err, errUnknownType)
}

func TestBindURI(t *testing.T) {
	var s struct {
		F int `uri:"field"`
	}
	err := BindURI(map[string][]string{"field": {"6"}}, &s)
	assert.NilError(t, err)
	assert.Equal(t, 6, s.F)
}

func TestMappingForm(t *testing.T) {
	var s struct {
		F int `form:"field"`
	}
	err := decode(&s, map[string][]string{"field": {"6"}}, "form")
	assert.NilError(t, err)
	assert.Equal(t, 6, s.F)
}

func TestMappingSlice(t *testing.T) {
	var s struct {
		Slice []int `form:"slice"`
	}

	// default value
	err := decode(&s, formSource{"slice": []string{"9"}}, "form")
	assert.NilError(t, err)
	assert.DeepEqual(t, []int{9}, s.Slice)

	// ok
	err = decode(&s, formSource{"slice": {"3", "4"}}, "form")
	assert.NilError(t, err)
	assert.DeepEqual(t, []int{3, 4}, s.Slice)

	// error
	err = decode(&s, formSource{"slice": {"wrong"}}, "form")
	assert.ErrorContains(t, err, "invalid syntax")
}

func TestMappingArray(t *testing.T) {
	var s struct {
		Array [2]int `form:"array"`
	}

	// wrong default
	err := decode(&s, formSource{"array": []string{"9"}}, "form")
	assert.ErrorContains(t, err, "is not valid value for [2]int")

	// ok
	err = decode(&s, formSource{"array": {"3", "4"}}, "form")
	assert.NilError(t, err, "what")
	assert.Equal(t, [2]int{3, 4}, s.Array)

	// error - not enough vals
	err = decode(&s, formSource{"array": {"3"}}, "form")
	assert.ErrorContains(t, err, "is not valid value for [2]int")

	// error - wrong value
	err = decode(&s, formSource{"array": {"wrong"}}, "form")
	assert.ErrorContains(t, err, "is not valid value for [2]int")
}

func TestMappingIgnoredCircularRef(t *testing.T) {
	type S struct {
		S *S `form:"-"`
	}
	var s S

	err := decode(&s, formSource{}, "form")
	assert.NilError(t, err)
}
