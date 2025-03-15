package context

import (
	"context"
)

type State struct {
	Triggers []func(ctx context.Context, Servers map[string]bool) bool
	Actions  []func(Servers map[string]bool) string
	Vars     map[string]interface{}

	Ctx    context.Context
	Cancel context.CancelFunc

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

	Ctx    context.Context
	Cancel context.CancelFunc
}
