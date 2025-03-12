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

	self.UserView.Traffic[torName] = traffic

	return self.UserView.Traffic[torName]
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
			var traffics []ocss_context.TrafficMatrix

			// Collect traffic info for each ToR
			for _, tor := range self.UserView.ToRs {
				traffic := p.getTraffic(tor)
				if traffic != nil {
					traffics = append(traffics, traffic)
				}
			}

			// Assume flows in self.UserView.Traffic[torName] >>> app.IterTrafficMatrix
			for _, app := range self.UserView.AppServer.Apps {
				if app.Active {
					app.Lock()

					for srcIp, dstIps := range app.IterTrafficMatrix[app.Iter].Start {
						for dstIp, _ := range dstIps {
							traffic := 0

							for _, t := range traffics {
								if _, ok := t[srcIp]; ok {
									if diff, ok := t[srcIp][dstIp]; ok && diff > 0 {
										if traffic == 0 {
											traffic = t[srcIp][dstIp]
										} else {
											traffic = min(traffic, t[srcIp][dstIp])
										}
									}
								}
							}
							if app.IterTrafficMatrix[app.Iter].Init {
								app.IterTrafficMatrix[app.Iter].Start[srcIp][dstIp] = traffic
								app.IterTrafficMatrix[app.Iter].Init = false
							} else {
								app.IterTrafficMatrix[app.Iter].End[srcIp][dstIp] = traffic
							}
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
