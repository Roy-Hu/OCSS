package system

import (
	"github.com/comp590/ocss/internal/context"
)

type System struct {
	Hardware context.Hardware
	Traffic  context.Traffic
	States   context.StatesDiagram
}

func (s System) RunState(stateName string, state context.State) string {
	nextState := ""

	for {
		for _, trigger := range state.Triggers {
			if trigger() {
				nxtStateName := state.Actions[0]()

				if nxtStateName != stateName {
					return nextState
				}
			}
		}
	}
}

func (s System) Run() {
	for stateName, state := range s.States.States {
		if state.InitState {
			go func() {
				for {
					nextState := s.RunState(stateName, state)
					if nextState != "" {
						s.RunState(nextState, s.States.States[nextState])
					} else {
						break
					}
				}
			}()
		}

	}
}
