#include <stdio.h>
#include <stdlib.h>
#include <mpi.h>

int main(int argc, char *argv[]) {
    int rank, size;
    MPI_Init(&argc, &argv);                   // Initialize MPI
    MPI_Comm_rank(MPI_COMM_WORLD, &rank);     // Get this process rank
    MPI_Comm_size(MPI_COMM_WORLD, &size);     // Get the total number of processes

    // Define dimensions for the 2D matrix
    int rows = 400;
    int cols = 500;
    int total_elements = rows * cols;

    // Allocate memory for the local matrix and the result matrix.
    double *local_matrix = (double *)malloc(total_elements * sizeof(double));
    double *global_matrix = (double *)malloc(total_elements * sizeof(double));

    // Initialize the local matrix. Example: each element is 1.0
    int i;
    for (i = 0; i < total_elements; i++) {
        local_matrix[i] = (rank + 1);
    }

    // Rank 0 notifies the external traffic collector that we're starting iteration #1
    if (rank == 0) {
        system(
            "curl -v -X POST -H \"Content-Type: application/json\" "
            "-d '{"
              "\"192.168.50.111\": [\"192.168.50.113\", \"192.168.50.114\", \"192.168.50.147\", \"192.168.50.148\", \"192.168.50.149\"],"
              "\"192.168.50.113\": [\"192.168.50.111\", \"192.168.50.114\", \"192.168.50.147\", \"192.168.50.148\", \"192.168.50.149\"],"
              "\"192.168.50.114\": [\"192.168.50.111\", \"192.168.50.113\", \"192.168.50.147\", \"192.168.50.148\", \"192.168.50.149\"],"
              "\"192.168.50.147\": [\"192.168.50.111\", \"192.168.50.113\", \"192.168.50.114\", \"192.168.50.148\", \"192.168.50.149\"],"
              "\"192.168.50.148\": [\"192.168.50.111\", \"192.168.50.113\", \"192.168.50.114\", \"192.168.50.147\", \"192.168.50.149\"],"
              "\"192.168.50.149\": [\"192.168.50.111\", \"192.168.50.113\", \"192.168.50.114\", \"192.168.50.147\", \"192.168.50.148\"]"
            "}' "
            "http://10.224.92.112:8081/startiter/allreduce/1"
        );        
    }

    // Perform an Allreduce operation (sum all elements across ranks)
    MPI_Allreduce(local_matrix, global_matrix, total_elements, MPI_DOUBLE, MPI_SUM, MPI_COMM_WORLD);

    // Optional: barrier to ensure *all processes* have completed any post-Allreduce work
    MPI_Barrier(MPI_COMM_WORLD);
    
    // Rank 0 notifies that the iteration has ended (all Allreduce calls are finished)
    if (rank == 0) {
        sleep(3);
        system("curl -v -X POST -H \"Content-Type: application/json\" http://10.224.92.112:8081/enditer/allreduce/1");
    }

    // Optionally print some results on rank 0
    if (rank == 0) {
        printf("Global matrix after MPI_Allreduce (sum):\n");
        // e.g., confirm the sum is size * 1.0 per element
        for (i = 0; i < 10; i++) {
            printf("%f ", global_matrix[i]);
        }
    }

    // Clean up
    free(local_matrix);
    free(global_matrix);
    MPI_Finalize();  // Finalize MPI

    return 0;
}
