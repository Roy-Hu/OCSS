package ocss

import (
	"context"
	"sync"

	ocss_context "github.com/comp590/ocss/internal/context"
	"github.com/comp590/ocss/internal/logger"
	"github.com/comp590/ocss/pkg/app"
)

type ServerApp interface {
	app.App
	Processor() *Processor
}

type Server struct {
	ServerApp
}

func NewServer(ocss ServerApp) (*Server, error) {
	s := &Server{
		ServerApp: ocss,
	}

	return s, nil
}

func (s *Server) runState(stateName string, state ocss_context.State) string {
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

func (s *Server) Run(traceCtx context.Context, wg *sync.WaitGroup) error {
	logger.ServerLog.Info("OCSS Server is running")

	s.Processor().SetupForwardingTables()
	s.Processor().SetupController()
	// for stateName, state := range s.States.States {
	// 	if state.InitState {
	// 		go func() {
	// 			for {
	// 				nextState := s.runState(stateName, state)
	// 				if nextState != "" {
	// 					s.runState(nextState, s.States.States[nextState])
	// 				} else {
	// 					break
	// 				}
	// 			}
	// 		}()
	// 	}
	// }

	return nil
}

func (s Server) Stop() {
}
