package context

import (
	"strconv"

	"github.com/comp590/ocss/internal/util"
)

type App struct {
	Address string
	View    map[string]*AppView
}
type AppView struct {
	AppId             string
	IterTrafficMatrix map[int]TrafficMatrix
	Active            bool
	Iter              int
}

func (a *AppView) ConstructHeapMap() {
	traffic := make(map[string]map[string]int)
	for srcIp, dstIps := range a.IterTrafficMatrix[a.Iter] {
		for dstIp, vol := range dstIps {
			if _, ok := traffic[srcIp]; !ok {
				traffic[srcIp] = make(map[string]int)
			}

			src_server := ocssContext.IpToServer[srcIp]
			dst_server := ocssContext.IpToServer[dstIp]

			traffic[src_server][dst_server] = vol
		}
	}

	util.CreateHeatmap(traffic, a.AppId+"_"+strconv.Itoa(a.Iter))
}
