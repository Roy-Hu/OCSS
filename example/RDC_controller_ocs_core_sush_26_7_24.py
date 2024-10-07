import copy
import logging
import socket, threading,time
from collections import namedtuple,defaultdict
from ryu.base import app_manager
from ryu.controller import dpset
from ryu.controller import ofp_event
from ryu.controller.handler import MAIN_DISPATCHER
from ryu.controller.handler import set_ev_cls
from ryu.ofproto import ofproto_v1_3
from ryu.lib.packet import packet
from ofdpa.config_parser import ConfigParser
from ofdpa.mods import Mods

# For OCS
from OCSRelated import ocsConfigs
from OCSRelated.connections import GxcConnections
from OCSRelated.optionsForShareBackup import Options
from OCSRelated import SwitchLinkMapping

# ryu_loggers = logging.Logger.manager.loggerDict
#
#
# def ryu_loggers_on(on):
#     for key in ryu_loggers.keys():
#         ryu_logger = logging.getLogger(key)
#         ryu_logger.propagate = on

import RDC_controller_util

logging.basicConfig(level=logging.ERROR)
LOG = logging.getLogger("RDC_8Servers")
# create formatter and add it to the handlers
ch = logging.StreamHandler()
formatter = logging.Formatter('%(asctime)s - %(message)s')
ch.setFormatter(formatter)
ch.setLevel(logging.INFO)
LOG.addHandler(ch)

