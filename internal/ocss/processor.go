package ocss

import (
	"fmt"
	"os"

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

// ocs_in_port should setup server pointer before calling this function
// setup the sever pointer for the switch port that connects to the ocs port
func setOcsNxtSwitch(self *ocss_context.OCSSContext, ocs *ocss_context.OCS, ocs_in_port int) error {
	ocs_out_port := ocs.ConnectedPort(ocs_in_port)
	if self.UserView.OCSs[ocs.Name].PortServerConn[ocs_out_port].FixedServer {
		logger.ProcessorLog.Errorf("OCS [%s] Port [%d] and Port [%d] both connects to a fixed server, skip", ocs.Name, ocs_in_port, ocs_out_port)
		os.Exit(1)

		return nil
	}

	ocs.PortServerConn[ocs_out_port].CopyConnServerInfo(ocs.PortServerConn[ocs_in_port])

	if self.DeviceType[ocs.PortConnToMap[ocs_out_port].Device] == ocss_context.SWITCH {
		// the switch port that connects to ocs_out_port should also connect to the server
		// but it is not psycially connected to the ocs in port(it may change if the user change the connection in ocs)
		// so we need to set the server pointer
		device := ocs.PortConnToMap[ocs_out_port].Device
		port := ocs.PortConnToMap[ocs_out_port].Port

		tor := self.UserView.FindToRByDeviceAndPort(device, port)
		tor.PortServerConn[port].CopyConnServerInfo(ocs.PortServerConn[ocs_out_port])
	} else {
		logger.ProcessorLog.Errorf("Currently OCS port %d should connect to a switch instead of %s", ocs_out_port, ocs.PortConnToMap[ocs_out_port].Device)
		return fmt.Errorf("Currently OCS should connect to a switch")
	}

	return nil
}

func (p *Processor) CreateForwardingTables() error {
	self := ocss_context.GetSelf()
	ocs_finished_setup := make(map[string]bool)

	// Note: ports connect through physical link should share the same server pointer
	// for logical edge ocs that connects to the server
	for _, ocs := range self.UserView.OCSs {
		connPorts := ocs.GetConnPorts()
		for _, ocs_port := range connPorts {
			connTo := ocs.PortConnToMap[ocs_port]
			// ocs logically connects to the server
			// ocs_port is the port in ocs that connects to the server
			if self.DeviceType[connTo.Device] == ocss_context.SERVER {
				logger.ProcessorLog.Infof("OCS [%s] Port [%d] -> [%s] Port [%d]", ocs.Name, ocs_port, connTo.Device, connTo.Port)

				// TODO: Support two or more switch between ocs and server
				server := connTo.Device
				// what the server actually connects to
				device := self.Servers[server].PortConnToMap[connTo.Port].Device
				in_port := self.Servers[server].PortConnToMap[connTo.Port].Port

				// All others port that connects to this ocs port should also connect to the server
				switch self.DeviceType[device] {
				case ocss_context.OPTICAL_SWITCH:
					// TODO: Support optical switch
					logger.ProcessorLog.Errorf("Currently Server must connect to a switch")
				// server actually connects to a switch instead of the ocs
				// TODO: currently only support one switch between server and ocs
				case ocss_context.SWITCH:

					sw := self.Switches[device]
					for port, sw_connTo := range sw.PortConnToMap {
						logger.ProcessorLog.Debugf("Switch [%s] Port [%d] -> Server [%s] Port [%d]", device, port, sw_connTo.Device, sw_connTo.Port)

						// switch port connects to the current ocs
						if sw_connTo.Device == ocs.Device && sw_connTo.Port == ocs_port {
							logger.ProcessorLog.Infof("OCS [%s] Port [%d] -> Switch [%s] Port [%d]", ocs.Name, ocs_port, device, port)
							out_port := port

							// the in port connects to the server and the out port connects to the ocs beed to be treated as a psycial link

							sw.AddForwardingRule(0, in_port, out_port, "", "")

							// server - > sw in port -> sw out port -> ocs in port share the same sever pointer
							// since we logically connect the server to the ocs
							// the forwarding of in/out port of the sw be treated as psycial link
							ocs.PortServerConn[ocs_port] = &ocss_context.ConnServerInfo{
								Server:      make(map[string]bool),
								FixedServer: true,
							}
							ocs.PortServerConn[ocs_port].Server[sw.PortConnToMap[in_port].Device] = true

							if err := setOcsNxtSwitch(self, ocs, ocs_port); err != nil {
								logger.ProcessorLog.Errorf("Error setting OCS [%s] Port [%d] -> [%s] Port [%d]: ", ocs.Name, ocs_port, connTo.Device, connTo.Port, err)
								return err
							}

							continue
						}
					}
				}

				ocs_finished_setup[ocs.Name] = true
			}
		}
	}

	core_ocs := make(map[string]bool)
	for _, ocs := range self.UserView.OCSs {
		if !ocs_finished_setup[ocs.Name] {
			core_ocs[ocs.Name] = true
		}
	}

	updateCoreOCSAndTor(core_ocs)
	p.createController()

	return nil
}

func (p *Processor) UpdateForwardingTables() error {
	logger.ProcessorLog.Info("Update Forwarding Tables")
	self := ocss_context.GetSelf()

	for _, tor := range self.UserView.ToRs {
		for _, serverInfo := range tor.PortServerConn {
			if !serverInfo.FixedServer {
				serverInfo.Server = make(map[string]bool)
			}
		}
	}

	ocs_finished_setup := make(map[string]bool)

	for _, ocs := range self.UserView.OCSs {
		// if !ocs.Changed {
		// 	ocs_finished_setup[ocs.Name] = true
		// 	logger.ProcessorLog.Infof("OCS [%s] not changed, skip", ocs.Name)
		// 	continue
		// }

		ocs_finished_setup[ocs.Name] = false
		connPorts := ocs.GetConnPorts()
		for _, ocs_port := range connPorts {
			if ocs.PortServerConn[ocs_port].FixedServer {
				if err := setOcsNxtSwitch(self, ocs, ocs_port); err != nil {
					logger.ProcessorLog.Errorf("Error setting OCS [%s] Port [%d] <-> Port [%d]", ocs.Name, ocs_port, ocs.ConnectedPort(ocs_port))
					return err
				}

				// TODO: This may be more dedicated, maybe per port instead of per ocs
				ocs_finished_setup[ocs.Name] = true
			}
		}
	}

	core_ocs := make(map[string]bool)
	for _, ocs := range self.UserView.OCSs {
		if !ocs_finished_setup[ocs.Name] {
			core_ocs[ocs.Name] = true
		}
	}

	updateCoreOCSAndTor(core_ocs)

	p.updateController()

	return nil
}

func updateCoreOCSAndTor(core_ocs map[string]bool) {
	logger.ProcessorLog.Info("Update Core OCS and ToR")
	self := ocss_context.GetSelf()
	// for logical core ocs that connects to the switch
	for _, ocs := range self.UserView.OCSs {
		// ocs without logical connection to the server wull be the core
		setup := make(map[int]bool)
		if core_ocs[ocs.Name] {
			logger.ProcessorLog.Infof("OCS [%s] Update Forwarding Table", ocs.Name)

			for i := range len(ocs.Conn.In_port) {
				inPort := ocs.Conn.In_port[i]
				outPort := ocs.Conn.Out_port[i]
				if setup[inPort] || setup[outPort] {
					continue
				} else {
					setup[inPort] = true
					setup[outPort] = true
				}

				logger.ProcessorLog.Infof("OCS [%s] Connection Port [%d](Connect to %s %d) <-> Port [%d](Connect to %s %d)",
					ocs.Name, inPort, ocs.PortConnToMap[inPort].Name, ocs.PortConnToMap[inPort].Port, outPort, ocs.PortConnToMap[outPort].Name, ocs.PortConnToMap[outPort].Port)

				ports := []int{inPort, outPort}
				tors := []*ocss_context.ToR{}
				tor_servers := [][]string{}
				conn_to_ocs_port := []int{}
				for _, port := range ports {
					connTo := ocs.PortConnToMap[port]
					device := connTo.Device
					// TODO: What if OCS connected to another OCS?
					// ocs in/out port connects to a switch
					if self.DeviceType[device] == ocss_context.SWITCH {
						sw_port := connTo.Port
						conn_to_ocs_port = append(conn_to_ocs_port, sw_port)
						// Find the tor that connects to the ocs in/out port
						tor := self.UserView.FindToRByDeviceAndPort(device, sw_port)
						if tor == nil {
							logger.ProcessorLog.Errorf("Switch [%s] Port [%d] is not a ToR", device, sw_port)
							continue
						}

						tors = append(tors, tor)
						tor_servers = append(tor_servers, tor.ConnectedServers())
					}
				}

				for i := range 2 {
					src_tor := tors[i]
					dst_tor_servers := tor_servers[1-i]
					tor_ocs_port := conn_to_ocs_port[i]

					for tor_server_port, tor_connTo := range src_tor.PortConnToMap {
						if tor_server_port == tor_ocs_port {
							continue
						} else if ocs.HavePort(tor_connTo.Port) {
							logger.ProcessorLog.Debugf("[%s] Port [%d] and Port [%d] both connect to OCS [%s], skip ", src_tor.Name, tor_server_port, tor_ocs_port, ocs.Name)
							continue
						} else {
							src_sw := self.Switches[src_tor.Device]
							src_servers := src_tor.PortServerConn[tor_server_port].Server
							for src_server, ok := range src_servers {
								if ok {
									src_server_ip := self.Servers[src_server].Ip
									for _, dst_server := range dst_tor_servers {
										dst_server_ip := self.Servers[dst_server].Ip
										// src server -> src tor -> ocs -> dst tor -> dst server
										src_sw.AddForwardingRule(src_tor.Id, tor_server_port, tor_ocs_port, src_server_ip, dst_server_ip)
										logger.ProcessorLog.Infof("Server[%s] -> %s[Port [%d] -> [%d]] -> Server [%s] ", self.Servers[src_server].Ip, src_tor.Name, tor_server_port, tor_ocs_port, dst_server_ip)

										// dst server -> dst tor -> ocs -> src tor -> src server
										src_sw.AddForwardingRule(src_tor.Id, tor_ocs_port, tor_server_port, dst_server_ip, src_server_ip)
										logger.ProcessorLog.Infof("Server [%s] -> %s[Port [%d] -> [%d]] -> Server [%s] ", self.Servers[dst_server].Ip, src_tor.Name, tor_ocs_port, tor_server_port, src_server_ip)
									}
								}
							}
						}
					}
				}
			}
		}

		ocs.Changed = false
	}
}

func (p *Processor) setupSwitch() {
	self := ocss_context.GetSelf()
	forwarder := p.Forwarder()

	for _, sw := range self.Switches {
		hostsPorts := make([]int, 0)
		switchPorts := make([]int, 0)

		for port, connTo := range sw.PortConnToMap {
			if self.DeviceType[connTo.Device] == ocss_context.SERVER {
				hostsPorts = append(hostsPorts, port)
			} else {
				switchPorts = append(switchPorts, port)
			}
		}

		logger.CfgLog.Infof("Switch Info: %d, %v, %v", sw.Id, hostsPorts, switchPorts)
		forwarder.SetSwitchInfo(sw.Id, hostsPorts, switchPorts)
	}
}

func (p *Processor) setupOCS() {
	self := ocss_context.GetSelf()
	forwarder := p.Forwarder()

	ocs_in_port := make([]int, 0)
	ocs_out_port := make([]int, 0)

	for _, ocs := range self.UserView.OCSs {
		// ip := self.OCSs[ocs.Device].Ip
		ocs_out_port = append(ocs_out_port, ocs.Conn.Out_port...)
		ocs_in_port = append(ocs_in_port, ocs.Conn.In_port...)
	}

	forwarder.SetOCS("10.224.92.88", ocs_in_port, ocs_out_port)
	logger.CfgLog.Infof("OCS Info: %s, %v, %v", "10.224.92.88", ocs_in_port, ocs_out_port)
}

func (p *Processor) setupForwardingTable() {
	self := ocss_context.GetSelf()
	forwarder := p.Forwarder()

	for _, sw := range self.Switches {

		for torId, torRules := range sw.ForwardingRule {
			for _, rule := range torRules {
				in_port := make([]int, 0)
				out_port := make([]int, 0)

				delete_in_port := make([]int, 0)
				delete_out_port := make([]int, 0)

				create_in_port_ip := make([]int, 0)
				create_out_port_ip := make([]int, 0)
				create_src_ips := make([]string, 0)
				create_dst_ips := make([]string, 0)

				update_in_port_ip := make([]int, 0)
				update_out_port_ip := make([]int, 0)
				update_src_ips := make([]string, 0)
				update_dst_ips := make([]string, 0)

				delete_in_port_ip := make([]int, 0)
				delete_out_port_ip := make([]int, 0)
				delete_src_ips := make([]string, 0)
				delete_dst_ips := make([]string, 0)

				if rule.Status == ocss_context.ACTIVE {
					continue
				} else if rule.Status == ocss_context.CREATE {
					if rule.DstIp != "" {
						create_in_port_ip = append(create_in_port_ip, rule.SrcPort)
						create_out_port_ip = append(create_out_port_ip, rule.DestPort)
						create_src_ips = append(create_src_ips, rule.SrcIp)
						create_dst_ips = append(create_dst_ips, rule.DstIp)
					} else {
						in_port = append(in_port, rule.SrcPort)
						out_port = append(out_port, rule.DestPort)
					}

					rule.Status = ocss_context.ACTIVE

				} else if rule.Status == ocss_context.UPDATE {
					if rule.DstIp != "" {
						update_in_port_ip = append(update_in_port_ip, rule.SrcPort)
						update_out_port_ip = append(update_out_port_ip, rule.DestPort)
						update_src_ips = append(update_src_ips, rule.SrcIp)
						update_dst_ips = append(update_dst_ips, rule.DstIp)
					}

					rule.Status = ocss_context.ACTIVE

				} else if rule.Status == ocss_context.DELETE {
					if rule.DstIp != "" {
						delete_in_port_ip = append(delete_in_port_ip, rule.SrcPort)
						delete_out_port_ip = append(delete_out_port_ip, rule.DestPort)
						delete_src_ips = append(delete_src_ips, rule.SrcIp)
						delete_dst_ips = append(delete_dst_ips, rule.DstIp)
					} else {
						delete_in_port = append(delete_in_port, rule.SrcPort)
						delete_out_port = append(delete_out_port, rule.DestPort)
					}
				}

				if len(in_port) != 0 && len(out_port) != 0 {
					logger.ProcessorLog.Infof("Forwarding Table: %d, %v, %v", sw.Id, in_port, out_port)
					empty_ips := make([]string, len(in_port))

					forwarder.CreateForwardingTable(sw.Id, torId, in_port, out_port, empty_ips, empty_ips)
				}

				if len(create_in_port_ip) != 0 && len(create_out_port_ip) != 0 {
					logger.ProcessorLog.Infof("Update Forwarding Table with IP: %d, In Port %d, Out Port %d, Src %v, Dst %v", sw.Id, create_in_port_ip, create_out_port_ip, create_src_ips, create_dst_ips)

					forwarder.CreateForwardingTable(sw.Id, torId, create_in_port_ip, create_out_port_ip, create_src_ips, create_dst_ips)
				}

				if len(update_in_port_ip) != 0 && len(update_out_port_ip) != 0 {
					logger.ProcessorLog.Infof("Update Forwarding Table with IP: %d, In Port %d, Out Port %d, Src %v, Dst %v", sw.Id, update_in_port_ip, update_out_port_ip, update_src_ips, update_dst_ips)

					forwarder.UpdateForwardingTable(sw.Id, torId, update_in_port_ip, update_out_port_ip, update_src_ips, update_dst_ips)
				}

				if len(delete_in_port_ip) != 0 && len(delete_out_port_ip) != 0 {
					logger.ProcessorLog.Infof("Delete Forwarding Table with IP: %d, In Port %d, Out Port %d, Src %v, Dst %v", sw.Id, delete_in_port_ip, delete_out_port_ip, delete_src_ips, delete_dst_ips)

					forwarder.DeleteForwardingTable(sw.Id, torId, delete_in_port_ip, delete_out_port_ip, delete_src_ips, delete_dst_ips)
				}

				if len(delete_in_port) != 0 && len(delete_out_port) != 0 {
					logger.ProcessorLog.Infof("Delete Forwarding Table: %d, %v, %v", sw.Id, delete_in_port, delete_out_port)

					empty_ips := make([]string, len(delete_in_port))

					forwarder.DeleteForwardingTable(sw.Id, torId, delete_in_port, delete_out_port, empty_ips, empty_ips)
				}
			}
		}

	}
}

func (p *Processor) GetTraffic(swId int, torId int) ocss_context.TrafficMatrix {
	forwarder := p.Forwarder()

	traffic, err := forwarder.GetTrafficMatrix(swId, torId)
	if err != nil {
		logger.ProcessorLog.Errorf("Error getting traffic matrix: %v", err)
		return nil
	}

	return traffic
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
	logger.ProcessorLog.Info("OCSS Processor is stopping")

	for _, sw := range ocss_context.GetSelf().Switches {
		for _, rules := range sw.ForwardingRule {
			for _, rule := range rules {
				rule.Status = ocss_context.DELETE
			}
		}
	}

	p.setupForwardingTable()
}
