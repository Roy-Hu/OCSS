package ocss

import (
	"context"
	"sort"

	ocss_context "github.com/comp590/ocss/internal/context"

	"github.com/comp590/ocss/internal/logger"
)

func (s *StateController) setupStates(user *ocss_context.UserView) map[string]ocss_context.StateMachine {

	// These Initialization is needed for the code to run
	// Init start
	stateMachine := make(map[string]ocss_context.StateMachine)

	MyActions := make(map[string]func(Servers map[string]bool) string)
	MyTriggers := make(map[string]func(ctx context.Context, servers map[string]bool) bool)
	// Init end

	MyTriggers["initAllReduce"] = func(ctx context.Context, servers map[string]bool) bool {
		return s.MonitorApp("allreduce", 1)(ctx)
	}

	MyActions["initAllReduce"] = func(Servers map[string]bool) string {

		newInPort := []int{}
		newOutPort := []int{}

		servers := make(map[string]*ocss_context.Server)

		for server, ok := range Servers {
			if ok {
				servers[server] = user.Servers[server]
			}
		}
		ringLen := len(servers)

		torPortCnt := make(map[string][]int)
		for _, server := range servers {
			// TODO: currently assume server used only one port

			for _, connTo := range server.PortConnToMap {
				// torName := connTo.Device
				logger.ActionLog.Debugf("Server: %v, connTo: %v", server.Name, connTo)
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

		sort.Slice(tors, func(i, j int) bool {
			return len(torPortCnt[tors[i]]) > len(torPortCnt[tors[j]])
		})

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
				return ""
			}

			ring = append(ring, cur)
		}

		// assume the ring starts at Link[i].src
		for i := 0; i < ringLen; {
			// always allocate server to the tor with the largest number of ports

			for _, ports := range torPortCnt {
				for _, port := range ports {

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

		user.OCSs["ocs_edge"].UpdateConn(newConn)

		return "initAllReduce"
	}

	states := make(map[string]*ocss_context.State)

	states["initAllReduce"] = &ocss_context.State{
		Triggers: []func(ctx context.Context, servers map[string]bool) bool{
			MyTriggers["initAllReduce"],
		},
		Actions: []func(Servers map[string]bool) string{
			MyActions["initAllReduce"],
		},
		InitState: true,
	}

	stateMachine["allreduce"] = ocss_context.StateMachine{
		States:  states,
		Servers: make(map[string]bool),
	}

	return stateMachine
}
