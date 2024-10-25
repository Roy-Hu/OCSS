#*********************************************************************
#
# (C) Copyright Broadcom Corporation 2013-2014
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.
#
#*********************************************************************

#
# Implements Matches object
#

from ryu.lib import mac
from ryu.lib import ip

class Matches():

    def __init__(self, *args, **kwargs):
        super(Matches, self).__init__(*args, **kwargs)

    @staticmethod
    def create_matches(dp, config):
        #
        #TODO: vlan mask
        matches = dp.ofproto_parser.OFPMatch()
        is_vlan_vid_mask = 0
        #
        # matches config is a list
        # print "matches config: %s" % config
        #
        for key in config.keys():
            #
            #print "key: %s" % key
            #print "val: %s" % config[key]
            #
            val = config[key]
            #
            if (key == "in_port"):
                matches.set_in_port(int(val))

            elif (key == "in_phy_port"):
                matches.set_in_phy_port(int(val))

            elif (key == "metadata"):
                matches.set_metadata(int(val))

            elif (key == "eth_dst"):
                matches.set_dl_dst(mac.haddr_to_bin(val))

            elif (key == "eth_dst_mask"):
                haddr = mac.haddr_to_bin(config["eth_dst"])
                mask = mac.haddr_to_bin(val)
                matches.set_dl_dst_masked(haddr,mask)

            elif (key == "eth_src"):
                matches.set_dl_src(mac.haddr_to_bin(val))

            elif (key == "eth_src_mask"):
                haddr = mac.haddr_to_bin(config["eth_src"])
                mask = mac.haddr_to_bin(val)
                matches.set_dl_src_masked(mac,mask)

            elif (key == "eth_type"):
                matches.set_dl_type(int(val,16))

            elif (key == "vlan_vid"):
                # print "*******************************"
                # print "***vlan_vid"
                # print "***val: %s" % int(val)
                # print "*******************************"            
                if (is_vlan_vid_mask == 0):
                    matches.set_vlan_vid(int(val))

            elif (key == "vlan_vid_mask"):
                vid = int(config["vlan_vid"])
                mask =  (int(val,16))
                #vid = 0
                #mask = 0x0fff                
                matches.set_vlan_vid_masked(vid, mask)
                is_vlan_vid_mask = 1
                # print "*******************************"
                # print "***vlan_vid_mask"
                # print "***val: %s" % val
                # print "***vid: %d" % vid                
                # print "***mask: %x" % mask
                # print "***vlan_vid_mask: %s" % matches._wc.vlan_vid_mask             
                # print "*******************************"
                
            elif (key == "vlan_pcp"):
                matches.set_vlan_pcp(int(val))

            elif (key == "ip_dscp"):
                raise Exception("Wrong match name:", key)

            elif (key == "ip_ecn"):
                raise Exception("Wrong match name:", key)

            elif (key == "ip_proto"):
                matches.set_ip_proto(int(val))

            elif (key == "ipv4_src"):
                matches.set_ipv4_src(Matches.ipv4_to_int(val))

            elif (key == "ipv4_src_mask"):
                addr = Matches.ipv4_to_int(config["ipv4_src"])
                mask = Matches.mask_ntob(int(val))
                matches.set_ipv4_src_masked(addr, mask)

            elif (key == "ipv4_dst"):
                matches.set_ipv4_dst(Matches.ipv4_to_int(val))

            elif (key == "ipv4_dst_mask"):
                addr = Matches.ipv4_to_int(config["ipv4_dst"])
                mask = Matches.mask_ntob(int(val))
                # print "======================================================="
                # print addr
                # print mask
                # print "======================================================="
                matches.set_ipv4_dst_masked(addr, mask)

            elif (key == "tcp_src"):
                matches.set_tcp_src(int(val))

            elif (key == "tcp_dst"):
                matches.set_tcp_dst(int(val))

            elif (key == "udp_src"):
                matches.set_udp_src(int(val))

            elif (key == "udp_dst"):
                matches.set_tcp_dst(int(val))

            elif (key == "sctp_src"):
                raise Exception("Wrong match name:", key)

            elif (key == "sctp_dst"):
                raise Exception("Wrong match name:", key)

            elif (key == "icmpv4_type"):
                raise Exception("Wrong match name:", key)
            
            elif (key == "icmpv4_code"):
                raise Exception("Wrong match name:", key)

            elif (key == "arp_op"):
                raise Exception("Wrong match name:", key)

            elif (key == "arp_spa"):
                raise Exception("Wrong match name:", key)

            elif (key == "arp_tpa"):
                raise Exception("Wrong match name:", key)

            elif (key == "arp_sha"):
                raise Exception("Wrong match name:", key)

            elif (key == "arp_tha"):
                raise Exception("Wrong match name:", key)

            elif (key == "ipv6_src"):
                raise Exception("Wrong match name:", key)

            elif (key == "ipv6_dst"):
                raise Exception("Wrong match name:", key)

            elif (key == "ipv6_flabel"):
                raise Exception("Wrong match name:", key)

            elif (key == "icmpv6_type"):
                raise Exception("Wrong match name:", key)

            elif (key == "icmpv6_code"):
                raise Exception("Wrong match name:", key)

            elif (key == "ipv6_nd_target"):
                raise Exception("Wrong match name:", key)

            elif (key == "ipv6_nd_sll"):
                raise Exception("Wrong match name:", key)

            elif (key == "ipv6_nd_tll"):
                raise Exception("Wrong match name:", key)

            elif (key == "mpls_label"):
                raise Exception("Wrong match name:", key)

            elif (key == "mpls_tc"):
                raise Exception("Wrong match name:", key)

            elif (key == "mpls_bos"):
                raise Exception("Wrong match name:", key)

            elif (key == "pbb_isid"):
                raise Exception("Wrong match name:", key)

            elif (key == "tunnel_id"):
                matches.set_tunnel_id(int(val))

            elif (key == "ipv6_exthdr"):
                raise Exception("Wrong match name:", key)

            else:
                raise Exception("Wrong match name:", key)

        #print "matches: %s" % matches
        return matches

    @staticmethod
    def ipv4_to_int(string):
        ip = string.split('.')
        assert len(ip) == 4
        i = 0
        for b in ip:
            b = int(b)
            i = (i << 8) | b
        return i



    @staticmethod
    def ipv4_text_to_int(ip_text):
        if ip_text == 0:
            return ip_text
        assert isinstance(ip_text, str)
        return struct.unpack('!I', addrconv.ipv4.text_to_bin(ip_text))[0]

    @staticmethod
    def mask_ntob(mask, err_msg=None):
        try:
            return (0xffffffff << (32 - mask)) & 0xffffffff
        except ValueError:
            msg = 'illegal netmask'
            if err_msg is not None:
                msg = '%s %s' % (err_msg, msg)
            raise ValueError(msg)



