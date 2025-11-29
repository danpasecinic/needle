package reflect

import (
	"reflect"
	"sync"
)

var typeKeyCache sync.Map

func TypeKey[T any]() string {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		t = reflect.TypeOf((*T)(nil)).Elem()
	}
	return typeKeyFromReflect(t)
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
	return TypeKey[T]() + "#" + name
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
