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

func matchOp(op1 *ocss_context.Operation, op2 *ocss_context.Operation) bool {
	if op1.Op == op2.Op {
		matchServer := make(map[string]bool)
		for _, server := range op2.Servers {
			matchServer[server] = true
		}

		for _, server := range op1.Servers {
			if _, ok := matchServer[server]; !ok {
				return false
			}
		}

		return true
	}

	return false
}

func (s *StateController) addOptoState(stateMachine *ocss_context.StateMachine, op *ocss_context.Operation) {
	self := ocss_context.GetSelf()

	curStateId := stateMachine.CurStateId
	curTraffic := s.Processor().GetFlowTraffic(op.Servers)

	for _, serverName := range op.Servers {
		stateMachine.ServerToRPort[serverName] = self.UserView.Servers[serverName].ConnToRPort
	}
	// Start CC Operation
	if op.Status == ocss_context.START {
		op.IterTraffic = append(op.IterTraffic,
			ocss_context.IterTraffic{
				Start: curTraffic,
			},
		)

		if int(curStateId) < len(stateMachine.States) && stateMachine.States[curStateId].Status == ocss_context.RUNNING {
			// If there are CC operation running, we need to add the new operation to the current state
			stateMachine.States[curStateId].OPs = append(stateMachine.States[curStateId].OPs, op)
			logger.StateLog.Infof("State[%v] is running, merge OP[%v] to current state for APP[%v]", curStateId, op.Op, stateMachine.App)
			logger.StateLog.Debugf("Add Trigger OP[%v] to State[%d]for APP[%v]", op.Op, curStateId-1, stateMachine.App)
			stateMachine.States[curStateId-1].Triggers = append(stateMachine.States[curStateId].Triggers, op)
			if policyFunc, ok := self.Policies[op.Op]; ok {
				stateMachine.States[curStateId-1].Policies = append(stateMachine.States[curStateId-1].Policies, &ocss_context.Policy{
					PolicyFunc: policyFunc,
					Op:         op,
				})
			} else {
				logger.StateLog.Warnf("No policy found for OP[%v]", op.Op)
			}
		} else {
			// If there is no CC operation running, we need to create a new state
			id := stateMachine.GenerateStateId()
			logger.StateLog.Infof("State[%v] is idle, create new state[%d] for APP[%v]", stateMachine.CurStateId, id, stateMachine.App)
			newState := &ocss_context.State{
				Id:     id,
				Status: ocss_context.RUNNING,
				OPs:    []*ocss_context.Operation{op},
			}

			stateMachine.States[curStateId].Triggers = append(stateMachine.States[curStateId].Triggers, op)
			if policyFunc, ok := self.Policies[op.Op]; ok {
				stateMachine.States[curStateId].Policies = append(stateMachine.States[curStateId].Policies, &ocss_context.Policy{
					PolicyFunc: policyFunc,
					Op:         op,
				})
			}

			stateMachine.States = append(stateMachine.States, newState)
			stateMachine.CurStateId = id
			stateMachine.States[stateMachine.CurStateId].IterTraffic = append(stateMachine.States[stateMachine.CurStateId].IterTraffic,
				ocss_context.IterTraffic{
					Start: curTraffic,
				},
			)
		}
	} else if op.Status == ocss_context.END {
		for _, stateOp := range stateMachine.States[curStateId].OPs {
			if matchOp(stateOp, op) {
				stateOp.IterTraffic[stateMachine.IterNum-1].End = curTraffic
				stateOp.Status = ocss_context.END
				break
			}
		}

		for _, stateOp := range stateMachine.States[curStateId].OPs {
			if stateOp.Status != ocss_context.END {
				return
			}
		}
		stateMachine.States[curStateId].IterTraffic[0].End = curTraffic
		logger.StateLog.Infof("State[%v] is finished, transit", curStateId)
		stateMachine.States[curStateId].Status = ocss_context.IDLE
	} else {
		logger.StateLog.Warnf("Invalid Status[%v] for Operation[%v]", op.Status, op.Op)
	}
}

