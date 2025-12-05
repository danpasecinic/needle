package needle_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/danpasecinic/needle"
)

func TestPrintGraphEmpty(t *testing.T) {
	t.Parallel()

	c := needle.New()

	var buf bytes.Buffer
	c.FprintGraph(&buf)

	if !strings.Contains(buf.String(), "empty container") {
		t.Errorf("expected empty container message, got: %s", buf.String())
	}
}

func TestPrintGraph(t *testing.T) {
	t.Parallel()

	c := needle.New()

	_ = needle.ProvideValue(c, &Config{Port: 8080})
	_ = needle.Provide(
		c, func(ctx context.Context, r needle.Resolver) (*Database, error) {
			_ = needle.MustInvoke[*Config](c)
			return &Database{}, nil
		},
	)

	var buf bytes.Buffer
	c.FprintGraph(&buf)

	output := buf.String()
	if !strings.Contains(output, "Config") {
		t.Errorf("expected Config in output, got: %s", output)
	}
	if !strings.Contains(output, "Database") {
		t.Errorf("expected Database in output, got: %s", output)
	}
}

func TestPrintGraphWithInstantiated(t *testing.T) {
	t.Parallel()

	c := needle.New()

	_ = needle.ProvideValue(c, &Config{Port: 8080})
	_ = needle.MustInvoke[*Config](c)

	var buf bytes.Buffer
	c.FprintGraph(&buf)

	output := buf.String()
	if !strings.Contains(output, "●") {
		t.Errorf("expected instantiated marker (●), got: %s", output)
	}
}

func TestPrintGraphNotInstantiated(t *testing.T) {
	t.Parallel()

	c := needle.New()

	_ = needle.Provide(
		c, func(ctx context.Context, r needle.Resolver) (*Config, error) {
			return &Config{Port: 8080}, nil
		},
	)

	var buf bytes.Buffer
	c.FprintGraph(&buf)

	output := buf.String()
	if !strings.Contains(output, "○") {
		t.Errorf("expected not-instantiated marker (○), got: %s", output)
	}
}

func TestSprintGraph(t *testing.T) {
	t.Parallel()

	c := needle.New()
	_ = needle.ProvideValue(c, &Config{Port: 8080})

	output := c.SprintGraph()
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestPrintGraphDOT(t *testing.T) {
	t.Parallel()

	c := needle.New()

	_ = needle.ProvideValue(c, &Config{Port: 8080})
	_ = needle.Provide(
		c, func(ctx context.Context, r needle.Resolver) (*Database, error) {
			return &Database{}, nil
		}, needle.WithDependencies("*needle_test.Config"),
	)

	var buf bytes.Buffer
	c.FprintGraphDOT(&buf)

	output := buf.String()
	if !strings.Contains(output, "digraph dependencies") {
		t.Errorf("expected digraph header, got: %s", output)
	}
	if !strings.Contains(output, "rankdir=LR") {
		t.Errorf("expected rankdir, got: %s", output)
	}
	if !strings.Contains(output, "->") {
		t.Errorf("expected edge, got: %s", output)
	}
}

func TestSprintGraphDOT(t *testing.T) {
	t.Parallel()

	c := needle.New()
	_ = needle.ProvideValue(c, &Config{Port: 8080})

	output := c.SprintGraphDOT()
	if !strings.Contains(output, "digraph") {
		t.Error("expected digraph in output")
	}
}

func TestGraphInfo(t *testing.T) {
	t.Parallel()

	c := needle.New()

	_ = needle.ProvideValue(c, &Config{Port: 8080})
	_ = needle.Provide(
		c, func(ctx context.Context, r needle.Resolver) (*Database, error) {
			return &Database{}, nil
		}, needle.WithDependencies("*needle_test.Config"),
	)

	info := c.Graph()

	if len(info.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(info.Services))
	}
}
