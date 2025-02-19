package ocss

import (
	"context"
	"time"

	ocss_context "github.com/comp590/ocss/internal/context"
	"github.com/comp590/ocss/internal/logger"
)

func (p *Processor) getTraffic(swId int, torId int) int {
	forwarder := p.Forwarder()

	traffic, err := forwarder.GetTrafficMatrix(swId, torId)
	if err != nil {
		logger.ProcessorLog.Errorf("Error getting traffic matrix: %v", err)
		return 0
	}

	tol := 0
	for _, trafficMap := range traffic {
		for _, traffic := range trafficMap {
			tol += traffic
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

func (p *Processor) Stop() {
	logger.ProcessorLog.Errorf("OCSS Processor is stopping")

	for _, sw := range ocss_context.GetSelf().Switches {
		for _, rules := range sw.ForwardingRule {
			for _, rule := range rules {
				rule.Status = ocss_context.DELETE
			}
		}
	}

	logger.ProcessorLog.Errorf("Delete all forwarding rules")
	p.setupForwardingTable()
}

func (p *Processor) MonitorTraffic(ctx context.Context) {
	self := ocss_context.GetSelf()

	ticker := time.NewTicker(1000 * time.Millisecond)
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

			// Check each subscribed threshold
			self.ThresholdMu.Lock()
			for tk := range self.ThresholdSubscribers {
				traffic, ok := torTraffic[tk.ToR]
				if !ok {
					// No traffic recorded for this ToR or it doesn't exist
					continue
				}

				if traffic > tk.ThresholdValue {
					// Traffic surpassed the threshold defined by ThresholdKey tk
					event := ocss_context.ThresholdEvent{
						ToR:            tk.ToR,
						SurpassedValue: traffic,
						ThresholdValue: tk.ThresholdValue,
					}
					select {
					case self.ThresholdEvents <- event:
						logger.ProcessorLog.Infof("Traffic %v on ToR %s surpassed threshold %v. Event sent.",
							traffic, tk.ToR, tk.ThresholdValue)
					default:
						logger.ProcessorLog.Warnf("Threshold notification channel is full.")
					}
				}
			}
			self.ThresholdMu.Unlock()

		case <-ctx.Done():
			logger.ProcessorLog.Infof("Monitor Traffic is stopping")
			return
		}
	}
}
