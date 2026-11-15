package reflect

import (
	"reflect"
	"strconv"
	"strings"
	"sync"
)

var typeKeyCache sync.Map
var namedKeyCache sync.Map

func TypeKey[T any]() string {
	return typeKeyFromReflect(reflect.TypeFor[T]())
}

type namedKey struct {
	t    reflect.Type
	name string
}

func typeKeyFromReflect(t reflect.Type) string {
	if cached, ok := typeKeyCache.Load(t); ok {
		return cached.(string)
	}

	key := buildTypeKey(t)
	typeKeyCache.Store(t, key)
	return key
}

func buildTypeKey(t reflect.Type) string {
	if t == nil {
		return "<nil>"
	}

	switch t.Kind() {
	case reflect.Pointer:
		return "*" + buildTypeKey(t.Elem())
	case reflect.Slice:
		return "[]" + buildTypeKey(t.Elem())
	case reflect.Array:
		return "[" + strconv.Itoa(t.Len()) + "]" + buildTypeKey(t.Elem())
	case reflect.Map:
		return "map[" + buildTypeKey(t.Key()) + "]" + buildTypeKey(t.Elem())
	case reflect.Chan:
		switch t.ChanDir() {
		case reflect.RecvDir:
			return "<-chan " + buildTypeKey(t.Elem())
		case reflect.SendDir:
			return "chan<- " + buildTypeKey(t.Elem())
		default:
			return "chan " + buildTypeKey(t.Elem())
		}
	case reflect.Func:
		return t.String()
	default:
		if t.PkgPath() != "" {
			return t.PkgPath() + "." + t.Name()
		}
		return t.Name()
	}
}

func TypeKeyFromValue(v any) string {
	if v == nil {
		return "<nil>"
	}
	return typeKeyFromReflect(reflect.TypeOf(v))
}

func TypeKeyNamed[T any](name string) string {
	t := reflect.TypeFor[T]()

	key := namedKey{t: t, name: name}
	if cached, ok := namedKeyCache.Load(key); ok {
		return cached.(string)
	}

	result := typeKeyFromReflect(t) + "#" + name
	namedKeyCache.Store(key, result)
	return result
}

func TypeKeyNamedFromValue(v any, name string) string {
	return TypeKeyFromValue(v) + "#" + name
}

func IsNil(v any) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return rv.IsNil()
	default:
		return false
	}
}

func TypeName[T any]() string {
	return reflect.TypeFor[T]().String()
}

func IsInterface[T any]() bool {
	t := reflect.TypeFor[T]()
	return t.Kind() == reflect.Interface
}

func Implements[T any](v any) bool {
	if v == nil {
		return false
	}
	t := reflect.TypeFor[T]()
	return reflect.TypeOf(v).Implements(t)
}

type FieldInfo struct {
	Name     string
	TypeKey  string
	Index    int
	Optional bool
	Named    string
}

func StructFields[T any](tagKey string) ([]FieldInfo, error) {
	t := reflect.TypeFor[T]()
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, nil
	}

	var fields []FieldInfo
	for field := range t.Fields() {
		tag, ok := field.Tag.Lookup(tagKey)
		if !ok {
			continue
		}

		info := FieldInfo{
			Name:    field.Name,
			TypeKey: typeKeyFromReflect(field.Type),
			Index:   field.Index[0],
		}

		for part := range strings.SplitSeq(tag, ",") {
			switch part {
			case "":
			case "optional":
				info.Optional = true
			default:
				info.Named = part
			}
		}

		fields = append(fields, info)
	}

	return fields, nil
}

type FuncParamInfo struct {
	Index   int
	TypeKey string
	Type    reflect.Type
}

func FuncParams(fn any) ([]FuncParamInfo, reflect.Type, error) {
	t := reflect.TypeOf(fn)
	if t == nil || t.Kind() != reflect.Func {
		return nil, nil, nil
	}

	params := make([]FuncParamInfo, 0, t.NumIn())
	for i := range t.NumIn() {
		paramType := t.In(i)
		params = append(
			params, FuncParamInfo{
				Index:   i,
				TypeKey: typeKeyFromReflect(paramType),
				Type:    paramType,
			},
		)
	}

	var returnType reflect.Type
	if t.NumOut() > 0 {
		returnType = t.Out(0)
	}

	return params, returnType, nil
}
