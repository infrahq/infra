package cliopts

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/mitchellh/mapstructure"
	"github.com/mitchellh/reflectwalk"
	"github.com/spf13/pflag"
)

func loadFromEnv(target interface{}, opts Options) error {
	opts.EnvPrefix = strings.ToUpper(opts.EnvPrefix)
	walker := &flatSourceWalker{
		source:   toMap(opts.EnvPrefix, os.Environ()),
		location: []string{opts.EnvPrefix},
		fieldNameFormat: func(name string) string {
			return strings.ToUpper(strcase.ToSnake(name))
		},
		fieldSeparator: "_",
	}

	if err := reflectwalk.Walk(target, walker); err != nil {
		return fmt.Errorf("failed to load from environment variables: %w", err)
	}
	return nil
}

func loadFromFlags(target interface{}, opts Options) error {
	source := map[string]interface{}{}
	opts.Flags.VisitAll(func(flag *pflag.Flag) {
		source[flag.Name] = flag.Value
	})

	walker := &flatSourceWalker{
		source:          source,
		fieldNameFormat: strcase.ToKebab,
		fieldSeparator:  "-",
	}

	if err := reflectwalk.Walk(target, walker); err != nil {
		return fmt.Errorf("failed to load from command line flag: %w", err)
	}
	return nil
}

type flatSourceWalker struct {
	opts            Options
	location        []string
	source          map[string]interface{}
	fieldNameFormat func(string) string
	fieldSeparator  string
}

func (w *flatSourceWalker) Enter(reflectwalk.Location) error {
	return nil
}

func (w *flatSourceWalker) Exit(loc reflectwalk.Location) error {
	if loc == reflectwalk.Struct && len(w.location) > 0 {
		w.location = w.location[:len(w.location)-1]
	}
	return nil
}

func (w *flatSourceWalker) Struct(value reflect.Value) error {
	if !value.CanAddr() {
		// TODO: what is this case?
		return nil
	}
	cfg := DecodeConfig(value.Addr().Interface())
	cfg.WeaklyTypedInput = true
	cfg.MatchName = w.matchName

	decoder, err := mapstructure.NewDecoder(&cfg)
	if err != nil {
		return fmt.Errorf("failed to create decoder for struct: %w", err)
	}
	if err := decoder.Decode(w.source); err != nil {
		return fmt.Errorf("failed to decode into struct: %w", err)
	}
	return nil
}

func (w *flatSourceWalker) StructField(field reflect.StructField, value reflect.Value) error {
	if value.Kind() == reflect.Struct || isPtrToStruct(value) {
		if field.Anonymous { // embedded struct
			w.location = append(w.location, "")
			return nil
		}
		w.location = append(w.location, w.fieldNameFormat(field.Name))
	}
	return nil
}

func (w *flatSourceWalker) matchName(key string, fieldName string) bool {
	var sb strings.Builder
	for _, part := range w.location {
		if part == "" { // skip empty part for embedded struct
			continue
		}
		sb.WriteString(part)
		sb.WriteString(w.fieldSeparator)
	}
	sb.WriteString(w.fieldNameFormat(fieldName))
	return key == sb.String()
}

func isPtrToStruct(value reflect.Value) bool {
	return value.Kind() == reflect.Ptr && value.Elem().Kind() == reflect.Struct
}

// toMap converts an environment variable slice to a map of key/value pairs.
// The environment slice is filtered to only include keys that match the prefix
// because mapstructure iterates over this map, and any keys that do not match
// the prefix will never be used.
func toMap(prefix string, env []string) map[string]interface{} {
	result := map[string]interface{}{}
	for _, raw := range env {
		key, value := getParts(raw)
		if strings.HasPrefix(key, prefix) {
			result[key] = value
		}
	}
	return result
}

func getParts(raw string) (string, string) {
	if raw == "" {
		return "", ""
	}
	// Environment variables on windows can begin with =
	// http://blogs.msdn.com/b/oldnewthing/archive/2010/05/06/10008132.aspx
	parts := strings.SplitN(raw[1:], "=", 2)
	key := raw[:1] + parts[0]
	if len(parts) == 1 {
		return key, ""
	}
	return key, parts[1]
}
