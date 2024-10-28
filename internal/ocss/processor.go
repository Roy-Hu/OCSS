package ocss

import (
	ocss_context "github.com/comp590/ocss/internal/context"
	"github.com/comp590/ocss/internal/logger"
	"github.com/comp590/ocss/pkg/app"
)

type ProcessorOCSS interface {
	app.App

	Forwarder() *Forwarder
}

type Processor struct {
	ProcessorOCSS
}

func NewProcessor(ocss ProcessorOCSS) (*Processor, error) {
	o := &Processor{
		ProcessorOCSS: ocss,
	}

	return o, nil
}

func (p *Processor) SetupForwardingTables() error {
	self := ocss_context.GetSelf()
	ocs_finished_setup := make(map[string]bool)

	for _, ocs := range self.UserView.OCSs {
		for ocs_port, connTo := range ocs.Ports {
			logger.ProcessorLog.Infof("OCS [%s] Port [%d] -> [%s] Port [%d]", ocs.Name, ocs_port, connTo.Device, connTo.Port)

			if self.DeviceType[connTo.Device] == ocss_context.SERVER {
				// TODO: Support two or more switch between ocs and server
				device := self.Servers[connTo.Device].Ports[connTo.Port].Device
				in_port := self.Servers[connTo.Device].Ports[connTo.Port].Port

				switch self.DeviceType[device] {
				// TODO: Support optical switch
				case ocss_context.SWITCH:
					for port, dst := range self.Switches[device].Ports {
						logger.ProcessorLog.Infof("Switch [%s] Port [%d] -> Server [%s] Port [%d]", device, port, dst.Device, dst.Port)
						if dst.Device == ocs.Device && dst.Port == ocs_port {
							logger.ProcessorLog.Warnf("OCS [%s] Port [%d] -> Switch [%s] Port [%d]", ocs.Name, ocs_port, device, port)
							out_port := port

							self.ForwardingTables[device] = append(self.ForwardingTables[device], &ocss_context.Forward{
								Device:   device,
								SrcPort:  in_port,
								DestPort: out_port,
							})

							dst.Server[connTo.Device] = true

							for i := range len(ocs.Conn.In_port) {
								if ocs.Conn.In_port[i] == ocs_port {
									out := ocs.Conn.Out_port[i]
									self.OCSs[ocs.Device].Ports[out].Server[connTo.Device] = true

									// TODO: What about ocs is connected to another ocs?
									if self.DeviceType[ocs.Ports[out].Device] == ocss_context.SWITCH {
										self.Switches[ocs.Ports[out].Device].Ports[ocs.Ports[out].Port].Server[connTo.Device] = true
									}
								} else if ocs.Conn.Out_port[i] == ocs_port {
									in := ocs.Conn.In_port[i]
									self.OCSs[ocs.Device].Ports[in].Server[connTo.Device] = true

									if self.DeviceType[ocs.Ports[in].Device] == ocss_context.SWITCH {
										self.Switches[ocs.Ports[in].Device].Ports[ocs.Ports[in].Port].Server[connTo.Device] = true
									}
								}
							}
							continue
						}
					}
				}

				ocs_finished_setup[ocs.Name] = true
			}
		}
	}

	for _, ocs := range self.UserView.OCSs {
		if !ocs_finished_setup[ocs.Name] {
			for i := range len(ocs.Conn.In_port) {
				inPort := ocs.Conn.In_port[i]
				outPort := ocs.Conn.Out_port[i]
				ports := []*ocss_context.ConnectedTo{ocs.Ports[inPort], ocs.Ports[outPort]}
				for _, connTo := range ports {
					port := connTo.Port
					device := connTo.Device
					// TODO: What if OCS connected to another OCS?
					if self.DeviceType[device] == ocss_context.SWITCH {
						tor := self.UserView.FindToRByDeviceAndPort(device, port)
						for dst_port, dst_connTo := range tor.Ports {
							if dst_port == port {
								continue
							}

							if ocs.HavePort(dst_port) {
								logger.ProcessorLog.Debugf("Switch [%s] Port [%d] and Port [%d] both connect to OCS [%s], skip ", connTo.Device, dst_port, port, ocs.Device)
								continue
							}

							for server, ok := range dst_connTo.Server {
								if ok {
									self.Switches[device].Ports[port].Server[server] = true
									connTo.Server[server] = true
									self.ForwardingTables[device] = append(self.ForwardingTables[device], &ocss_context.Forward{
										Device:   device,
										SrcPort:  port,
										DestPort: dst_port,
										Ip:       self.Servers[server].Ip,
									})
									logger.ProcessorLog.Warnf("Switch [%s] Port [%d] -> Server [%s] Port [%d]", device, port, server, dst_port)
								}
							}
						}
					}
				}

				ports = []*ocss_context.ConnectedTo{ocs.Ports[inPort], ocs.Ports[outPort]}
				for i, connTo := range ports {
					port := ports[1-i].Port
					device := ports[1-i].Device
					servers := ports[i].Server
					// TODO: What if OCS connected to another OCS?
					if self.DeviceType[device] == ocss_context.SWITCH {
						tor := self.UserView.FindToRByDeviceAndPort(device, port)
						for dst_port, _ := range tor.Ports {
							if dst_port == port {
								continue
							}

							if ocs.HavePort(dst_port) {
								logger.ProcessorLog.Debugf("Switch [%s] Port [%d] and Port [%d] both connect to OCS [%s], skip ", connTo.Device, dst_port, port, ocs.Device)
								continue
							}

							for server, ok := range servers {
								if ok {
									self.Switches[device].Ports[port].Server[server] = true
									connTo.Server[server] = true
									self.ForwardingTables[device] = append(self.ForwardingTables[device], &ocss_context.Forward{
										Device:   device,
										SrcPort:  dst_port,
										DestPort: port,
										Ip:       self.Servers[server].Ip,
									})
									logger.ProcessorLog.Errorf("Switch [%s] Port [%d] -> Server [%s] Port [%d]", device, dst_port, server, port)
								}
							}
						}
					}
				}
			}
		}
	}

	ocss_context.Print()
	return nil
}

