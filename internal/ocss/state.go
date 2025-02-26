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

		// traffic := s.GetTraffic("tor1")
		// for srcIp, dstIpByte := range traffic {
		// 	for dstIp, bytes := range dstIpByte {
		// 		logger.ActionLog.Infof("Traffic from %s to %s: %d bytes", srcIp, dstIp, bytes)
		// 	}
		// }

		logger.ActionLog.Infof("State1: %v", newConn)
		user.OCSs["ocs_edge"].UpdateConn(newConn)

		return "State1"
	}

	// MyActions["2ToRS2"] = func() string {
	// 	newConn := &ocss_context.Connection{
	// 		In_port:  user.OCSs["ocs_edge"].Conn.In_port,
	// 		Out_port: user.OCSs["ocs_edge"].Conn.Out_port,
	// 	}

	// 	first_out_port := user.OCSs["ocs_edge"].Conn.Out_port[0]

	// 	for i := 0; i < len(user.OCSs["ocs_edge"].Conn.Out_port)-1; i++ {
	// 		newConn.Out_port[i] = user.OCSs["ocs_edge"].Conn.Out_port[i+1]
	// 	}

	// 	newConn.Out_port[len(user.OCSs["ocs_edge"].Conn.Out_port)-1] = first_out_port

	// 	// traffic := s.GetTraffic("tor1")
	// 	// for srcIp, dstIpByte := range traffic {
	// 	// 	for dstIp, bytes := range dstIpByte {
	// 	// 		logger.ActionLog.Infof("Traffic from %s to %s: %d bytes", srcIp, dstIp, bytes)
	// 	// 	}
	// 	// }

	// 	logger.ActionLog.Infof("State1: %v", newConn)
	// 	user.OCSs["ocs_edge"].UpdateConn(newConn)

	// 	return "State2"
	// }

	// // Right shift
	// MyActions["2ToRS3"] = func() string {
	// 	newConn := &ocss_context.Connection{
	// 		In_port:  user.OCSs["ocs_edge"].Conn.In_port,
	// 		Out_port: user.OCSs["ocs_edge"].Conn.Out_port,
	// 	}

	// 	first_out_port := user.OCSs["ocs_edge"].Conn.Out_port[0]

	// 	for i := 0; i < len(user.OCSs["ocs_edge"].Conn.Out_port)-1; i++ {
	// 		newConn.Out_port[i] = user.OCSs["ocs_edge"].Conn.Out_port[i+1]
	// 	}

	// 	newConn.Out_port[len(user.OCSs["ocs_edge"].Conn.Out_port)-1] = first_out_port

	// 	// traffic := s.GetTraffic("tor1")
	// 	// for srcIp, dstIpByte := range traffic {
	// 	// 	for dstIp, bytes := range dstIpByte {
	// 	// 		logger.ActionLog.Infof("Traffic from %s to %s: %d bytes", srcIp, dstIp, bytes)
	// 	// 	}
	// 	// }

	// 	logger.ActionLog.Infof("State1: %v", newConn)
	// 	user.OCSs["ocs_edge"].UpdateConn(newConn)

	// 	return "State3"
	// }

	// MyActions["2ToRS4"] = func() string {
	// 	newConn := &ocss_context.Connection{
	// 		In_port:  user.OCSs["ocs_edge"].Conn.In_port,
	// 		Out_port: user.OCSs["ocs_edge"].Conn.Out_port,
	// 	}

	// 	first_out_port := user.OCSs["ocs_edge"].Conn.Out_port[0]

	// 	for i := 0; i < len(user.OCSs["ocs_edge"].Conn.Out_port)-1; i++ {
	// 		newConn.Out_port[i] = user.OCSs["ocs_edge"].Conn.Out_port[i+1]
	// 	}

	// 	newConn.Out_port[len(user.OCSs["ocs_edge"].Conn.Out_port)-1] = first_out_port

	// 	// traffic := s.GetTraffic("tor1")
	// 	// for srcIp, dstIpByte := range traffic {
	// 	// 	for dstIp, bytes := range dstIpByte {
	// 	// 		logger.ActionLog.Infof("Traffic from %s to %s: %d bytes", srcIp, dstIp, bytes)
	// 	// 	}
	// 	// }

	// 	logger.ActionLog.Infof("State1: %v", newConn)
	// 	user.OCSs["ocs_edge"].UpdateConn(newConn)

	// 	return "State4"
	// }

	// stage := 0
	// out_port := [][]int{
	// 	{13, 30},
	// 	{30, 10},
	// 	{10, 13},
	// }
	// MyActions["4ToR"] = func() string {
	// 	stage++

	// 	newConn := &ocss_context.Connection{
	// 		In_port:  []int{25, 26},
	// 		Out_port: out_port[stage%3],
	// 	}

	// 	logger.ActionLog.Errorf("State1: %v", newConn)
	// 	user.OCSs["ocs_core"].UpdateConn(newConn)

	// 	return "State1"
	// }

	// initThreshold := 10000

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
			// createThresholdTrigger("tor1", initThreshold),
		},
		Actions: []func() string{
			MyActions["2ToR"],
			func() string {
				return "State1"
			},
		},
		// Vars: map[string]interface{}{
		// 	"threshold": initThreshold,
		// },

		InitState: true,
	}

	// states["State2"] = &ocss_context.State{
	// 	Triggers: []func(ctx context.Context) bool{
	// 		func(ctx context.Context) bool {
	// 			ticker := time.NewTicker(3000 * time.Millisecond)
	// 			defer ticker.Stop()

	// 			select {
	// 			case <-ticker.C:
	// 				return true
	// 			case <-ctx.Done():
	// 				return false
	// 			}
	// 		},
	// 		createThresholdTrigger("tor1", states["State1"].Vars["threshold"].(int)*2),
	// 	},
	// 	Actions: []func() string{
	// 		MyActions["2ToRS2"],
	// 		func() string {
	// 			threshold := states["State2"].Vars["threshold"].(int)
	// 			threshold *= 2
	// 			states["State2"].Vars["threshold"] = threshold

	// 			states["State2"].Triggers[1] = createThresholdTrigger("tor1", states["State2"].Vars["threshold"].(int))
	// 			return "State2"
	// 		},
	// 	},
	// 	Vars: map[string]interface{}{
	// 		"threshold": states["State1"].Vars["threshold"].(int) * 2,
	// 	},
	// 	InitState: false,
	// }

	// states["State3"] = &ocss_context.State{
	// 	Triggers: []func(ctx context.Context) bool{
	// 		func(ctx context.Context) bool {
	// 			ticker := time.NewTicker(1000 * time.Millisecond)
	// 			defer ticker.Stop()

	// 			select {
	// 			case <-ticker.C:
	// 				return true
	// 			case <-ctx.Done():
	// 				return false
	// 			}
	// 		},
	// 		createThresholdTrigger("tor1", initThreshold),
	// 	},
	// 	Actions: []func() string{
	// 		MyActions["2ToRS3"],
	// 		func() string {
	// 			return "State4"
	// 		},
	// 	},
	// 	Vars: map[string]interface{}{
	// 		"threshold": initThreshold * 2,
	// 	},

	// 	InitState: true,
	// }

	// states["State4"] = &ocss_context.State{
	// 	Triggers: []func(ctx context.Context) bool{
	// 		func(ctx context.Context) bool {
	// 			ticker := time.NewTicker(1000 * time.Millisecond)
	// 			defer ticker.Stop()

	// 			select {
	// 			case <-ticker.C:
	// 				return true
	// 			case <-ctx.Done():
	// 				return false
	// 			}
	// 		},
	// 		createThresholdTrigger("tor1", states["State3"].Vars["threshold"].(int)*2),
	// 	},
	// 	Actions: []func() string{
	// 		MyActions["2ToRS4"],
	// 		func() string {
	// 			threshold := states["State4"].Vars["threshold"].(int)
	// 			threshold *= 2
	// 			states["State4"].Vars["threshold"] = threshold

	// 			states["State4"].Triggers[1] = createThresholdTrigger("tor1", states["State4"].Vars["threshold"].(int))
	// 			return "State4"
	// 		},
	// 	},
	// 	Vars: map[string]interface{}{
	// 		"threshold": states["State3"].Vars["threshold"].(int) * 2,
	// 	},
	// 	InitState: false,
	// }

	return states
}
