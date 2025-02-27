# Define the list of ports (last octet of IP) you want to target
PORTS := 111 112 113 114 115 116 117 118 143 144 145 146 147 148 149 150

# Phony targets so they are always rebuilt
.PHONY: all compile mkdir send run

# Default target will compile, create remote directories, and send files
all: compile mkdir send

# 1. Compile the MPI program
compile:
	@echo "Compiling allreduce.c ..."
	mpicc -o allreduce allreduce.c
	mpicc -o allreduce2 allreduce2.c

# 2. Create the directory on each remote node
mkdir: $(PORTS:%=mkdir-%)

mkdir-%:
	@echo "Creating directory on node 10.224.92.$* ..."
	ssh 10.224.92.$* "mkdir -p /home/ch155/OCSS/example"

# 3. Copy the directory to each remote node
send: $(PORTS:%=send-%)

send-%:
	@echo "Copying files to node 10.224.92.$* ..."
	scp -r /home/ch155/OCSS/example 10.224.92.$*:/home/ch155/OCSS/

# 4. Run the MPI program
run:
	@echo "Running the MPI program with ring allreduce ..."
	mpirun --allow-run-as-root \
	       --mca btl_tcp_if_include eth3 \
		   --mca btl_base_verbose 6 \
	       --mca coll_tuned_use_dynamic_rules 1 \
	       --mca coll_tuned_allreduce_algorithm 4 \
	       --rankfile rank.txt \
	       --hostfile hosts.txt \
	       -np 6 \
	       ./allreduce

run2:
	@echo "Running the MPI program with ring allreduce ..."
	mpirun --allow-run-as-root \
			--mca btl_tcp_if_include eth3 \
			--mca btl_base_verbose 6 \
			--mca coll_tuned_use_dynamic_rules 1 \
			--mca coll_tuned_allreduce_algorithm 4 \
			--rankfile rank2.txt \
			--hostfile hosts2.txt \
			-np 6 \
			./allreduce2