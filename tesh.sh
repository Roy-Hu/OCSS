curl -v -X POST -H "Content-Type: application/json" -d '["192.168.50.111", "192.168.50.113", "192.168.50.114"]' "http://10.224.92.112:8081/startiter/test/allreduce/1"
curl -v -X POST -H "Content-Type: application/json" -d '["192.168.50.111", "192.168.50.113"]' "http://10.224.92.112:8081/startiter/test/allreduce/1"
curl -v -X POST -H "Content-Type: application/json" -d '["192.168.50.111", "192.168.50.113"]' "http://10.224.92.112:8081/enditer/test/allreduce/1"
curl -v -X POST -H "Content-Type: application/json" -d '["192.168.50.111", "192.168.50.113", "192.168.50.114"]' "http://10.224.92.112:8081/enditer/test/allreduce/1"

curl -v -X POST -H "Content-Type: application/json" -d '["192.168.50.113", "192.168.50.114"]' "http://10.224.92.112:8081/startiter/test/allreduce/1"
curl -v -X POST -H "Content-Type: application/json" -d '["192.168.50.113", "192.168.50.114"]' "http://10.224.92.112:8081/enditer/test/allreduce/1"

curl -v -X POST -H "Content-Type: application/json" -d '["192.168.50.111", "192.168.50.113", "192.168.50.114"]' "http://10.224.92.112:8081/startiter/test/allreduce/2"
curl -v -X POST -H "Content-Type: application/json" -d '["192.168.50.111", "192.168.50.113"]' "http://10.224.92.112:8081/startiter/test/allreduce/2"
curl -v -X POST -H "Content-Type: application/json" -d '["192.168.50.111", "192.168.50.113"]' "http://10.224.92.112:8081/enditer/test/allreduce/2"
curl -v -X POST -H "Content-Type: application/json" -d '["192.168.50.111", "192.168.50.113", "192.168.50.114"]' "http://10.224.92.112:8081/enditer/test/allreduce/2"

curl -v -X POST -H "Content-Type: application/json" -d '["192.168.50.113", "192.168.50.114"]' "http://10.224.92.112:8081/startiter/test/allreduce/2"
curl -v -X POST -H "Content-Type: application/json" -d '["192.168.50.113", "192.168.50.114"]' "http://10.224.92.112:8081/enditer/test/allreduce/2"