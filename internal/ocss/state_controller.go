package ocss

import (
	"context"
	"sync"

	ocss_context "github.com/comp590/ocss/internal/context"
	"github.com/comp590/ocss/internal/logger"
	"github.com/comp590/ocss/pkg/app"
)

type StateControllerOCSS interface {
	app.App

	Processor() *Processor
}

type StateController struct {
	StateControllerOCSS
}

func NewStateController(ocss StateControllerOCSS) (*StateController, error) {
	s := &StateController{
		StateControllerOCSS: ocss,
	}

	return s, nil
}

func (s *StateController) MonitorApp(appId string, iter int) func(ctx context.Context) bool {
	user := ocss_context.GetSelf().UserView
	if _, ok := user.AppServer.Apps[appId]; !ok {
		user.AppServer.Apps[appId] = &ocss_context.App{
			AppId:             appId,
			FinishedIter:      make(chan int),
			AppChan:           make(chan bool),
			IterTrafficMatrix: make(map[int]ocss_context.TrafficMatrix),
			MonitoredApp:      true,
		}
	}

	return func(ctx context.Context) bool {
		for {
			select {
			case <-ctx.Done():
				return false
			case <-user.AppServer.Apps[appId].AppChan:
				if iter != 0 {
					logger.StateLog.Debugf("State1: App %s started", appId)
					user.AppServer.Apps[appId].MonitoredIter = true
					for {
						select {
						case <-ctx.Done():
							return false
						case iter := <-user.AppServer.Apps[appId].FinishedIter:
							if iter == 1 {
								logger.StateLog.Debugf("State1: AllReduce finished")
								return true
							}
						}
					}
				} else {
					return true
				}
			}
		}
	}
}

func (s *StateController) runState(parentCtx context.Context, stateName string, state *ocss_context.State) string {
	// Create a cancelable context derived from the parent context
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	fanIn := make(chan int)
	var once sync.Once
	var wg sync.WaitGroup

	// Start a goroutine for each trigger
	for i, triggerFunc := range state.Triggers {
		wg.Add(1)
		go func(index int, tf func(ctx context.Context) bool) {
			defer wg.Done()
			activated := tf(ctx) // Pass the cancelable context
			if activated {
				once.Do(func() {
					fanIn <- index
					cancel() // Cancel other triggers
				})
			} else {
				return
			}
		}(i, triggerFunc)
	}

	var resultState string

	// Wait for the first trigger to activate or context cancellation
	select {
	case triggerIndex := <-fanIn:
		logger.StateLog.Infof("State [%s] trigger [%d] activated", stateName, triggerIndex)
		nxtStateName := state.Actions[triggerIndex]()

		if nxtStateName != stateName {
			resultState = nxtStateName
		} else {
			resultState = stateName
		}

	case <-ctx.Done():
		// Context was canceled externally or by a trigger
		logger.StateLog.Infof("State [%s] shutting down due to context cancellation", stateName)
		resultState = ""
	}

	// Wait for all goroutines to finish
	wg.Wait()
	return resultState
}

func (s *StateController) Start(ctx context.Context, wg *sync.WaitGroup) {
	logger.StateLog.Info("State Controller is running")

	wg.Add(1)
	go func(ctx context.Context, wg *sync.WaitGroup) {
		defer wg.Done()

		s.Processor().MonitorTraffic(ctx)
	}(ctx, wg)

	self := ocss_context.GetSelf()

	self.States = s.setupStates(self.UserView)
	for stateName, state := range self.States {
		if state.InitState {
			wg.Add(1)
			go func(initialStateName string, states map[string]*ocss_context.State) {
				defer wg.Done()
				currentStateName := initialStateName
				for {
					logger.StateLog.Infof("State [%s] is running", currentStateName)
					currentState, exists := states[currentStateName]
					if !exists {
						logger.StateLog.Errorf("State [%s] does not exist", currentStateName)
						break
					}

					nextState := s.runState(ctx, currentStateName, currentState)
					logger.StateLog.Errorf("State [%s] transitioning to [%s]", currentStateName, nextState)

					if nextState == "" {
						break
					}
					// ocss_context.PrintFowardingRule()

					s.Processor().UpdateForwardingTables()

					for _, server := range self.UserView.Servers {
						logger.StateLog.Warnf("Server %v connected to %v", server.Name, server.ConnToR)
					}

					currentStateName = nextState
				}
			}(stateName, self.States)
		}
	}
}
