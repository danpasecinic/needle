package needle_test

import (
	"context"
	"testing"

	"github.com/danpasecinic/needle"
)

type ReplaceConfig struct {
	Value string
}

type ReplaceService struct {
	Config *ReplaceConfig
}

func TestReplace(t *testing.T) {
	t.Run(
		"replaces existing provider", func(t *testing.T) {
			c := needle.New()

			_ = needle.ProvideValue(c, &ReplaceConfig{Value: "original"})

			cfg, err := needle.Invoke[*ReplaceConfig](c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Value != "original" {
				t.Errorf("expected 'original', got '%s'", cfg.Value)
			}

			_ = needle.ReplaceValue(c, &ReplaceConfig{Value: "replaced"})

			cfg, err = needle.Invoke[*ReplaceConfig](c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Value != "replaced" {
				t.Errorf("expected 'replaced', got '%s'", cfg.Value)
			}
		},
	)

	t.Run(
		"replaces provider with dependencies", func(t *testing.T) {
			c := needle.New()

			_ = needle.ProvideValue(c, &ReplaceConfig{Value: "v1"})
			_ = needle.Provide(
				c, func(ctx context.Context, r needle.Resolver) (*ReplaceService, error) {
					cfg := needle.MustInvoke[*ReplaceConfig](c)
					return &ReplaceService{Config: cfg}, nil
				},
			)

			svc := needle.MustInvoke[*ReplaceService](c)
			if svc.Config.Value != "v1" {
				t.Errorf("expected 'v1', got '%s'", svc.Config.Value)
			}

			_ = needle.ReplaceValue(c, &ReplaceConfig{Value: "v2"})

			_ = needle.Replace(
				c, func(ctx context.Context, r needle.Resolver) (*ReplaceService, error) {
					cfg := needle.MustInvoke[*ReplaceConfig](c)
					return &ReplaceService{Config: cfg}, nil
				},
			)

			svc = needle.MustInvoke[*ReplaceService](c)
			if svc.Config.Value != "v2" {
				t.Errorf("expected 'v2', got '%s'", svc.Config.Value)
			}
		},
	)

	t.Run(
		"replace non-existent service creates it", func(t *testing.T) {
			c := needle.New()

			_ = needle.ReplaceValue(c, &ReplaceConfig{Value: "new"})

			cfg, err := needle.Invoke[*ReplaceConfig](c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Value != "new" {
				t.Errorf("expected 'new', got '%s'", cfg.Value)
			}
		},
	)
}

func TestReplaceNamed(t *testing.T) {
	t.Run(
		"replaces named provider", func(t *testing.T) {
			c := needle.New()

			_ = needle.ProvideNamedValue(c, "primary", &ReplaceConfig{Value: "orig"})

			cfg, err := needle.InvokeNamed[*ReplaceConfig](c, "primary")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Value != "orig" {
				t.Errorf("expected 'orig', got '%s'", cfg.Value)
			}

			_ = needle.ReplaceNamedValue(c, "primary", &ReplaceConfig{Value: "new"})

			cfg, err = needle.InvokeNamed[*ReplaceConfig](c, "primary")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Value != "new" {
				t.Errorf("expected 'new', got '%s'", cfg.Value)
			}
		},
	)
}

func TestMustReplace(t *testing.T) {
	t.Run(
		"does not panic on valid replace", func(t *testing.T) {
			c := needle.New()

			_ = needle.ProvideValue(c, &ReplaceConfig{Value: "original"})

			needle.MustReplaceValue(c, &ReplaceConfig{Value: "replaced"})

			cfg := needle.MustInvoke[*ReplaceConfig](c)
			if cfg.Value != "replaced" {
				t.Errorf("expected 'replaced', got '%s'", cfg.Value)
			}
		},
	)
}

func TestReplaceWithOptions(t *testing.T) {
	t.Run(
		"replaces with scope option", func(t *testing.T) {
			c := needle.New()

			_ = needle.ProvideValue(c, &ReplaceConfig{Value: "singleton"})

			_ = needle.Replace(
				c, func(ctx context.Context, r needle.Resolver) (*ReplaceConfig, error) {
					return &ReplaceConfig{Value: "transient"}, nil
				},
				needle.WithScope(needle.Transient),
			)

			cfg1 := needle.MustInvoke[*ReplaceConfig](c)
			cfg2 := needle.MustInvoke[*ReplaceConfig](c)

			if cfg1 == cfg2 {
				t.Error("expected different instances for transient scope")
			}
		},
	)
}

func NewReplaceService(cfg *ReplaceConfig) *ReplaceService {
	return &ReplaceService{Config: cfg}
}

func TestReplaceFunc(t *testing.T) {
	t.Run(
		"replaces with auto-wired constructor", func(t *testing.T) {
			c := needle.New()

			_ = needle.ProvideValue(c, &ReplaceConfig{Value: "v1"})
			_ = needle.ProvideFunc[*ReplaceService](c, NewReplaceService)

			svc := needle.MustInvoke[*ReplaceService](c)
			if svc.Config.Value != "v1" {
				t.Errorf("expected 'v1', got '%s'", svc.Config.Value)
			}

			_ = needle.ReplaceValue(c, &ReplaceConfig{Value: "v2"})
			_ = needle.ReplaceFunc[*ReplaceService](c, NewReplaceService)

			svc = needle.MustInvoke[*ReplaceService](c)
			if svc.Config.Value != "v2" {
				t.Errorf("expected 'v2', got '%s'", svc.Config.Value)
			}
		},
	)
}

type ReplaceStructService struct {
	Config *ReplaceConfig `needle:""`
}

func TestReplaceStruct(t *testing.T) {
	t.Run(
		"replaces with struct injection", func(t *testing.T) {
			c := needle.New()

			_ = needle.ProvideValue(c, &ReplaceConfig{Value: "original"})
			_ = needle.ProvideStruct[*ReplaceStructService](c)

			svc := needle.MustInvoke[*ReplaceStructService](c)
			if svc.Config.Value != "original" {
				t.Errorf("expected 'original', got '%s'", svc.Config.Value)
			}

			_ = needle.ReplaceValue(c, &ReplaceConfig{Value: "replaced"})
			_ = needle.ReplaceStruct[*ReplaceStructService](c)

			svc = needle.MustInvoke[*ReplaceStructService](c)
			if svc.Config.Value != "replaced" {
				t.Errorf("expected 'replaced', got '%s'", svc.Config.Value)
			}
		},
	)
}
