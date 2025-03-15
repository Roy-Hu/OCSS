package ocss

import (
	"context"
	"sync"

	ocss_context "github.com/comp590/ocss/internal/context"
	"github.com/comp590/ocss/internal/logger"
	"github.com/comp590/ocss/pkg/app"
	"github.com/mitchellh/copystructure"
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
			FinishedIter:      []chan int{},
			IterTrafficMatrix: make(map[int]*ocss_context.IterTraffic),
			MonitoredIter:     []bool{},
		}
	}

	j := len(user.AppServer.Apps[appId].FinishedIter)
	user.AppServer.Apps[appId].MonitoredIter = append(user.AppServer.Apps[appId].MonitoredIter, true)
	user.AppServer.Apps[appId].FinishedIter = append(user.AppServer.Apps[appId].FinishedIter, make(chan int))

	return func(ctx context.Context) bool {
		for {
			select {
			case <-ctx.Done():
				return false
			case recvIter := <-user.AppServer.Apps[appId].FinishedIter[j]:
				if iter == recvIter {
					logger.StateLog.Infof("App %s finished iter %d", appId, iter)
					user.AppServer.Apps[appId].MonitoredIter[j] = false
					return true
				}
			}
		}
	}
}

func (s *StateController) runState(parentCtx context.Context, stateName string, stateMachine *ocss_context.StateMachine) string {
	fanIn := make(chan int)
	var once sync.Once
	var wg sync.WaitGroup

	state := stateMachine.States[stateName]
	servers := stateMachine.Servers

	// Create a cancelable context derived from the parent context
	state.Ctx, state.Cancel = context.WithCancel(parentCtx)
	defer state.Cancel()

	// Start a goroutine for each trigger
	for i, triggerFunc := range state.Triggers {
		wg.Add(1)
		go func(index int, tf func(ctx context.Context, servers map[string]bool) bool, servers map[string]bool) {
			defer wg.Done()
			activated := tf(state.Ctx, servers) // Pass the cancelable context
			if activated {
				once.Do(func() {
					fanIn <- index
					state.Cancel() // Cancel other triggers
				})
			} else {
				return
			}
		}(i, triggerFunc, servers)
	}

	var resultState string

	// Wait for the first trigger to activate or context cancellation
	select {
	case triggerIndex := <-fanIn:
		logger.StateLog.Infof("State [%s] trigger [%d] activated", stateName, triggerIndex)
		nxtStateName := state.Actions[triggerIndex](stateMachine.Servers)

		if nxtStateName != stateName {
			resultState = nxtStateName
		} else {
			resultState = stateName
		}

	case <-state.Ctx.Done():
		// Context was canceled externally or by a trigger
		logger.StateLog.Infof("State [%s] shutting down due to context cancellation", stateName)
		resultState = ""
	}

	// Wait for all goroutines to finish
	wg.Wait()
	return resultState
}

func addNewStateMachine(stateInfo *ocss_context.StateInfo) int {
	self := ocss_context.GetSelf()
	idx := -1

	logger.ActionLog.Infof("Adding new state machine for App[%v], Servers[%v]", stateInfo.Name, stateInfo.Servers)
	if _, exists := self.RunningAppStateMachines[stateInfo.Name]; !exists {
		// Copy the servers slice into a new map.
		servers := make(map[string]bool, len(stateInfo.Servers))
		for _, server := range stateInfo.Servers {
			servers[server] = true
		}

		// Use copystructure to deep copy each state.
		states := make(map[string]*ocss_context.State)
		for stateName, origState := range self.AppStateMachines[stateInfo.Name].States {
			copied, err := copystructure.Copy(origState)
			if err != nil {
				logger.StateLog.Errorf("Error deep copying state %s: %v", stateName, err)
				return idx
			}

			// Attempt type assertion.
			newState, ok := copied.(*ocss_context.State)
			if !ok {
				// In case the copy returns a non-pointer type.
				st, ok := copied.(ocss_context.State)
				if !ok {
					logger.StateLog.Errorf("Error asserting deep copied state type for %s", stateName)
					return idx
				}
				newState = &st
			}

			states[stateName] = newState
		}

		// Create the new state machine with the deep-copied states and servers.
		stateMachine := &ocss_context.StateMachine{
			States:  states,
			Servers: servers,
		}

		self.RunningAppStateMachines[stateInfo.Name] = []*ocss_context.StateMachine{stateMachine}

		idx = 0

	} else {
		// TODO: merge to running state if there are common servers
	}

	return idx
}

func (s *StateController) Start(ctx context.Context, wg *sync.WaitGroup) {
	logger.StateLog.Info("State Controller is running")

	wg.Add(1)
	go func(ctx context.Context, wg *sync.WaitGroup) {
		defer wg.Done()

		s.Processor().MonitorTraffic(ctx)
	}(ctx, wg)

	self := ocss_context.GetSelf()

	self.AppStateMachines = s.setupStates(self.UserView)

	wg.Add(1)
	go func(ctx context.Context, wg *sync.WaitGroup) {
		for {
			select {
			case stateInfo := <-self.AppStateInfoChan:

				wg.Add(1)
				go func(stateInfo *ocss_context.StateInfo, states map[string]ocss_context.StateMachine) {
					appName := stateInfo.Name

					logger.StateLog.Infof("State Controller is starting for App [%s]", appName)
					defer wg.Done()

					if _, exists := states[appName]; !exists {
						logger.StateLog.Errorf("State [%s] does not exist", appName)
						return
					}

					appStateIdx := addNewStateMachine(stateInfo)
					if appStateIdx == -1 {
						logger.StateLog.Errorf("Fail to run state machine for [%s]", appName)
						return
					}

					stateMachine := self.RunningAppStateMachines[appName][appStateIdx]

					// Create an independent context for this state machine.
					smCtx, smCancel := context.WithCancel(ctx)
					// Optionally store smCtx and smCancel in your stateMachine struct if you need to cancel it later from another part of your program.
					stateMachine.Ctx = smCtx
					stateMachine.Cancel = smCancel

					for stateName, state := range stateMachine.States {
						if !state.InitState {
							logger.StateLog.Debugf("State [%s] is not an initial state", appName)
							continue
						}

						go func(currentStateName string, stateMachine *ocss_context.StateMachine) {
							for {
								nextState := s.runState(stateMachine.Ctx, currentStateName, stateMachine)
								logger.StateLog.Errorf("State [%s] transitioning to [%s]", currentStateName, nextState)

								if nextState == "" {
									break
								}

								s.Processor().UpdateForwardingTables()

								currentStateName = nextState
							}
						}(stateName, stateMachine)
					}

				}(stateInfo, self.AppStateMachines)
			case <-ctx.Done():
				wg.Done()
				return
			}
		}
	}(ctx, wg)
}
