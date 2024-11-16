package context

import (
	"fmt"

	"github.com/comp590/ocss/internal/logger"
)

type Switch struct {
	Device         string
	Id             int
	PortConnToMap  map[int]*ConnectedTo
	ForwardingRule map[string]*Forward
	PortServerConn map[int]*ConnServerInfo
}

func (s *Switch) AddForwardingRule(inPort int, outPort int, srcIp string, dstIp string) {
	rule_name := getRuleName(s.Device, inPort, outPort, srcIp, dstIp)

	if _, ok := s.ForwardingRule[rule_name]; ok {
		if s.ForwardingRule[rule_name].DestPort != outPort {
			logger.SwitchLog.Infof("Update Forwarding Rule: %s", rule_name)
			s.ForwardingRule[rule_name].DestPort = outPort
			s.ForwardingRule[rule_name].Status = UPDATE
		}
	} else {
		logger.SwitchLog.Infof("Add New Forwarding Rule: %s", rule_name)
		s.ForwardingRule[rule_name] = &Forward{
			Device:   s.Device,
			SrcPort:  inPort,
			DestPort: outPort,
			SrcIp:    srcIp,
			DstIp:    dstIp,
			Status:   CREATE,
		}
	}

}

func getRuleName(device string, inPort int, outPort int, srcIp string, dstIp string) string {
	var rule_name string
	if dstIp == "" {
		rule_name = fmt.Sprintf("%s_%d_%d", device, inPort, outPort)
	} else {
		rule_name = fmt.Sprintf("%s_%d_%d_%s_%s", device, inPort, outPort, srcIp, dstIp)
	}

	return rule_name
}
