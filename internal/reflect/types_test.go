package reflect

import (
	"context"
	"testing"
)

type testInterface interface {
	DoSomething()
}

type testStruct struct {
	Name string
}

func (t *testStruct) DoSomething() {}

func TestTypeKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		typeFunc func() string
		wantNot  string
	}{
		{
			name:     "int",
			typeFunc: TypeKey[int],
			wantNot:  "",
		},
		{
			name:     "string",
			typeFunc: TypeKey[string],
			wantNot:  "",
		},
		{
			name:     "pointer to struct",
			typeFunc: TypeKey[*testStruct],
			wantNot:  "",
		},
		{
			name:     "slice",
			typeFunc: TypeKey[[]string],
			wantNot:  "",
		},
		{
			name:     "map",
			typeFunc: TypeKey[map[string]int],
			wantNot:  "",
		},
		{
			name:     "interface",
			typeFunc: TypeKey[testInterface],
			wantNot:  "",
		},
		{
			name:     "context.Context",
			typeFunc: TypeKey[context.Context],
			wantNot:  "",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()
				got := tt.typeFunc()
				if got == "" {
					t.Error("TypeKey returned empty string")
				}
			},
		)
	}
}

func TestTypeKeyUnique(t *testing.T) {
	t.Parallel()

	keys := map[string]bool{}
	testCases := []func() string{
		TypeKey[int],
		TypeKey[int32],
		TypeKey[int64],
		TypeKey[string],
		TypeKey[*string],
		TypeKey[[]string],
		TypeKey[map[string]int],
		TypeKey[testStruct],
		TypeKey[*testStruct],
	}

	for _, tc := range testCases {
		key := tc()
		if keys[key] {
			t.Errorf("duplicate key: %s", key)
		}
		keys[key] = true
	}
}

func TestTypeKeyNamed(t *testing.T) {
	t.Parallel()

	key1 := TypeKeyNamed[testStruct]("primary")
	key2 := TypeKeyNamed[testStruct]("secondary")
	key3 := TypeKey[testStruct]()

	if key1 == key2 {
		t.Error("named keys should be different")
	}
	if key1 == key3 {
		t.Error("named key should differ from unnamed")
	}
}

func TestTypeKeyFromValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
	}{
		{"nil", nil},
		{"int", 42},
		{"string", "hello"},
		{"struct", testStruct{}},
		{"pointer", &testStruct{}},
		{"slice", []string{"a", "b"}},
		{"map", map[string]int{"a": 1}},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()
				key := TypeKeyFromValue(tt.value)
				if key == "" {
					t.Error("TypeKeyFromValue returned empty string")
				}
			},
		)
	}
}

func TestIsNil(t *testing.T) {
	t.Parallel()

	var nilPtr *testStruct
	var nilSlice []string
	var nilMap map[string]int
	var nilInterface testInterface

	tests := []struct {
		name string
		v    any
		want bool
	}{
		{"nil", nil, true},
		{"nil pointer", nilPtr, true},
		{"nil slice", nilSlice, true},
		{"nil map", nilMap, true},
		{"nil interface", nilInterface, true},
		{"non-nil int", 42, false},
		{"non-nil string", "hello", false},
		{"non-nil struct", testStruct{}, false},
		{"non-nil pointer", &testStruct{}, false},
		{"non-nil slice", []string{"a"}, false},
		{"non-nil map", map[string]int{"a": 1}, false},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()
				if got := IsNil(tt.v); got != tt.want {
					t.Errorf("IsNil() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func TestIsInterface(t *testing.T) {
	t.Parallel()

	if !IsInterface[testInterface]() {
		t.Error("testInterface should be detected as interface")
	}
	if IsInterface[testStruct]() {
		t.Error("testStruct should not be detected as interface")
	}
	if IsInterface[*testStruct]() {
		t.Error("*testStruct should not be detected as interface")
	}
}

func TestImplements(t *testing.T) {
	t.Parallel()

	ts := &testStruct{}

	if !Implements[testInterface](ts) {
		t.Error("*testStruct should implement testInterface")
	}
	if Implements[testInterface](42) {
		t.Error("int should not implement testInterface")
	}
	if Implements[testInterface](nil) {
		t.Error("nil should not implement testInterface")
	}
}

func BenchmarkTypeKey(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = TypeKey[*testStruct]()
	}
}

func BenchmarkTypeKeyNamed(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = TypeKeyNamed[*testStruct]("primary")
	}
}