class RDC(app_manager.RyuApp):
	OFP_VERSIONS = [ofproto_v1_3.OFP_VERSION]
	CONFIG_FILE = 'config/ofdpa_te.json'

	def __init__(self, *args, **kwargs):

		super(RDC, self).__init__(*args, **kwargs)

		self.dataPaths = {}

		config_dir = 'config/te_test/rdc'
		template_group_l2_interface_filename = "%s/%s.json" % (config_dir, "template_group_l2_interface")
		self.groupConfig = ConfigParser.get_config(template_group_l2_interface_filename)

		template_acl_unicast_vlan_inPort = "%s/%s.json" % (config_dir, "te_acl_unicast_vlan_inPort")
		self.configVlanInPort = ConfigParser.get_config(template_acl_unicast_vlan_inPort)

		template_acl_unicast_vlan_desIP = "%s/%s.json" % (config_dir, "te_acl_unicast_vlan_desIP")
		self.configVlanDesIP = ConfigParser.get_config(template_acl_unicast_vlan_desIP)

		template_acl_unicast_vlan_inPort_desIP = "%s/%s.json" % (config_dir, "te_acl_unicast_vlan_inPort_desIP")
		self.configVlanInPortDesIP = ConfigParser.get_config(template_acl_unicast_vlan_inPort_desIP)

		template_vlan_tagged = "%s/%s.json" % (config_dir, "template_vlan_tagged")
		self.configVlanTagged = ConfigParser.get_config(template_vlan_tagged)

		template_vlan_untagged = "%s/%s.json" % (config_dir, "template_vlan_untagged")
		self.configVlanUnTagged = ConfigParser.get_config(template_vlan_untagged)

		l2InterfacePopVlan = "%s/%s.json" % (config_dir, "group_l2_interface_popVlan")
		self.groupConfigPopVlan = ConfigParser.get_config(l2InterfacePopVlan)

		template_to_controller = "%s/%s.json" % (config_dir, "te_acl_vlan_inPort_desIP_Controller")
		self.configToController = ConfigParser.get_config(template_to_controller)

		template_acl_unicast_inPort_srcIP_desIP = "%s/%s.json" % (config_dir, "te_acl_unicast_inPort_srcIP_desIP")
		self.configVlanInPortSrcIPDesIP = ConfigParser.get_config(template_acl_unicast_inPort_srcIP_desIP)

		template_acl_unicast_srcIP_desIP = "%s/%s.json" % (config_dir, "template_acl_unicast_src_dst")
		self.configVlanSrcIPDesIP = ConfigParser.get_config(template_acl_unicast_srcIP_desIP)

		template_acl_arp_multicast_filename = "%s/%s.json" % (config_dir, "template_acl_arp_multicast")
		self.config_acl_arp_multicast = ConfigParser.get_config(template_acl_arp_multicast_filename)


		self.OCS_create_initial_connections()

	@staticmethod
	def install_group_mod(dp, config):
		for type_ in ConfigParser.get_config_type(config):
			if type_ == "group_mod":
				mod_config = ConfigParser.get_group_mod(config)
				mod = Mods.create_group_mod(dp, mod_config)
				dp.send_msg(mod)
				return mod
			else:
				raise Exception("Wrong type", type_)
		return None


	@staticmethod
	def install_flow_mod(dp, config):
		for type_ in ConfigParser.get_config_type(config):
			if type_ == "flow_mod":
				mod_config = ConfigParser.get_flow_mod(config)
				mod = Mods.create_flow_mod(dp, mod_config)
				dp.send_msg(mod)
				return mod
			else:
				raise Exception("Wrong type", type_)
		return None

	def create_group_l2_interface(self, template, dp, vlan, outputPort):
		group_l2_interface = copy.deepcopy(template)
		group_l2_interface['group_mod']['_name'] += "%03x%04x" % (vlan, outputPort)
		group_l2_interface['group_mod']['group_id'] += "%03x%04x" % (vlan, outputPort)
		group_l2_interface['group_mod']['buckets'][0]['actions'][0]['output']['port'] += str(outputPort)
		self.install_group_mod(dp, group_l2_interface)
		return


	def create_vlan(self, dp, vlan, inPort):
		vlan_tagged = copy.deepcopy(self.configVlanTagged)
		vlan_tagged['flow_mod']['_name'] += str(vlan) + "_" + str(inPort)
		vlan_tagged['flow_mod']['match']['in_port'] += str(inPort)
		vlan_tagged['flow_mod']['match']['vlan_vid'] += str(vlan)
		self.install_flow_mod(dp, vlan_tagged)

		vlan_untagged = copy.deepcopy(self.configVlanUnTagged)
		vlan_untagged['flow_mod']['_name'] += str(vlan) + "_" + str(inPort)
		vlan_untagged['flow_mod']['match']['in_port'] += str(inPort)
		vlan_untagged['flow_mod']['instructions'][0]['apply'][0]['actions'][0]['set_field']['vlan_vid'] += str(vlan)
		self.install_flow_mod(dp, vlan_untagged)
		return

	def create_acl_unicast_vlan_inPort(self, dp, vlan, inPort, outputPort, priority=3):
		acl_unicast = copy.deepcopy(self.configVlanInPort)
		acl_unicast['flow_mod']['_name'] += str(vlan) + '_' + str(inPort) + '_' + str(outputPort)
		acl_unicast['flow_mod']['priority'] += str(priority)
		acl_unicast['flow_mod']['cmd'] = 'add'
		acl_unicast['flow_mod']['match']['vlan_vid'] += str(vlan)
		acl_unicast['flow_mod']['match']["in_port"] += str(inPort)
		acl_unicast['flow_mod']['instructions'][0]['write'][0]['actions'][0]['set_queue']['queue_id'] += str(1)
		acl_unicast['flow_mod']['instructions'][0]['write'][0]['actions'][1]['group']['group_id'] += "%03x%04x" % (
			vlan, outputPort)
		# print("dpid: %d vlan: %d inPort: %d outPort: %d" % (dp.id, vlan, inPort, outputPort))
                print (acl_unicast)
		self.install_flow_mod(dp, acl_unicast)
		return

	def create_acl_unicast_vlan_inPort_dstIp(self, dp, vlan, inPort, ip, outputPort, priority=3):
		acl_unicast = copy.deepcopy(self.configVlanInPortDesIP)
		queue = 1
		acl_unicast['flow_mod']['_name'] += str(vlan) + '_' + str(inPort) + '_' + ip + '_' + str(outputPort)
		acl_unicast['flow_mod']['priority'] += str(priority)
		acl_unicast['flow_mod']['cmd'] = 'add'
		acl_unicast['flow_mod']['match']['vlan_vid'] += str(vlan)
		acl_unicast['flow_mod']['match']["in_port"] += str(inPort)
		acl_unicast['flow_mod']['match']['ipv4_dst'] += ip
		acl_unicast['flow_mod']['instructions'][0]['write'][0]['actions'][0]['set_queue']['queue_id'] += str(queue)
		acl_unicast['flow_mod']['instructions'][0]['write'][0]['actions'][1]['group']['group_id'] += "%03x%04x" % (
			vlan, outputPort)
                print (acl_unicast)
		self.install_flow_mod(dp, acl_unicast)
		return acl_unicast

	def create_acl_unicast_srcIP_dstIp(self,dp, vlan, srcIP, dstIP, outputPort, priority=3):
		acl_unicast = copy.deepcopy(self.configVlanSrcIPDesIP)
		queue = 1
		acl_unicast['flow_mod']['_name'] += str(vlan) + '_' + srcIP + '_' + dstIP + '_' + str(outputPort)
		acl_unicast['flow_mod']['priority'] += str(priority)
		acl_unicast['flow_mod']['cmd'] = 'add'
		acl_unicast['flow_mod']['match']['vlan_vid'] += str(vlan)
		acl_unicast['flow_mod']['match']["ipv4_src"] += str(srcIP)
		acl_unicast['flow_mod']['match']['ipv4_dst'] += str(dstIP)
		acl_unicast['flow_mod']['instructions'][0]['write'][0]['actions'][0]['set_queue']['queue_id'] += str(queue)
		acl_unicast['flow_mod']['instructions'][0]['write'][0]['actions'][1]['group']['group_id'] += "%03x%04x" % (
			vlan, outputPort)
		self.install_flow_mod(dp, acl_unicast)
		return acl_unicast

	def create_acl_unicast_vlan_dstIp(self, dp, vlan, ip, outputPort, priority=3):
		acl_unicast = copy.deepcopy(self.configVlanDesIP)
		acl_unicast['flow_mod']['_name'] += str(vlan)+'_'+str(ip)+'_'+str(outputPort)
		acl_unicast['flow_mod']['priority'] += str(priority)
		acl_unicast['flow_mod']['cmd'] = 'add'
		acl_unicast['flow_mod']['match']['vlan_vid'] += str(vlan)
		acl_unicast['flow_mod']['match']['ipv4_dst'] += ip
		acl_unicast['flow_mod']['instructions'][0]['write'][0]['actions'][0]['set_queue']['queue_id'] += str(1)
		acl_unicast['flow_mod']['instructions'][0]['write'][0]['actions'][1]['group']['group_id'] += "%03x%04x" % (
			vlan, outputPort)
		#print("dpid: %d vlan: %d desIp: %s, outPort: %d" % (dp.id, vlan, ip, outputPort))
                print (acl_unicast)
		self.install_flow_mod(dp, acl_unicast)
		return acl_unicast

	def create_acl_arp_flood(self, dp,  vlan, id = 3276):
		acl_arp = copy.deepcopy(self.config_acl_arp_multicast)
		acl_arp['flow_mod']['_name'] += str(vlan)
		acl_arp['flow_mod']['match']['vlan_vid'] += str(vlan)
		acl_arp['flow_mod']['instructions'][0]['write'][0]['actions'][0]['set_queue']['queue_id'] += str(1)
		acl_arp['flow_mod']['instructions'][0]['write'][0]['actions'][1]['group']['group_id'] += "%03x%04x" % (vlan, id)
		self.install_flow_mod(dp, acl_arp)
		return


	def tagVlan(self, dp, vlan=10):
		if(dp.id == 1 or dp.id ==5):
			for inPort in range(1, 42):
				self.create_vlan(dp, vlan, inPort)

	def createGroupInterfaces(self, dp, vlan=10):

		#if(dp.id == 0):
		#	hostPorts=[]
		#	switchPorts=[1, 8, 33, 40]

		if(dp.id == 1 or dp.id == 5):
			hostPorts = range(1, 9)
			switchPorts = range(17, 25) + range(25,42) # + range(33,42)
		else:
			return

		for port in hostPorts:
			self.create_group_l2_interface(self.groupConfigPopVlan, dp, vlan, port)
		for port in switchPorts:
			self.create_group_l2_interface(self.groupConfig, dp, vlan, port)


	def LotusSwitch(self, dp, vlan = 10):
		inPorts = [1, 2, 3, 4, 5, 6, 7, 8]
		toPorts = [17, 18, 19, 20, 21, 22, 23, 24]
		for port in inPorts:
			self.create_acl_unicast_vlan_inPort(dp, vlan, port, port + 16, priority = 52)
			self.create_acl_unicast_vlan_inPort(dp, vlan, port + 16, port, priority = 52)


	def EdgeSwitch(self, dp, vlan=10):
		ipPrefix = "192.168.50."

		pivot = 146 if dp.id == 5 else 114
		groupList = [0, 1] if dp.id == 5 else [2, 3]

		def is_same_group(port1, port2):
			l = min(port1, port2)
			r = max(port1, port2)
			if l <= 32 and r > 32:
				return False
			elif l <= 32 and r <=32:
				return r-l <=3 and (r <=28 or l>28)
			elif l >=32 and r >= 32:
				return r-l <=3 and (r <=36 or l>36)

		hosts_5 = range(143, 151)
		hosts_1 = range(111, 119)
		allHostPool = hosts_5 + hosts_1

		tor_ports = { 0:range(25, 29), 1:range(29,33), 2:range(25,29), 3:range(29,33)}
		tor_port_upLink = {25:9, 26:9, 27:9, 28:9, 29:16, 30:16, 31:16, 32:16}
		up_ports = [9, 16]


		default_host_to_port = defaultdict()
		default_port_to_host = defaultdict()

		for host in hosts_5:
			default_host_to_port[host] = host - (143-25)
		for host in hosts_1:
			default_host_to_port[host] = host - (111-25)


		if(dp.id == 5):
			for host in hosts_5:
				default_port_to_host[host-143+25] = host
		else:
			for host in hosts_1:
				default_port_to_host[host-111+25] = host

		for tor in groupList:
			srcPorts = tor_ports[tor]
			# data from server
			for srcPort in srcPorts:
				# src = default_port_to_host[srcPort]
				for dst in allHostPool:
					if dst in hosts_5:
						switch = 5
					elif dst in hosts_1:
						switch = 1
					else:
						print("DST host name Error!")
					if switch - dp.id == 0:
						dstPort = default_host_to_port[dst]
						if(srcPort != dstPort):
							# src_ip = ipPrefix + str(src)
							dst_ip = ipPrefix + str(dst)
							if( is_same_group(srcPort, dstPort)):
								self.create_acl_unicast_vlan_inPort_dstIp(dp, vlan, srcPort, dst_ip, dstPort)
								print("{}    {}->{}:{}".format(dp.id, srcPort, dst, dstPort))
							else:
								self.create_acl_unicast_vlan_inPort_dstIp(dp, vlan, srcPort, dst_ip, tor_port_upLink[srcPort])
								print("{}    {}->{}:{}".format(dp.id, srcPort, dst, tor_port_upLink[srcPort]))
						else:
							dst_ip = ipPrefix + str(dst)
							self.create_acl_unicast_vlan_inPort_dstIp(dp, vlan, srcPort, dst_ip, 33)
							print("{}    {}->{}:{}".format(dp.id, srcPort, dst, 33))
					else:
						dstPort = tor_port_upLink[srcPort]
						dst_ip = ipPrefix + str(dst)
						self.create_acl_unicast_vlan_inPort_dstIp(dp, vlan, srcPort, dst_ip, dstPort)
						print("{}    {}->{}:{}".format(dp.id, srcPort, dst, dstPort))
			# data from core to tor switch
			pod_ports = range(25,33)
			srcPort = up_ports[tor%2]
			for dstPort in pod_ports: 
				if dstPort in tor_ports[tor]: # legal port
					dst = default_port_to_host[dstPort]
					if (srcPort != dstPort):
						dst_ip = ipPrefix + str(dst)
						print ("%d    %d -> %d = %d" % (dp.id, srcPort, dst, dstPort) )
						self.create_acl_unicast_vlan_inPort_dstIp(dp, vlan, srcPort, dst_ip, dstPort)
				else: # illegal port
					dst = default_port_to_host[dstPort]
					dst_ip = ipPrefix + str(dst)
					dstPort = 33
					print ("%d    %d -> %d = %d" % (dp.id, srcPort, dst, dstPort) )
					self.create_acl_unicast_vlan_inPort_dstIp(dp, vlan, srcPort, dst_ip, dstPort)
			# add rules for servers in other switches
			if (dp.id == 5):
				for dst in range(111,119):
					dst_ip = ipPrefix + str(dst)
					dstPort = 33
					print ("%d    %d -> %d = %d" % (dp.id, srcPort, dst, dstPort) )
					self.create_acl_unicast_vlan_inPort_dstIp(dp, vlan, srcPort, dst_ip, dstPort)
			else:
				for dst in range(143,151):
					dst_ip = ipPrefix + str(dst)
					dstPort = 33
					print ("%d    %d -> %d = %d" % (dp.id, srcPort, dst, dstPort) )
					self.create_acl_unicast_vlan_inPort_dstIp(dp, vlan, srcPort, dst_ip, dstPort)

	def CoreSwitch(self,dp,vlan=10):
		ipPrefix = "192.168.50."
		if(dp.id != 0):
			return
                
		#for ipPost in [143, 144, 145, 146]:
		#	rip = ipPrefix + str(ipPost)
		#	self.create_acl_unicast_vlan_dstIp(dp,vlan,rip,33)

		#for ipPost in [147, 148, 149, 150]:
		#	rip = ipPrefix + str(ipPost)
		#	self.create_acl_unicast_vlan_dstIp(dp,vlan,rip,40)

		for ipPost in [111, 112]:
			rip = ipPrefix + str(ipPost)
			self.create_acl_unicast_vlan_dstIp(dp,vlan,rip,1)

		for ipPost in [115, 116]:
			rip = ipPrefix + str(ipPost)
			self.create_acl_unicast_vlan_dstIp(dp,vlan,rip,8)


	@set_ev_cls(dpset.EventDP, dpset.DPSET_EV_DISPATCHER)
	def handler_datapath(self, ev):
		LOG.info("Datapath Id: %i - 0x%x", ev.dp.id, ev.dp.id)
		LOG.info("Datapath Address: %s", ev.dp.address[0])
		self.dataPaths[ev.dp.id] = ev.dp
		if ev.enter:
			self.build_packets(ev.dp, ev.dp.id)

	@set_ev_cls(ofp_event.EventOFPPacketIn, MAIN_DISPATCHER)
	def packet_in_handler(self, ev):
		LOG.info("=====PacketIn Event=====")


	def build_packets(self, dp, dpid):
		if(dpid == 1 or dpid == 5):
			defaultVlan = 10
			self.createGroupInterfaces(dp, defaultVlan)
			##self.create_acl_arp_flood(dp,10)
                        
			if(dpid == 1):
				self.tagVlan(dp, defaultVlan)
                                

                                #self.create_acl_unicast_vlan_inPort(dp, vlan, inPort, outputPort, priority=3)                                
                                
                                self.create_acl_unicast_vlan_inPort(dp, 10, 1, 17)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 2, 18)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 3, 19)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 4, 20)
                                
                                self.create_acl_unicast_vlan_inPort(dp, 10, 5, 21)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 6, 22)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 7, 23)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 8, 24)
 
                                self.create_acl_unicast_vlan_inPort(dp, 10, 17, 1)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 18, 2)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 19, 3)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 20, 4)
                                
                                self.create_acl_unicast_vlan_inPort(dp, 10, 21, 5)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 22, 6)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 23, 7)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 24, 8)
 

                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, vlan, srcPort, dst_ip, dstPort)
 
                                # for self server to server I have used dummy port 34, practically traffic will not come 
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.111", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.112", 26)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.113", 27)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.114", 28)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.115", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.116", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.117", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.118", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.143", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.144", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.145", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.146", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.147", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.148", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.149", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.150", 33)
                                 
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.111", 25)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.112", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.113", 27)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.114", 28)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.115", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.116", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.117", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.118", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.143", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.144", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.145", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.146", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.147", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.148", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.149", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.150", 33)
                                
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.111", 25)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.112", 26)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.113", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.114", 28)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.115", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.116", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.117", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.118", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.143", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.144", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.145", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.146", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.147", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.148", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.149", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.150", 33)
                                
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.111", 25)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.112", 26)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.113", 27)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.114", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.115", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.116", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.117", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.118", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.143", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.144", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.145", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.146", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.147", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.148", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.149", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.150", 33)
                                
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.111", 38)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.112", 38)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.113", 38)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.114", 38)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.115", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.116", 30)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.117", 31)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.118", 32)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.143", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.144", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.145", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.146", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.147", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.148", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.149", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.150", 40)
                                
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.111", 38)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.112", 38)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.113", 38)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.114", 38)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.115", 29)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.116", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.117", 31)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.118", 32)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.143", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.144", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.145", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.146", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.147", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.148", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.149", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.150", 40)
                                
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.111", 38)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.112", 38)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.113", 38)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.114", 38)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.115", 29)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.116", 30)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.117", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.118", 32)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.143", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.144", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.145", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.146", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.147", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.148", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.149", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.150", 40)
                                
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.111", 38)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.112", 38)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.113", 38)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.114", 38)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.115", 29)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.116", 30)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.117", 31)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.118", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.143", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.144", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.145", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.146", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.147", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.148", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.149", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.150", 40)

                                ######################
                                # now these are rules when traffic incoming to these uplink ports from other ToRs
                                # adding all servers rule for ToR uplink ports, with dummy group id
                                # so that we can use those to change appropriately as and when necessary
                                # normal time those rules will not impact
                                # note actual dpid 1 has two logical ToRs (T0 and T1)
                                # from ToR T0->T1: 35, T0->T2: 36, T0->T3: 33
                                # from ToR T1->T0: 38, T1->T2: 39, T1->T3: 40
                                # dummy port 34 for dpid 1, use the group corresponding to that for all "not"effective rules 

                                
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.111", 25)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.112", 26)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.113", 27)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.114", 28)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.115", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.116", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.117", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.118", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.143", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.144", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.145", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.146", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.147", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.148", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.149", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.150", 34)

                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.111", 25)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.112", 26)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.113", 27)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.114", 28)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.115", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.116", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.117", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.118", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.143", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.144", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.145", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.146", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.147", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.148", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.149", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.150", 34)

                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.111", 25)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.112", 26)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.113", 27)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.114", 28)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.115", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.116", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.117", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.118", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.143", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.144", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.145", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.146", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.147", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.148", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.149", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.150", 34)

                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 38, "192.168.50.111", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 38, "192.168.50.112", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 38, "192.168.50.113", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 38, "192.168.50.114", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 38, "192.168.50.115", 29)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 38, "192.168.50.116", 30)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 38, "192.168.50.117", 31)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 38, "192.168.50.118", 32)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 38, "192.168.50.143", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 38, "192.168.50.144", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 38, "192.168.50.145", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 38, "192.168.50.146", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 38, "192.168.50.147", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 38, "192.168.50.148", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 38, "192.168.50.149", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 38, "192.168.50.150", 34)

                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.111", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.112", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.113", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.114", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.115", 29)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.116", 30)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.117", 31)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.118", 32)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.143", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.144", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.145", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.146", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.147", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.148", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.149", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.150", 34)

                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.111", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.112", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.113", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.114", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.115", 29)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.116", 30)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.117", 31)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.118", 32)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.143", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.144", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.145", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.146", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.147", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.148", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.149", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.150", 34)



                                ########################

                                # this was actually effective, we keep them as it is already

                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.111", 25)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.112", 26)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.113", 27)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.114", 28)

                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.111", 25)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.112", 26)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.113", 27)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.114", 28)
                                
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.111", 25)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.112", 26)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.113", 27)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.114", 28)

                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 38, "192.168.50.115", 29)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 38, "192.168.50.116", 30)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 38, "192.168.50.117", 31)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 38, "192.168.50.118", 32)

                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.115", 29)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.116", 30)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.117", 31)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.118", 32)

                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.115", 29)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.116", 30)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.117", 31)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.118", 32)
                                
                        elif(dpid == 5):
				self.tagVlan(dp, defaultVlan)
                                

                                #self.create_acl_unicast_vlan_inPort(dp, vlan, inPort, outputPort, priority=3)                                
                                
                                self.create_acl_unicast_vlan_inPort(dp, 10, 1, 17)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 2, 18)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 3, 19)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 4, 20)
                                
                                self.create_acl_unicast_vlan_inPort(dp, 10, 5, 21)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 6, 22)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 7, 23)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 8, 24)
 
                                self.create_acl_unicast_vlan_inPort(dp, 10, 17, 1)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 18, 2)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 19, 3)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 20, 4)
                                
                                self.create_acl_unicast_vlan_inPort(dp, 10, 21, 5)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 22, 6)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 23, 7)
                                self.create_acl_unicast_vlan_inPort(dp, 10, 24, 8)
 

                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, vlan, srcPort, dst_ip, dstPort)
 
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.111", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.112", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.113", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.114", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.115", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.116", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.117", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.118", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.143", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.144", 26)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.145", 27)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.146", 28)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.147", 37)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.148", 37)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.149", 37)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 25, "192.168.50.150", 37)
                                 
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.111", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.112", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.113", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.114", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.115", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.116", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.117", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.118", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.143", 25)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.144", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.145", 27)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.146", 28)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.147", 37)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.148", 37)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.149", 37)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 26, "192.168.50.150", 37)
                                
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.111", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.112", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.113", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.114", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.115", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.116", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.117", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.118", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.143", 25)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.144", 26)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.145", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.146", 28)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.147", 37)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.148", 37)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.149", 37)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 27, "192.168.50.150", 37)
                                
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.111", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.112", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.113", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.114", 35)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.115", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.116", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.117", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.118", 36)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.143", 25)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.144", 26)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.145", 27)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.146", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.147", 37)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.148", 37)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.149", 37)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 28, "192.168.50.150", 37)
                                
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.111", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.112", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.113", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.114", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.115", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.116", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.117", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.118", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.143", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.144", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.145", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.146", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.147", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.148", 30)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.149", 31)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 29, "192.168.50.150", 32)
                                
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.111", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.112", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.113", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.114", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.115", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.116", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.117", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.118", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.143", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.144", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.145", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.146", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.147", 29)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.148", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.149", 31)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 30, "192.168.50.150", 32)
                                
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.111", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.112", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.113", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.114", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.115", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.116", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.117", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.118", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.143", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.144", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.145", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.146", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.147", 29)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.148", 30)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.149", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 31, "192.168.50.150", 32)
                                
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.111", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.112", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.113", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.114", 33)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.115", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.116", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.117", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.118", 39)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.143", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.144", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.145", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.146", 40)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.147", 29)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.148", 30)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.149", 31)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 32, "192.168.50.150", 34)

                                ############

                                # note actual dpid 5 has two logical ToRs (T2 and T3)
                                # from ToR T2->T0: 35, T2->T1: 36, T2->T3: 37
                                # from ToR T3->T0: 33, T3->T1: 39, T3->T2: 40
                                # dummy port 34 for dpid 5, use the group corresponding to that for all "not"effective rules 

                                
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.111", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.112", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.113", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.114", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.115", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.116", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.117", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.118", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.143", 25)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.144", 26)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.145", 27)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.146", 28)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.147", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.148", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.149", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.150", 34)

                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.111", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.112", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.113", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.114", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.115", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.116", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.117", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.118", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.143", 25)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.144", 26)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.145", 27)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.146", 28)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.147", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.148", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.149", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.150", 34)

                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 37, "192.168.50.111", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 37, "192.168.50.112", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 37, "192.168.50.113", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 37, "192.168.50.114", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 37, "192.168.50.115", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 37, "192.168.50.116", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 37, "192.168.50.117", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 37, "192.168.50.118", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 37, "192.168.50.143", 25)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 37, "192.168.50.144", 26)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 37, "192.168.50.145", 27)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 37, "192.168.50.146", 28)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 37, "192.168.50.147", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 37, "192.168.50.148", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 37, "192.168.50.149", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 37, "192.168.50.150", 34)

                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.111", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.112", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.113", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.114", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.115", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.116", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.117", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.118", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.143", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.144", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.145", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.146", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.147", 29)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.148", 30)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.149", 31)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.150", 32)

                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.111", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.112", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.113", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.114", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.115", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.116", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.117", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.118", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.143", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.144", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.145", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.146", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.147", 29)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.148", 30)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.149", 31)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.150", 32)

                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.111", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.112", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.113", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.114", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.115", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.116", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.117", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.118", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.143", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.144", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.145", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.146", 34)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.147", 29)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.148", 30)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.149", 31)
                                self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.150", 32)


                                ############
                                
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.143", 25)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.144", 26)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.145", 27)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 35, "192.168.50.146", 28)

                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.143", 25)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.144", 26)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.145", 27)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 36, "192.168.50.146", 28)
                                
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 37, "192.168.50.143", 25)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 37, "192.168.50.144", 26)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 37, "192.168.50.145", 27)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 37, "192.168.50.146", 28)

                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.147", 29)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.148", 30)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.149", 31)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 33, "192.168.50.150", 32)

                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.147", 29)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.148", 30)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.149", 31)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 39, "192.168.50.150", 32)

                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.147", 29)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.148", 30)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.149", 31)
                                #self.create_acl_unicast_vlan_inPort_dstIp(dp, 10, 40, "192.168.50.150", 32)
                                

                                
			#elif(dpid == 5 or dpid == 1):
				#print("install vlan rules on switch %i"% dpid)
				#self.tagVlan(dp, defaultVlan)
				#print("install lotus rules on switch %i"% dpid)
				## self.create_acl_unicast_vlan_inPort(dp, defaultVlan, 1, 17, priority=4)
				## self.create_acl_unicast_vlan_inPort(dp, defaultVlan, 8, 24, priority=4)
				#self.LotusSwitch(dp)
				#print("install tor rules on switch %i"%dpid)
				##self.EdgeSwitch(dp)
                                


	def OCS_create_initial_connections(self):
		from OCSRelated.connections import GxcConnections
		from OCSRelated.optionsForShareBackup import Options
		#serverPorts = range(33, 41) + range(1,9)  # [33-40] are connected to server
		#torPorts = range(61, 77)
                #serverPorts = [1,2,5,6]
                #torPorts = [69,70,73,74]
                
                ocs_in_edge = [1, 2, 3, 4, 5, 6, 7, 8, 33, 34, 35, 36, 37, 38, 39, 40, 69, 70, 71, 72, 73, 74, 75, 76, 61, 62, 63, 64, 65, 66, 67, 68]
                ocs_out_edge = [69, 70, 71, 72, 73, 74, 75, 76, 61, 62, 63, 64, 65, 66, 67, 68, 1, 2, 3, 4, 5, 6, 7, 8, 33, 34, 35, 36, 37, 38, 39, 40]
                
                #ocs_in_core = [9, 10, 11, 12, 13, 14, 25, 26, 27, 28, 29, 30]
                #ocs_out_core = [12, 25, 28, 9, 26, 29, 10, 13, 30, 11, 14, 27]

                ocs_in_core = [9, 10, 16, 12, 13, 14, 25, 26, 27, 31, 29, 30]
                ocs_out_core = [12, 25, 31, 9, 26, 29, 10, 13, 30, 16, 14, 27]

                #print (serverPorts + torPorts, torPorts + serverPorts) 
		connectionObj = GxcConnections(Options())
		#connectionObj.ent_crs_fiber(serverPorts + torPorts, torPorts + serverPorts)
                connectionObj.ent_crs_fiber(ocs_in_edge + ocs_in_core, ocs_out_edge + ocs_out_core)
                #connectionObj.ent_crs_fiber(ocs_in_core, ocs_out_core)



if __name__ == "__main__":
	obj = RDC(app_manager.RyuApp)
	DP = namedtuple("DP", 'dpid')
	switchID = [1, 5]
	obj.OCS_create_initial_connections()
	for x in switchID:
		dp = DP(dpid = x)
		obj.initPacketSwitch(dp, x)
