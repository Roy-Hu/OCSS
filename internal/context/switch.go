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

func (s *Switch) AddForwardingRule(inPort int, outPort int, ip string) {
	rule_name := getRuleName(s.Device, inPort, outPort, ip)

	if _, ok := s.ForwardingRule[rule_name]; ok {
		return
	}
	logger.SwitchLog.Infof("Add New Forwarding Rule: %s", rule_name)
	s.ForwardingRule[rule_name] = &Forward{
		Device:   s.Device,
		SrcPort:  inPort,
		DestPort: outPort,
		Ip:       ip,
		Init:     true,
	}
}

func getRuleName(device string, inPort int, outPort int, ip string) string {
	var rule_name string
	if ip == "" {
		rule_name = fmt.Sprintf("%s_%d_%d", device, inPort, outPort)
	} else {
		rule_name = fmt.Sprintf("%s_%d_%d_%s", device, inPort, outPort, ip)
	}

	return rule_name
}
