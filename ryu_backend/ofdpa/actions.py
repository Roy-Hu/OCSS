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

# Implements Actions object
#
#
from ofdpa.utils import Utils
from ryu.lib import mac
#
class Actions():

    def __init__(self, *args, **kwargs):
        super(Actions, self).__init__(*args, **kwargs)

    @staticmethod
    def create_actions(dp,config):
        #
        actions = []
        #
        # actions config is a list
        # print "actions config: %s" % config
        for action in config:
            for key in action.keys():
                #
                val = action[key]
                #
                # print "key: %s" % key
                # print "val: %s" % val
                #
                if (key == "set_field"):
                    actions.append(Actions.action_set(dp,val))

                elif(key == "group"):
                    actions.append(Actions.action_group(dp,val))

                elif (key == "output"):
                    actions.append(Actions.action_output(dp,val))

                elif (key == "push_vlan"):
                    actions.append(Actions.action_push_vlan(dp,val))

                elif (key == "push_mpls"):
                    actions.append(Actions.action_push_mpls(dp,val))

                elif (key == "push_pbb"):
                    actions.append(Actions.action_push_pbb(dp,val))

                elif (key == "pop"):
                    # print "======================="
                    # print "key: %s" % key
                    # print "val: %s" % val
                    # print "======================="
                    actions.append(Actions.action_pop(dp,val))

                elif (key == "copy_ttl"):
                    actions.append(Actions.action_copy_ttl(dp,val))

                elif (key == "dec_ttl"):
                    actions.append(Actions.action_dec_ttl(dp,val))

                elif (key == "qos"):
                    actions.append(Actions.action_qos(dp,val))

                elif (key == "set_queue"):
                    actions.append(Actions.action_queue(dp,val))

                else:
                    raise Exception("Wrong action name:", key)

        # print "actions: %s" % actions
        return actions

    @staticmethod
    def action_set(dp,conf):
        for key in conf.keys():
                val = conf[key]
                #
                # print "key->: %s" % key
                # print "val->: %s" % val
                #
                if(key == "vlan_vid"):
                    return dp.ofproto_parser.OFPActionSetField(vlan_vid=Utils.to_int(val))

                elif(key == "eth_src"):
                    return dp.ofproto_parser.OFPActionSetField(eth_src=val)

                elif(key == "eth_dst"):
                    return dp.ofproto_parser.OFPActionSetField(eth_dst=val)
                
                elif(key == "vlan_pcp"):
                    return dp.ofproto_parser.OFPActionSetField(vlan_pcp=Utils.to_int(val))

                elif(key == "group_id"):
                    return dp.ofproto_parser.OFPActionSetField(vlan_vid=Utils.to_int(val))

                else:
                    raise Exception("Wrong filed name:", key)



    @staticmethod
    def action_group(dp,conf):
        return dp.ofproto_parser.OFPActionGroup(Utils.to_int(conf["group_id"]))

    @staticmethod
    def action_output(dp,conf):
        # Xiaoye
        # return dp.ofproto_parser.OFPActionOutput(int(conf["port"]))
        return dp.ofproto_parser.OFPActionOutput(Utils.get_mod_port(dp, conf["port"]))
    
    @staticmethod
    def action_push_vlan(dp,conf):
        return dp.ofproto_parser.OFPActionPushVlan()

    @staticmethod
    def action_push_mpls(dp,conf):
        return dp.ofproto_parser.OFPActionPushMpls()

    @staticmethod
    def action_push_pbb(dp,conf):
        ethertype = 0
        return dp.ofproto_parser.OFPActionPushPbb(ethertype)

    @staticmethod
    def action_pop(dp,conf):
        if(conf == "vlan"):
            return dp.ofproto_parser.OFPActionPopVlan()

    @staticmethod
    def action_copy_ttl(dp,conf):
        pass
        #return dp.ofproto_parser.OFPActionCopyTtlIn()
        #return dp.ofproto_parser.OFPActionCopyTtlOut()

    @staticmethod
    def action_dec_ttl(dp,conf):
        pass
        #return dp.ofproto_parser.OFPActionDecNwTtl()

    @staticmethod
    def action_qos(dp,conf):
        return dp.ofproto_parser.OFPAction

    @staticmethod
    def action_queue(dp,conf):
        return dp.ofproto_parser.OFPActionSetQueue(Utils.to_int(conf["queue_id"]))
        

