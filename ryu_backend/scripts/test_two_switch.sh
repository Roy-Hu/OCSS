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

curl -X POST http://127.0.0.1:8010/rdc/createforwardingtable/0000000000000001 \
-H "Content-Type: application/json" \
-d '{
  "entries": [
    {"in_port": 1, "out_port": 17, "src_ip": "", "dst_ip": ""},
    {"in_port": 25, "out_port": 33, "src_ip": "", "dst_ip": ""}
  ]
}'

curl -X POST http://127.0.0.1:8010/rdc/createforwardingtable/0000000000000005 \
-H "Content-Type: application/json" \
-d '{
  "entries": [
    {"in_port": 5, "out_port": 21, "src_ip": "192.168.50.111", "dst_ip": "192.168.50.147"},
    {"in_port": 29, "out_port": 33, "src_ip": "192.168.50.147", "dst_ip": "192.168.50.111"}
  ]
}'

curl -X POST http://127.0.0.1:8010/rdc/set_ocs/127.0.0.1 \
-H "Content-Type: application/json" \
-d '{
  "ocs_in_port": [1, 37, 16],
  "ocs_out_port": [69, 65, 31]
}'

while true; do
    curl -X POST http://127.0.0.1:8010/rdc/set_ocs/127.0.0.1 \
    -H "Content-Type: application/json" \
    -d '{
    "ocs_in_port": [1, 37, 16,2,3,4,5,6,7,8,33,34,35,36,38,39,40],
    "ocs_out_port": [69, 65, 31 ,70,71,72,73,74,75,76,61,62,63,64,66,67,68]
    }'

    sleep 0.5

    curl -X POST http://127.0.0.1:8010/rdc/set_ocs/127.0.0.1 \
    -H "Content-Type: application/json" \
    -d '{
    "ocs_in_port": [1, 37, 16,2,3,4,5,6,7,8,33,34,35,36,38,39,40],
    "ocs_out_port": [65, 69, 31,70,71,72,73,74,75,76,61,62,63,64,66,67,68]
    }'

    sleep 0.5
done