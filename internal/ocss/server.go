package ocss

import (
	"context"
	"sync"

	"github.com/comp590/ocss/internal/logger"
	"github.com/comp590/ocss/pkg/app"
)

type ServerApp interface {
	app.App
	Processor() *Processor
	StateController() *StateController
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

func (s *Server) Run(ctx context.Context, wg *sync.WaitGroup) error {
	logger.ServerLog.Info("OCSS Server is running")

	s.Processor().CreateForwardingTables()

	s.StateController().Start(ctx, wg)

	wg.Wait()

	return nil
}

func (s Server) Stop() {
}
