package ocss

import (
	"context"
	"os"
	"sort"

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

		newInPort := []int{}
		newOutPort := []int{}

		ringLen := len(user.Servers)

		torPortCnt := make(map[string][]int)
		for _, server := range user.Servers {
			// TODO: currently assume server used only one port

			for _, connTo := range server.PortConnToMap {
				// torName := connTo.Device
				logger.ActionLog.Warnf("Server: %v, connTo: %v", server.Name, connTo)
				torName := server.ConnToR
				if _, ok := torPortCnt[torName]; !ok {
					for _, connTo := range user.ToRs[torName].PortConnToMap {
						if connTo.Name == "ocs_edge" {
							torPortCnt[torName] = append(torPortCnt[torName], connTo.Port)
						}

					}
				}
				break
			}
		}

		// Extract keys from the map.
		tors := make([]string, 0, len(torPortCnt))
		for tor := range torPortCnt {
			tors = append(tors, tor)
		}

		// Sort keys based on the count value.
		// For ascending order (lowest to highest), use "<".
		// For descending order (highest to lowest), use ">".
		sort.Slice(tors, func(i, j int) bool {
			return len(torPortCnt[tors[i]]) > len(torPortCnt[tors[j]])
		})

		for tor, ports := range torPortCnt {
			logger.ActionLog.Warnf("Tor: %v, Ports: %v", tor, ports)
		}

		linkTraffics := user.GetTopKLinkTraffic("allreduce", 1, ringLen)

		cur := linkTraffics[0]
		ring := []ocss_context.TrafficPair{}

		for i := 0; i < ringLen; i++ {
			findNext := false
			for _, link := range linkTraffics {
				if cur.Dst == link.Src {
					cur = link
					findNext = true

					break
				}
			}

			if !findNext {
				logger.ActionLog.Errorf("Cannot form a ring")
				return "State1"
			}

			ring = append(ring, cur)
			logger.ActionLog.Warnf("Server: %v", cur.Src)
		}

		// assume the ring starts at Link[i].src
		for i := 0; i < ringLen; {
			// always allocate server to the tor with the largest number of ports

			for torName, ports := range torPortCnt {
				logger.ActionLog.Warnf("Tor: %v, Ports: %v", torName, ports)
				for _, port := range ports {
					logger.ActionLog.Warnf("PortConnToMap: %v", user.ToRs[torName].PortConnToMap)
					for _, connTo := range user.ToRs[torName].PortConnToMap {
						logger.ActionLog.Warnf("ConnTo: %v", connTo)
					}

					ocsOutPort := port

					server := user.Servers[ring[i].Src]

					// TODO: currently assume server used only one port
					for _, connTo := range server.PortConnToMap {
						ocsInPort := connTo.Port

						newInPort = append(newInPort, ocsInPort)
						newOutPort = append(newOutPort, ocsOutPort)

						break
					}
					i++

					if i == ringLen {
						break
					}
				}
			}

		}
		newConn := &ocss_context.Connection{
			In_port:  newInPort,
			Out_port: newOutPort,
		}

		logger.ActionLog.Infof("State1: %v", newConn)

		os.Exit(0)

		user.OCSs["ocs_edge"].UpdateConn(newConn)

		return "State1"
	}

	states["State1"] = &ocss_context.State{
		Triggers: []func(ctx context.Context) bool{
			s.MonitorApp("allreduce", 1),
		},
		Actions: []func() string{
			MyActions["AllReduce"],
			func() string {
				return "State1"
			},
		},

		InitState: true,
	}

	// states["State1"] = &ocss_context.State{
	// 	Triggers: []func(ctx context.Context) bool{
	// 		func(ctx context.Context) bool {
	// 			ticker := time.NewTicker(30000 * time.Second)
	// 			defer ticker.Stop()

	// 			select {
	// 			case <-ticker.C:
	// 				return true
	// 			case <-ctx.Done():
	// 				return false
	// 			}
	// 		},
	// 	},
	// 	Actions: []func() string{
	// 		MyActions["AllReduce"],
	// 		func() string {
	// 			return "State1"
	// 		},
	// 	},

	// 	InitState: true,
	// }

	return states
}
