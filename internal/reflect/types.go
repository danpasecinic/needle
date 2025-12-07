package reflect

import (
	"reflect"
	"sync"
)

var typeKeyCache sync.Map
var namedKeyCache sync.Map

func TypeKey[T any]() string {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		t = reflect.TypeOf((*T)(nil)).Elem()
	}
	return typeKeyFromReflect(t)
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
	case reflect.Ptr:
		return "*" + buildTypeKey(t.Elem())
	case reflect.Slice:
		return "[]" + buildTypeKey(t.Elem())
	case reflect.Array:
		return "[" + string(rune(t.Len())) + "]" + buildTypeKey(t.Elem())
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
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		t = reflect.TypeOf((*T)(nil)).Elem()
	}

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
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return rv.IsNil()
	default:
		return false
	}
}

func TypeName[T any]() string {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		t = reflect.TypeOf((*T)(nil)).Elem()
	}
	return t.String()
}

func IsInterface[T any]() bool {
	t := reflect.TypeOf((*T)(nil)).Elem()
	return t.Kind() == reflect.Interface
}

func Implements[T any](v any) bool {
	if v == nil {
		return false
	}
	t := reflect.TypeOf((*T)(nil)).Elem()
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
	t := reflect.TypeOf((*T)(nil)).Elem()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, nil
	}

	var fields []FieldInfo
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag, ok := field.Tag.Lookup(tagKey)
		if !ok {
			continue
		}

		info := FieldInfo{
			Name:    field.Name,
			TypeKey: typeKeyFromReflect(field.Type),
			Index:   i,
		}

		if tag != "" {
			parts := splitTag(tag)
			for _, part := range parts {
				if part == "optional" {
					info.Optional = true
				} else if part != "" {
					info.Named = part
				}
			}
		}

		fields = append(fields, info)
	}

	return fields, nil
}

func splitTag(tag string) []string {
	var parts []string
	current := ""
	for _, c := range tag {
		if c == ',' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
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

	var params []FuncParamInfo
	for i := 0; i < t.NumIn(); i++ {
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