func (p *Processor) SetupController() {
	self := ocss_context.GetSelf()
	forwarder := p.Forwarder()

	for _, sw := range self.Switches {
		hostsPorts := make([]int, 0)
		switchPorts := make([]int, 0)

		for port, connTo := range sw.Ports {
			if self.DeviceType[connTo.Device] == ocss_context.SERVER {
				hostsPorts = append(hostsPorts, port)
			} else {
				switchPorts = append(switchPorts, port)
			}
		}

		logger.CfgLog.Warnf("Switch Info: %d, %v, %v", sw.Id, hostsPorts, switchPorts)
		forwarder.SetSwitchInfo(sw.Id, hostsPorts, switchPorts)
	}

	ocs_in_port := make([]int, 0)
	ocs_out_port := make([]int, 0)

	for _, ocs := range self.OCSs {

		// ip := self.OCSs[ocs.Device].Ip
		ocs_out_port = append(ocs_out_port, ocs.Conn.Out_port...)
		ocs_in_port = append(ocs_in_port, ocs.Conn.In_port...)

		forwarder.SetOCS(ocs.Ip, ocs_in_port, ocs_out_port)
		logger.CfgLog.Warnf("OCS Info: %s, %v, %v", ocs.Ip, ocs_in_port, ocs_out_port)
	}

	for device, ft := range self.ForwardingTables {

		in_port := make([]int, 0)
		out_port := make([]int, 0)

		in_port_ip := make([]int, 0)
		out_port_ip := make([]int, 0)
		ips := make([]string, 0)

		for _, rule := range ft {
			if rule.Ip != "" {
				in_port_ip = append(in_port_ip, rule.SrcPort)
				out_port_ip = append(out_port_ip, rule.DestPort)
				ips = append(ips, rule.Ip)
			} else {
				in_port = append(in_port, rule.SrcPort)
				out_port = append(out_port, rule.DestPort)
			}
		}

		forwarder.SetForwardingTable(self.Switches[device].Id, in_port, out_port)
		forwarder.SetForwardingTableWithIp(self.Switches[device].Id, in_port_ip, out_port_ip, ips)

		logger.ProcessorLog.Warnf("Forwarding Table: %d, %v, %v", self.Switches[device].Id, in_port, out_port)
		logger.ProcessorLog.Warnf("Forwarding Table with IP: %d, %v, %v, %v", self.Switches[device].Id, in_port_ip, out_port_ip, ips)
	}
}
