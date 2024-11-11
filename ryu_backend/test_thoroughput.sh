#!/bin/bash
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