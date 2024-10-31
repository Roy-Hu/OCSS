#!/bin/bash
curl -X POST http://127.0.0.1:8010/rdc/set_switch_info/0000000000000001 \
-H "Content-Type: application/json" \
-d '{
  "hostPorts": [1],
  "switchPorts": [17, 25, 33]
}'

curl -X POST http://127.0.0.1:8010/rdc/set_switch_info/0000000000000005 \
-H "Content-Type: application/json" \
-d '{
  "hostPorts": [5],
  "switchPorts": [21, 29, 33]
}'

curl -X POST http://127.0.0.1:8010/rdc/forwardingtable/0000000000000001 \
-H "Content-Type: application/json" \
-d '{
  "entries": [
    [1, 17],
    [25, 33]
  ]
}'

curl -X POST http://127.0.0.1:8010/rdc/forwardingtable/0000000000000005 \
-H "Content-Type: application/json" \
-d '{
  "entries": [
    [5, 21],
    [29, 33]
  ]
}'

curl -X POST http://127.0.0.1:8010/rdc/set_ocs/127.0.0.1 \
-H "Content-Type: application/json" \
-d '{
  "ocs_in_port": [1, 37, 16],
  "ocs_out_port": [69, 65, 31]
}'
