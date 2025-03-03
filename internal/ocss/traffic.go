package ocss

import (
	"context"
	"time"

	ocss_context "github.com/comp590/ocss/internal/context"
	"github.com/comp590/ocss/internal/logger"
)

func (p *Processor) getTraffic(tor *ocss_context.ToR) ocss_context.TrafficMatrix {
	forwarder := p.Forwarder()
	self := ocss_context.GetSelf()

	sw := tor.Device
	dev, ok := self.Switches[sw]
	if !ok {
		logger.ProcessorLog.Errorf("Switch %s does not exist", sw)
		return nil
	}

	swId := dev.Id
	torId := tor.Id
	torName := tor.Name
	traffic, err := forwarder.GetTrafficMatrix(swId, torId)
	if err != nil {
		logger.ProcessorLog.Errorf("Error getting traffic matrix: %v", err)
		return nil
	}

	if _, ok := self.UserView.Traffic[torName]; !ok {
		self.UserView.Traffic[torName] = make(map[string]map[string]int)
	}

	delta := make(ocss_context.TrafficMatrix)
	for srcIp, trafficMap := range traffic {
		if _, ok := delta[srcIp]; !ok {
			delta[srcIp] = make(map[string]int)
		}

		if _, ok := self.UserView.Traffic[torName][srcIp]; !ok {
			self.UserView.Traffic[torName][srcIp] = make(map[string]int)
		}

		for dstIp, curTraffic := range trafficMap {
			if curTraffic < self.UserView.Traffic[torName][srcIp][dstIp] {
				logger.ProcessorLog.Errorf("Traffic matrix is not increasing, srcIp: %s, dstIp: %s, curTraffic: %d, prevTraffic: %d", srcIp, dstIp, curTraffic, self.UserView.Traffic[torName][srcIp][dstIp])
			} else {
				delta[srcIp][dstIp] = curTraffic - self.UserView.Traffic[torName][srcIp][dstIp]
				self.UserView.Traffic[torName][srcIp][dstIp] = curTraffic
			}
		}
	}

	return delta
}

func (p *Processor) createController() {
	p.setupSwitch()
	p.setupOCS()
	p.setupForwardingTable()
}

func (p *Processor) updateController() {
	p.setupOCS()
	p.setupForwardingTable()
}

func (p *Processor) MonitorTraffic(ctx context.Context) {
	self := ocss_context.GetSelf()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			var deltas []ocss_context.TrafficMatrix

			// Collect traffic info for each ToR
			for _, tor := range self.UserView.ToRs {
				delta := p.getTraffic(tor)
				if delta != nil {
					deltas = append(deltas, delta)
				}
			}

			// Assume flows in self.UserView.Traffic[torName] >>> app.IterTrafficMatrix
			for _, app := range self.UserView.AppServer.Apps {
				if app.Active {
					app.Lock()

					for srcIp, dstIps := range app.IterTrafficMatrix[app.Iter] {
						for dstIp, _ := range dstIps {
							delta := 0

							for _, d := range deltas {
								if _, ok := d[srcIp]; ok {
									if diff, ok := d[srcIp][dstIp]; ok && diff > 0 {
										if delta == 0 {
											delta = d[srcIp][dstIp]
										} else {
											delta = min(delta, d[srcIp][dstIp])
										}
									}
								}
							}

							app.IterTrafficMatrix[app.Iter][srcIp][dstIp] += delta

						}
					}

					app.Unlock()
				}
			}

		case <-ctx.Done():
			logger.ProcessorLog.Infof("Monitor Traffic is stopping")
			return
		}
	}
}
