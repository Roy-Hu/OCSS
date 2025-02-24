#include <stdio.h>
#include <stdlib.h>
#include <mpi.h>

int main(int argc, char *argv[]) {
    int rank, size;
    MPI_Init(&argc, &argv);                   // Initialize MPI
    MPI_Comm_rank(MPI_COMM_WORLD, &rank);       // Get the current process rank
    MPI_Comm_size(MPI_COMM_WORLD, &size);       // Get the total number of processes

    // Define dimensions for the 2D matrix
    int rows = 4;
    int cols = 5;
    int total_elements = rows * cols;

    // Allocate memory for the local matrix and the result matrix.
    // We treat the 2D matrix as a contiguous block (row-major order)
    double *local_matrix = (double *)malloc(total_elements * sizeof(double));
    double *global_matrix = (double *)malloc(total_elements * sizeof(double));

    // Initialize the local matrix with a value that depends on the process rank.
    // For example, each element is set to the rank of the process.
    int i = 0, j = 0;
    for (i = 0; i < total_elements; i++) {
        local_matrix[i] = 1.0;
    }

    if (rank == 0) {
        system("curl -v -X POST -H \"Content-Type: application/json\" -d '{\"192.168.50.111\": [\"192.168.50.147\"], \"192.168.50.147\": [\"192.168.50.111\"]}' http://10.224.92.112:8081/startiter/allreduce/1");
    } 

    // Perform an Allreduce operation using MPI_SUM.
    // This will sum corresponding elements of the matrices from all processes.
    MPI_Allreduce(local_matrix, global_matrix, total_elements, MPI_DOUBLE, MPI_SUM, MPI_COMM_WORLD);
    sleep(1);
    
    if (rank == 0) {
        system("curl -v -X POST -H \"Content-Type: application/json\" http://10.224.92.112:8081/enditer/allreduce/1");
    }

    // Optionally, print the resulting global matrix on one of the processes.
    if (rank == 0) {
        printf("Global matrix after MPI_Allreduce (sum):\n");
        for (i = 0; i < rows; i++) {
            for (j = 0; j < cols; j++) {
                printf("%5.1f ", global_matrix[i * cols + j]);
            }
            printf("\n");
        }
    }

    // Clean up
    free(local_matrix);
    free(global_matrix);
    MPI_Finalize();  // Finalize MPI
    return 0;
}
