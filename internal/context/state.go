package context

import (
	"context"
)

type State struct {
	Triggers  []func(ctx context.Context) bool `yaml:"-"`
	Actions   []func() string                  `yaml:"-"`
	Vars      map[string]interface{}           `yaml:"-"`
	InitState bool
}