func (s *StateController) stateTransition(stateMachine *ocss_context.StateMachine, op *ocss_context.Operation) {
	curStateId := stateMachine.CurStateId
	curState := stateMachine.States[curStateId]

	curTraffic := s.Processor().GetFlowTraffic(op.Servers)
	if op.Status == ocss_context.START {
		runningOp := 0
		for _, curOp := range curState.OPs {
			if curOp.Status == ocss_context.START {
				runningOp++
			}
		}

		if runningOp > 1 {
			logger.StateLog.Errorf("More than one operation is running which state transition is not supported yet")
			return
		}

		logger.StateLog.Infof("State[%v] is running, curStateId[%v] for APP[%v]", curStateId, stateMachine.CurStateId, stateMachine.App)
		logger.ActionLog.Infof("Trigger OP[%d] for APP[%v]", len(curState.Triggers), stateMachine.App)

		for _, triggerOp := range curState.Triggers {
			if matchOp(triggerOp, op) {
				logger.StateLog.Infof("State[%v] is triggered by OP[%v] for APP[%v]", curStateId, op.Op, stateMachine.App)
				decisions := make(map[string]string)
				conflict := make(map[string]bool)

				for _, policy := range curState.Policies {
					logger.ActionLog.Errorf("policies run for op[%v] at state[%d]", triggerOp.Op, curStateId)
					logger.ActionLog.Errorf("policy: %v", policy.Op.Servers)

					// use the traffic from the prev iteration to do the policy
					torPorts := make(map[string]int)
					for _, connTo := range stateMachine.ServerToRPort {
						// each server should connect to a unique tor port, so the ports in torPortsp[connTo.Device] should be unique
						torPorts[connTo.Name]++
					}

					newDecision := policy.PolicyFunc(torPorts, policy.Op)
					logger.ActionLog.Errorf("Decision: %v", newDecision)
					findConflict(decisions, newDecision, conflict)
					logger.ActionLog.Errorf("Decision: %v", decisions)
				}
				stateMachine.CurStateId = curStateId + 1

				stateMachine.States[curStateId].IterTraffic = append(stateMachine.States[curStateId].IterTraffic,
					ocss_context.IterTraffic{
						Start: curTraffic,
					},
				)

				stateMachine.States[curStateId].Status = ocss_context.RUNNING
				break
			}
		}
	}

	for _, stateOp := range stateMachine.States[stateMachine.CurStateId].OPs {
		if matchOp(stateOp, op) {
			if len(stateOp.IterTraffic) != stateMachine.IterNum-1 {
				logger.StateLog.Errorf("Invalid Iter Traffic for APP[%v] Operation[%v] Server[%v]", stateMachine.App, op.Op, op.Servers)

				return
			}

			switch stateOp.Status {
			case ocss_context.START:
				stateOp.IterTraffic = append(stateOp.IterTraffic,
					ocss_context.IterTraffic{
						Start: curTraffic,
					},
				)
				stateOp.Status = ocss_context.START
			case ocss_context.END:
				stateOp.IterTraffic[stateMachine.IterNum-2].End = curTraffic
				stateOp.Status = ocss_context.END

				allEnd := true
				for _, stateOp := range stateMachine.States[stateMachine.CurStateId].OPs {
					if stateOp.Status != ocss_context.END {
						allEnd = false
						break
					}
				}

				if allEnd {
					logger.StateLog.Infof("State[%v] is finished", stateMachine.CurStateId)
					stateMachine.States[stateMachine.CurStateId].Status = ocss_context.IDLE
					stateMachine.States[stateMachine.CurStateId].IterTraffic[len(stateMachine.States[stateMachine.CurStateId].IterTraffic)-1].End = curTraffic
				}
			}

			break
		}
	}
}

func findConflict(prevDecision map[string]string, newDecision map[string]string, confict map[string]bool) {
	for server, tor := range newDecision {
		if prevTor, ok := prevDecision[server]; ok {
			if prevTor != tor {
				confict[server] = true
			}
		}

		prevDecision[server] = tor
	}
}

func addNewStateMachine(app string, op *ocss_context.Operation) {
	self := ocss_context.GetSelf()

	logger.ActionLog.Infof("Adding new state machine for App[%v], Op[%vsServers[%v]", app, op.Op, op.Servers)
	if _, exists := self.RunningAppStateMachines[app]; !exists {
		// Copy the servers slice into a new map.
		servers := make(map[string]bool, len(op.Servers))
		for _, server := range op.Servers {
			servers[server] = true
		}

		// Create the new state machine with the deep-copied states and servers.
		self.RunningAppStateMachines[app] = &ocss_context.StateMachine{
			App:           app,
			ServerToRPort: make(map[string]*ocss_context.ConnectedTo),
			CurStateId:    0,
			IdGenerator:   0,
			IterNum:       1,
		}

		initState := &ocss_context.State{
			Id:     0,
			Status: ocss_context.IDLE,
		}
		self.RunningAppStateMachines[app].States = append(self.RunningAppStateMachines[app].States, initState)
	} else {
		// TODO: support multiple state machines for the same app
		logger.ActionLog.Warnf("State machine already exists for App[%v]", app)
	}
}

func (s *StateController) Start(ctx context.Context, wg *sync.WaitGroup) {
	logger.StateLog.Info("State Controller is running")

	self := ocss_context.GetSelf()

	self.Policies = initPolicies(self.UserView)

	wg.Add(1)
	go func(ctx context.Context, wg *sync.WaitGroup) {
		defer wg.Done()

		// We assume in each iteration of the app, the order of operation is fixed
		for {
			select {
			case opInfo := <-self.OpInfoChan:
				appName := opInfo.App
				iter := opInfo.Iter

				op := &ocss_context.Operation{
					Op:      opInfo.Op,
					Servers: opInfo.Servers,
					Status:  opInfo.Status,
				}

				logger.StateLog.Infof("State Controller is starting for App [%s] Op[%s] Status [%v] Servers[%v]", appName, op.Op, op.Status, op.Servers)

				if _, exists := self.RunningAppStateMachines[appName]; !exists {
					if iter != 1 {
						logger.ActionLog.Error("State Machines needs to be create in iter 1 for now")
						return
					}

					addNewStateMachine(appName, op)
				}

				stateMachine := self.RunningAppStateMachines[appName]
				// Contruct the order of and states in iter 1
				if opInfo.Iter == 1 {
					s.addOptoState(stateMachine, op)
				} else {
					if stateMachine.IterNum != opInfo.Iter {
						stateMachine.CurStateId = 0
					}

					stateMachine.IterNum = opInfo.Iter

					s.stateTransition(stateMachine, op)
				}

				// stateMachine.PrintStateTimeLine()

			case <-ctx.Done():
				logger.StateLog.Info("State Controller is stopping")
				return
			}
		}
	}(ctx, wg)
}
