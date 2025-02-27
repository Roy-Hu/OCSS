from mpi4py import MPI
import numpy as np
import requests

comm = MPI.COMM_WORLD
rank = comm.Get_rank()
size = comm.Get_size()

# Map MPI rank to node IP (adjust or extend as needed)
node_ips = {
    0: "192.168.50.111",
    1: "192.168.50.147"
}

# Determine the current node's source IP and the destination IP (next rank in a ring)
src_ip = node_ips.get(rank, "127.0.0.1")
dst_rank = (rank + 1) % size  # next rank (wrap-around)
dst_ip = node_ips.get(dst_rank, "127.0.0.1")

# Application parameters
appId = "allreduce"  # Replace with your application ID
iteration = 1        # Replace with the desired iteration number
port = 8080          # Port where your HTTP server is listening

# Construct the HTTP endpoint URLs (using the node's own IP)
start_url = "http://{}:{}/startiter/{}/{}".format(src_ip, port, appId, iteration)
end_url   = "http://{}:{}/enditer/{}/{}".format(src_ip, port, appId, iteration)

# Build the payload for start iteration: a map from the source IP to a list of destination IPs.
payload_start = { src_ip: [dst_ip] }

# --- Send HTTP POST to signal the start of the iteration ---
try:
    response_start = requests.post(start_url, json=payload_start)
    print("Rank {} (src IP {}) sent startiter with payload {}, response: {}".format(rank, src_ip, payload_start, response_start.status_code))
except Exception as e:
    print("Rank {} (src IP {}) failed to send startiter: {}".format(rank, src_ip, e))

# --- Perform a global allreduce operation (example: summing rank values) ---
send_data = np.array([rank], dtype='int')
recv_data = np.empty_like(send_data)
comm.Allreduce(send_data, recv_data, op=MPI.SUM)
print("Rank {}: Allreduce result = {}".format(rank, recv_data[0]))

# --- Send HTTP POST to signal the end of the iteration ---
try:
    response_end = requests.post(end_url, json={})
    print("Rank {} (src IP {}) sent enditer, response: {}".format(rank, src_ip, response_end.status_code))
except Exception as e:
    print("Rank {} (src IP {}) failed to send enditer: {}".format(rank, src_ip, e))
