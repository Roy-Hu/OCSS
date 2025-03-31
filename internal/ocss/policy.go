package ocss

import (
	"sort"

	ocss_context "github.com/comp590/ocss/internal/context"

	"github.com/comp590/ocss/internal/logger"
)

func initPolicies(user *ocss_context.UserView) map[string]ocss_context.PolicyFunc {

	MyActions := make(map[string]ocss_context.PolicyFunc)
	// Init end

	MyActions["allreduce"] = ocss_context.PolicyFunc(func(op ocss_context.Operation) string {
		newInPort := []int{}
		newOutPort := []int{}

		servers := make(map[string]*ocss_context.Server)
		serverMap := make(map[string]bool)
		for _, server := range op.Servers {
			serverMap[server] = true
		}

		for server, ok := range serverMap {
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

		linkTraffics := op.CurTraffic.GetTopKLinkTraffic(ringLen)

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
	})

	return MyActions
}
