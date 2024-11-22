package ocss

import (
	"context"
	"time"

	ocss_context "github.com/comp590/ocss/internal/context"

	"github.com/comp590/ocss/internal/logger"
)

func (s *StateController) setupStates(user *ocss_context.UserView) map[string]*ocss_context.State {

	states := make(map[string]*ocss_context.State)
	MyActions := make(map[string]func() string)

	// Right shift
	MyActions["2ToR"] = func() string {
		newConn := &ocss_context.Connection{
			In_port:  user.OCSs["ocs_edge"].Conn.In_port,
			Out_port: user.OCSs["ocs_edge"].Conn.Out_port,
		}

		first_out_port := user.OCSs["ocs_edge"].Conn.Out_port[0]

		for i := 0; i < len(user.OCSs["ocs_edge"].Conn.Out_port)-1; i++ {
			newConn.Out_port[i] = user.OCSs["ocs_edge"].Conn.Out_port[i+1]
		}

		newConn.Out_port[len(user.OCSs["ocs_edge"].Conn.Out_port)-1] = first_out_port

		traffic := s.GetTraffic("tor1")
		for srcIp, dstIpByte := range traffic {
			for dstIp, bytes := range dstIpByte {
				logger.ActionLog.Infof("Traffic from %s to %s: %d bytes", srcIp, dstIp, bytes)
			}
		}

		logger.ActionLog.Infof("State1: %v", newConn)
		user.OCSs["ocs_edge"].UpdateConn(newConn)

		return "State1"
	}

	stage := 0
	out_port := [][]int{
		{13, 30},
		{30, 10},
		{10, 13},
	}
	MyActions["4ToR"] = func() string {
		stage++

		newConn := &ocss_context.Connection{
			In_port:  []int{25, 26},
			Out_port: out_port[stage%3],
		}

		logger.ActionLog.Errorf("State1: %v", newConn)
		user.OCSs["ocs_core"].UpdateConn(newConn)

		return "State1"
	}

	states["State1"] = &ocss_context.State{
		Triggers: []func(ctx context.Context) bool{
			func(ctx context.Context) bool {
				ticker := time.NewTicker(500 * time.Millisecond)
				defer ticker.Stop()

				select {
				case <-ticker.C:
					return true
				case <-ctx.Done():
					return false
				}
			},
		},
		Actions: []func() string{
			MyActions["2ToR"],
		},
		InitState: true,
	}

	return states
}
