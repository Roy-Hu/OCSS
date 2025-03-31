package context

import "github.com/comp590/ocss/internal/logger"

type STATE_STATUS string

const (
	RUNNING STATE_STATUS = "RUNNING"
	IDLE    STATE_STATUS = "IDLE"
)

type OP_STATUS string

const (
	START OP_STATUS = "START"
	END   OP_STATUS = "END"
)

type PolicyFunc func(op Operation) string

type Policy struct {
	StateId uint

	PolicyFunc PolicyFunc
	Servers    []string
}

type Operation struct {
	Op      string
	Servers []string
	Status  OP_STATUS

	IterTraffic []IterTraffic
	CurTraffic  TrafficMatrix
}

type State struct {
	Id uint
	// Triggers need to be match to transit into the next state
	Triggers []*Operation

	// Policies will be executed while transit into the next state
	Policies []*Policy

	OPs []*Operation

	Status STATE_STATUS
}

type OpInfo struct {
	Servers []string

	App string

	Op string

	Status OP_STATUS

	Iter int
}

type StateMachine struct {
	App string

	// states by the time order
	States     []*State
	CurStateId uint

	// Available Hardware
	Servers map[string]bool

	IterNum int

	IdGenerator uint
}

func (s *StateMachine) GenerateStateId() uint {
	s.IdGenerator++
	return s.IdGenerator
}

func (s *StateMachine) PrintStateTimeLine() {
	for _, state := range s.States {
		logger.StateLog.Errorf("State[%v]", state.Id)

		for _, op := range state.OPs {
			serverName := ""
			for _, server := range op.Servers {
				serverName += server + " "
			}
			logger.StateLog.Errorf("\t OP[%v] Servers[%v]", op.Op, serverName)
		}
	}

}
