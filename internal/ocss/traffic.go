package ocss

import (
	ocss_context "github.com/comp590/ocss/internal/context"
	"github.com/comp590/ocss/internal/logger"
)

func (p *Processor) GetFlowTraffic(ips []string) ocss_context.TrafficMatrix {
	self := ocss_context.GetSelf()
	flowTraffic := make(ocss_context.TrafficMatrix)
	traffics := make([]ocss_context.TrafficMatrix, 0)

	for _, srcIp := range ips {
		flowTraffic[srcIp] = make(map[string]int)
		for _, dstIp := range ips {
			flowTraffic[srcIp][dstIp] = 0
		}
	}

	// Collect traffic info for each ToR
	for _, tor := range self.UserView.ToRs {
		traffic := p.getTraffic(tor)
		if traffic != nil {
			traffics = append(traffics, traffic)
		}
	}

	for _, srcIp := range ips {
		for _, dstIp := range ips {
			for _, traffic := range traffics {
				if _, ok := traffic[srcIp][dstIp]; ok {
					if flowTraffic[srcIp][dstIp] == 0 {
						flowTraffic[srcIp][dstIp] = traffic[srcIp][dstIp]
					} else {
						flowTraffic[srcIp][dstIp] = min(flowTraffic[srcIp][dstIp], traffic[srcIp][dstIp])
					}
				}
			}
		}
	}

	return flowTraffic
}

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
