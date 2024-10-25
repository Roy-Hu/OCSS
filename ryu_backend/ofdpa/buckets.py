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
# Implements Buckets object
#
from ofdpa.actions import Actions
from ofdpa.utils import Utils
#
class Buckets():

    def __init__(self, *args, **kwargs):
        super(Buckets, self).__init__(*args, **kwargs)

    @staticmethod
    def create_buckets(dp, config):
        #
        buckets = []
        #
        # print "@@@@@@@@@@@@@@@@@@@@@@"
        # print "buckets config: %s" % config
        # print "@@@@@@@@@@@@@@@@@@@@@@"
        #
        for entry in config:
            #
            # print "entry: %s " % entry

            actions = Actions.create_actions(dp, entry["actions"])

            bucket = dp.ofproto_parser.OFPBucket(
                weight = Utils.to_int(entry["weight"]),
                watch_port = Utils.get_mod_port(dp,entry["watch_port"]),
                watch_group = Utils.get_mod_group(dp,entry["watch_group"]),
                actions = actions
                #entry[]
            ) 
            # print "bucket: %s" % bucket
            buckets.append(bucket)
            # print "buckets: %s" % buckets
        return buckets
