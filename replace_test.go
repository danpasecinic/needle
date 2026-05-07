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

			_ = needle.Register(c, needle.SpecValue(&ReplaceConfig{Value: "original"}))

			cfg, err := needle.Invoke[*ReplaceConfig](c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Value != "original" {
				t.Errorf("expected 'original', got '%s'", cfg.Value)
			}

			_ = needle.Replace(c, needle.SpecValue(&ReplaceConfig{Value: "replaced"}))

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

			_ = needle.Register(c, needle.SpecValue(&ReplaceConfig{Value: "v1"}))
			_ = needle.Register(c, needle.Spec[*ReplaceService]{
				Provider: func(ctx context.Context, r needle.Resolver) (*ReplaceService, error) {
					cfg := needle.MustInvoke[*ReplaceConfig](c)
					return &ReplaceService{Config: cfg}, nil
				},
			})

			svc := needle.MustInvoke[*ReplaceService](c)
			if svc.Config.Value != "v1" {
				t.Errorf("expected 'v1', got '%s'", svc.Config.Value)
			}

			_ = needle.Replace(c, needle.SpecValue(&ReplaceConfig{Value: "v2"}))

			_ = needle.Replace(c, needle.Spec[*ReplaceService]{
				Provider: func(ctx context.Context, r needle.Resolver) (*ReplaceService, error) {
					cfg := needle.MustInvoke[*ReplaceConfig](c)
					return &ReplaceService{Config: cfg}, nil
				},
			})

			svc = needle.MustInvoke[*ReplaceService](c)
			if svc.Config.Value != "v2" {
				t.Errorf("expected 'v2', got '%s'", svc.Config.Value)
			}
		},
	)

	t.Run(
		"replace non-existent service creates it", func(t *testing.T) {
			c := needle.New()

			_ = needle.Replace(c, needle.SpecValue(&ReplaceConfig{Value: "new"}))

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

			_ = needle.Register(c, needle.SpecValue(&ReplaceConfig{Value: "orig"}).WithName("primary"))

			cfg, err := needle.InvokeNamed[*ReplaceConfig](c, "primary")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Value != "orig" {
				t.Errorf("expected 'orig', got '%s'", cfg.Value)
			}

			_ = needle.Replace(c, needle.SpecValue(&ReplaceConfig{Value: "new"}).WithName("primary"))

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

			_ = needle.Register(c, needle.SpecValue(&ReplaceConfig{Value: "original"}))

			needle.MustReplace(c, needle.SpecValue(&ReplaceConfig{Value: "replaced"}))

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

			_ = needle.Register(c, needle.SpecValue(&ReplaceConfig{Value: "singleton"}))

			_ = needle.Replace(c, needle.Spec[*ReplaceConfig]{
				Provider: func(ctx context.Context, r needle.Resolver) (*ReplaceConfig, error) {
					return &ReplaceConfig{Value: "transient"}, nil
				},
				Scope: needle.Transient,
			})

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

func TestReplaceConstructor(t *testing.T) {
	t.Run(
		"replaces with auto-wired constructor", func(t *testing.T) {
			c := needle.New()

			_ = needle.Register(c, needle.SpecValue(&ReplaceConfig{Value: "v1"}))
			_ = needle.Register(c, needle.SpecFromConstructor[*ReplaceService](NewReplaceService))

			svc := needle.MustInvoke[*ReplaceService](c)
			if svc.Config.Value != "v1" {
				t.Errorf("expected 'v1', got '%s'", svc.Config.Value)
			}

			_ = needle.Replace(c, needle.SpecValue(&ReplaceConfig{Value: "v2"}))
			_ = needle.Replace(c, needle.SpecFromConstructor[*ReplaceService](NewReplaceService))

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

			_ = needle.Register(c, needle.SpecValue(&ReplaceConfig{Value: "original"}))
			_ = needle.Register(c, needle.SpecFromStruct[*ReplaceStructService]())

			svc := needle.MustInvoke[*ReplaceStructService](c)
			if svc.Config.Value != "original" {
				t.Errorf("expected 'original', got '%s'", svc.Config.Value)
			}

			_ = needle.Replace(c, needle.SpecValue(&ReplaceConfig{Value: "replaced"}))
			_ = needle.Replace(c, needle.SpecFromStruct[*ReplaceStructService]())

			svc = needle.MustInvoke[*ReplaceStructService](c)
			if svc.Config.Value != "replaced" {
				t.Errorf("expected 'replaced', got '%s'", svc.Config.Value)
			}
		},
	)
}
