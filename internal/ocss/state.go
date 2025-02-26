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

		logger.ActionLog.Infof("State1: %v", newConn)
		user.OCSs["ocs_edge"].UpdateConn(newConn)

		return "State1"
	}

	MyActions["AllReduce"] = func() string {
		newConn := &ocss_context.Connection{
			In_port:  user.OCSs["ocs_edge"].Conn.In_port,
			Out_port: []int{65, 71, 67, 69, 66, 72},
		}

		logger.ActionLog.Infof("State1: %v", newConn)
		user.OCSs["ocs_edge"].UpdateConn(newConn)

		return "State1"
	}

	states["State1"] = &ocss_context.State{
		Triggers: []func(ctx context.Context) bool{
			func(ctx context.Context) bool {
				ticker := time.NewTicker(30000 * time.Second)
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
			MyActions["AllReduce"],
			func() string {
				return "State1"
			},
		},

		InitState: true,
	}

	return states
}
