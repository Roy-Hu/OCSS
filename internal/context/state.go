package context

import (
	"context"
)

type State struct {
	Triggers []func(ctx context.Context, Servers map[string]bool) bool `yaml:"-"`
	Actions  []func(Servers map[string]bool) string                    `yaml:"-"`
	Vars     map[string]interface{}                                    `yaml:"-"`

	InitState bool
}

type StateInfo struct {
	Servers []string

	Name string
}

type StateMachine struct {
	// All possible stater
	States map[string]*State

	Servers map[string]bool
}
