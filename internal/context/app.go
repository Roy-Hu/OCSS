package context

import (
	"strconv"
	"sync"

	"github.com/comp590/ocss/internal/logger"
	"github.com/comp590/ocss/internal/util"
)

type AppServer struct {
	Address string
	Apps    map[string]*App
}

type App struct {
	sync.RWMutex

	AppId             string
	IterTrafficMatrix map[int]TrafficMatrix
	Active            bool
	Iter              int

	MonitoredIter []bool
	FinishedIter  []chan int
}

func (a *App) ConstructHeapMap() {
	traffic := make(map[string]map[string]int)
	for srcIp, dstIps := range a.IterTrafficMatrix[a.Iter] {
		for dstIp, vol := range dstIps {
			src_server := ocssContext.IpToServer[srcIp]
			dst_server := ocssContext.IpToServer[dstIp]

			if _, ok := traffic[src_server]; !ok {
				traffic[src_server] = make(map[string]int)
			}

			traffic[src_server][dst_server] = vol
		}
	}

	logger.HttpLog.Infof("Creating heatmap for app %s, iter %d, traffic %v", a.AppId, a.Iter, traffic)
	util.CreateHeatmap(traffic, a.AppId+"_"+strconv.Itoa(a.Iter))
}

// func (a *AppView) GetStartAppTraffic() {
// 	self := GetSelf()

// 	if a.Active {
// 		logger.ProcessorLog.Debugf("App %s, Iter %d, Traffic Matrix %v", a.AppId, a.Iter, a.IterTrafficMatrix[a.Iter])
// 		for srcIp, dstIps := range a.IterTrafficMatrix[a.Iter] {
// 			if _, ok := self.Traffic[srcIp]; ok {
// 				for dstIp, _ := range dstIps {
// 					if _, ok2 := self.Traffic[srcIp][dstIp]; ok2 {
// 						a.IterTrafficMatrix[a.Iter][srcIp][dstIp] = self.Traffic[srcIp][dstIp]
// 					}
// 				}
// 			}
// 		}
// 	}
// }

// func (a *AppView) GetEndAppTraffic() {
// 	self := GetSelf()

// 	if a.Active {
// 		logger.ProcessorLog.Debugf("App %s, Iter %d, Traffic Matrix %v", a.AppId, a.Iter, a.IterTrafficMatrix[a.Iter])
// 		for srcIp, dstIps := range a.IterTrafficMatrix[a.Iter] {
// 			if _, ok := self.Traffic[srcIp]; ok {
// 				for dstIp, _ := range dstIps {
// 					if _, ok2 := self.Traffic[srcIp][dstIp]; ok2 {
// 						a.IterTrafficMatrix[a.Iter][srcIp][dstIp] = self.Traffic[srcIp][dstIp]
// 					}
// 				}
// 			}
// 		}

// 		logger.ActionLog.Errorf("App %s, Iter %d, Traffic Matrix %v", a.AppId, a.Iter, self.Traffic)
// 	}
// }
