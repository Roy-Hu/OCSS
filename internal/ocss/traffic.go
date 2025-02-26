package ocss

import (
	"context"
	"time"

	ocss_context "github.com/comp590/ocss/internal/context"
	"github.com/comp590/ocss/internal/logger"
)

func (p *Processor) getTraffic(swId int, torId int) int {
	forwarder := p.Forwarder()
	self := ocss_context.GetSelf()

	traffic, err := forwarder.GetTrafficMatrix(swId, torId)
	if err != nil {
		logger.ProcessorLog.Errorf("Error getting traffic matrix: %v", err)
		return 0
	}

	tol := 0

	delta := make(map[string]map[string]int)
	for srcIp, trafficMap := range traffic {
		for dstIp, traffic := range trafficMap {
			if _, ok := delta[srcIp]; !ok {
				delta[srcIp] = make(map[string]int)
			}

			if _, ok := (*self.UserView.Traffic)[srcIp][dstIp]; ok {
				delta[srcIp][dstIp] += traffic - (*self.UserView.Traffic)[srcIp][dstIp]
			} else {
				delta[srcIp][dstIp] += traffic
			}

			tol += traffic
		}
	}

	self.UserView.Traffic = &traffic

	for _, app := range self.UserView.AppServer.Apps {
		if app.Active {
			logger.ProcessorLog.Debugf("App %s, Iter %d, Traffic Matrix %v", app.AppId, app.Iter, app.IterTrafficMatrix[app.Iter])
			for srcIp, dstIps := range app.IterTrafficMatrix[app.Iter] {
				if _, ok := delta[srcIp]; !ok {
					continue
				}
				for dstIp := range dstIps {
					if _, ok := delta[srcIp][dstIp]; ok {
						logger.ProcessorLog.Debugf("srcIp %s, dstIp %s, delta %d", srcIp, dstIp, delta[srcIp][dstIp])
						// TODO: This should only record traffic in this iter instead of accumulated traffic
						app.IterTrafficMatrix[app.Iter][srcIp][dstIp] = traffic[srcIp][dstIp]
					}
				}

			}
		}
	}

	return tol
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

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Collect traffic info for each ToR
			torTraffic := make(map[string]int)
			for _, tor := range self.UserView.ToRs {
				sw := tor.Device
				dev, ok := self.Switches[sw]
				if !ok {
					logger.ProcessorLog.Errorf("Switch %s does not exist", sw)
					continue
				}

				swid := dev.Id
				torid := tor.Id
				torName := tor.Name

				traffic := p.getTraffic(swid, torid)
				torTraffic[torName] = traffic

				logger.ProcessorLog.Debugf("Monitor Traffic: Switch %d, ToR %s, traffic %v", swid, torName, traffic)
			}

		case <-ctx.Done():
			logger.ProcessorLog.Infof("Monitor Traffic is stopping")
			return
		}
	}
}
