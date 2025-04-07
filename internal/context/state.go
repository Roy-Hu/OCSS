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

type PolicyFunc func(torCapacity map[string]int, op *Operation) map[string]string

type Policy struct {
	PolicyFunc PolicyFunc
	Op         *Operation
}

type Operation struct {
	Op      string
	Servers []string
	Status  OP_STATUS

	IterTraffic []IterTraffic
}

type State struct {
	Id uint
	// Triggers need to be match to transit into the next state
	Triggers []*Operation

	// Policies will be executed while transit into the next state
	Policies []*Policy

	OPs []*Operation

	IterTraffic []IterTraffic

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
	States        []*State
	defaultPolicy *Policy

	CurStateId uint

	// Available ToRPorts
	ServerToRPort map[string]*ConnectedTo

	IterNum int

	IdGenerator uint
}

func (s *StateMachine) ImplementDecisions(state int, torPorts map[string][]int, decision map[string]string, confict map[string]bool) {
	self := GetSelf()
	for sever, conf := range confict {
		if conf {
			delete(decision, sever)
		}
	}

	op := &Operation{
		IterTraffic: s.States[state].IterTraffic,
	}

	usedToRPortCnt := make(map[string]int)
	for _, tor := range decision {
		usedToRPortCnt[tor]++
	}

	remainingPorts := make(map[string]int)
	for tor, ports := range torPorts {
		remainingPorts[tor] = len(ports) - usedToRPortCnt[tor]
	}

	confictDecision := s.defaultPolicy.PolicyFunc(remainingPorts, op)

	for server, tor := range confictDecision {
		if _, ok := decision[server]; !ok {
			decision[server] = tor
		} else {
			logger.StateLog.Errorf("Server[%v] already assigned to TOR[%v]", server, decision[server])
		}
	}

	updateConn := &Connection{
		In_port:  []int{},
		Out_port: []int{},
	}

	torServerMap := make(map[string][]string)
	for server, tor := range decision {
		if _, ok := torServerMap[tor]; !ok {
			torServerMap[tor] = []string{}
		}
		torServerMap[tor] = append(torServerMap[tor], server)
	}

	for tor, servers := range torServerMap {
		curServerIdx := 0
		for _, port := range torPorts[tor] {
			if curServerIdx >= len(servers) {
				break
			}
			inPort := -1
			outPort := -1

			for _, connTo := range self.Servers[servers[curServerIdx]].PortConnToMap {
				if self.DeviceType[connTo.Device] == OPTICAL_SWITCH {
					inPort = connTo.Port
				} else {
					logger.StateLog.Errorf("Server[%v] connect to non-optical switch device[%v]", servers[curServerIdx], connTo.Device)
					continue
				}
			}

			torConnTo := self.Servers[tor].PortConnToMap[port]
			if self.DeviceType[torConnTo.Device] == OPTICAL_SWITCH {
				outPort = torConnTo.Port
			} else {
				logger.StateLog.Errorf("Server[%v] connect to non-optical switch device[%v]", tor, torConnTo.Device)
				continue
			}

			updateConn.In_port = append(updateConn.In_port, inPort)
			updateConn.Out_port = append(updateConn.Out_port, outPort)

			curServerIdx++
		}
	}

	logger.StateLog.Errorf("Update connection: %v", updateConn)
	self.OCSs["ocs_edge"].UpdateConn(updateConn)
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
