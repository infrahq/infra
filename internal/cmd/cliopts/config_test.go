package cliopts

import (
	"testing"

	"github.com/spf13/pflag"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

type Example struct {
	One         string
	StringField string
	BoolField   bool
	Int32Field  int32
	Singleword  int
	HostTHING   string
	More        string

	StringFromEnv string
	BoolFromEnv   bool
	UintFromEnv   uint
	NetIPFromEnv  string
	SkipTLSVerify bool

	Nest    Nested
	NestPtr *Nested

	Nested

	ManyThings  []string
	ManyNumbers []int

	// TODO: and map
	// TODO: type that defines Unmarshal/Decode method
}

type Nested struct {
	Two     string
	Twine   string
	Numb    int
	Flag    bool
	Ratio   float64
	Another string
}

func TestLoad(t *testing.T) {
	content := `
stringField: from-file
boolField: true
int32Field: 2
singleword: 3
hostThing: ok
more: not-this

stringFromEnv: from-file-2
boolFromEnv: false
uintFromEnv: 5

nest:
    numb: -2
    another: not-this

nestPtr:
    two: "the-value"
    ratio: 3.15

two: "from-file-3"
manyThings: [one, two]
manyNumbers: [1,2,3]
`
	f := fs.NewFile(t, t.Name(), fs.WithContent(content))

	t.Setenv("APPNAME_STRING_FROM_ENV", "from-env-1")
	t.Setenv("APPNAME_BOOL_FROM_ENV", "true")
	t.Setenv("APPNAME_UINT_FROM_ENV", "412")
	t.Setenv("APPNAME_NET_IP_FROM_ENV", "0.0.0.0")
	t.Setenv("APPNAME_NEST_TWINE", "from-env-2")
	t.Setenv("APPNAME_NEST_RATIO", "3.14")
	t.Setenv("APPNAME_NEST_PTR_TWINE", "from-env-3")
	t.Setenv("APPNAME_NEST_PTR_FLAG", "true")
	t.Setenv("APPNAME_TWINE", "from-env-4")
	t.Setenv("APPNAME_MORE", "not-this-either")

	target := Example{
		One:         "left-as-default",
		StringField: "default-1",
		Int32Field:  1,
		More:        "default-2",
		Nest: Nested{
			Two: "left-as-default-2",
		},
	}

	opts := Options{
		Filename:  f.Path(),
		EnvPrefix: "APPNAME",
	}
	err := Load(&target, opts)
	assert.NilError(t, err)
	expected := Example{
		One:           "left-as-default",
		StringField:   "from-file",
		BoolField:     true,
		Int32Field:    2,
		Singleword:    3,
		HostTHING:     "ok",
		StringFromEnv: "from-env-1",
		BoolFromEnv:   true,
		UintFromEnv:   412,
		NetIPFromEnv:  "0.0.0.0",
		More:          "not-this-either",
		Nest: Nested{
			Two:     "left-as-default-2",
			Twine:   "from-env-2",
			Numb:    -2,
			Ratio:   3.14,
			Another: "not-this",
		},
		NestPtr: &Nested{
			Two:   "the-value",
			Twine: "from-env-3",
			Flag:  true,
			Ratio: 3.15,
		},
		Nested: Nested{
			Two:   "from-file-3",
			Twine: "from-env-4",
		},
		ManyThings:  []string{"one", "two"},
		ManyNumbers: []int{1, 2, 3},
	}
	assert.DeepEqual(t, target, expected)
}

func TestLoad_WithFlags(t *testing.T) {
	content := `
stringField: from-file
boolField: true
int32Field: 2
singleword: 3
hostThing: ok
more: not-this

stringFromEnv: from-file-2
boolFromEnv: false
uintFromEnv: 5

nest:
    numb: -2
    another: not-this

nestPtr:
    two: "the-value"
    ratio: 3.15

two: "from-file-3"
`
	f := fs.NewFile(t, t.Name(), fs.WithContent(content))

	t.Setenv("APPNAME_STRING_FROM_ENV", "from-env-1")
	t.Setenv("APPNAME_BOOL_FROM_ENV", "true")
	t.Setenv("APPNAME_UINT_FROM_ENV", "412")
	t.Setenv("APPNAME_NET_IP_FROM_ENV", "0.0.0.0")
	t.Setenv("APPNAME_NEST_TWINE", "from-env-2")
	t.Setenv("APPNAME_NEST_RATIO", "3.14")
	t.Setenv("APPNAME_NEST_PTR_TWINE", "from-env-3")
	t.Setenv("APPNAME_TWINE", "from-env-4")
	t.Setenv("APPNAME_MORE", "not-this-either")

	target := Example{
		One:         "left-as-default",
		StringField: "default-1",
		Int32Field:  1,
		More:        "default-2",
		Nest: Nested{
			Two: "left-as-default-2",
		},
	}

	flags := pflag.NewFlagSet("any", pflag.ContinueOnError)
	flags.String("string-field", "", "")
	flags.Int32("int-32-field", 0, "")
	flags.Bool("bool-field", false, "")
	flags.String("nest-twine", "", "")
	flags.String("two", "", "")
	flags.Bool("nest-ptr-flag", false, "")
	flags.StringSlice("many-things", nil, "")
	flags.IntSlice("many-numbers", nil, "")
	flags.Bool("skip-tls-verify", false, "")

	err := flags.Parse([]string{
		"--string-field=from-flag-1",
		"--int-32-field=45",
		"--bool-field=false",
		"--nest-twine=from-flag-2",
		"--two=from-flag-3",
		"--nest-ptr-flag",
		"--many-things=first",
		"--many-things=second",
		"--many-numbers=9",
		"--many-numbers=10",
		"--many-numbers=11",
		"--skip-tls-verify",
	})
	assert.NilError(t, err)

	opts := Options{
		Filename:  f.Path(),
		EnvPrefix: "APPNAME",
		Flags:     flags,
	}
	err = Load(&target, opts)
	assert.NilError(t, err)
	expected := Example{
		One:           "left-as-default",
		StringField:   "from-flag-1",
		BoolField:     false,
		Int32Field:    45,
		Singleword:    3,
		HostTHING:     "ok",
		StringFromEnv: "from-env-1",
		BoolFromEnv:   true,
		UintFromEnv:   412,
		NetIPFromEnv:  "0.0.0.0",
		SkipTLSVerify: true,
		More:          "not-this-either",
		Nest: Nested{
			Two:     "left-as-default-2",
			Twine:   "from-flag-2",
			Numb:    -2,
			Ratio:   3.14,
			Another: "not-this",
		},
		NestPtr: &Nested{
			Two:   "the-value",
			Twine: "from-env-3",
			Flag:  true,
			Ratio: 3.15,
		},
		Nested: Nested{
			Two:   "from-flag-3",
			Twine: "from-env-4",
		},
		ManyThings:  []string{"first", "second"},
		ManyNumbers: []int{9, 10, 11},
	}
	assert.DeepEqual(t, target, expected)
}
