package ocss

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"os"
// 	"sync"

// 	"github.com/comp590/ocss/internal/logger"
// 	"github.com/comp590/ocss/pkg/app"
// 	"github.com/gorilla/mux"
// )

// // TrafficMatrix represents the traffic data between source and destination IPs.
// type TrafficMatrix map[string]map[string]int64

// type HttpServerOCSS interface {
// 	app.App
// }

// type HttpServer struct {
// 	HttpServerOCSS
// }

// func NewHttpServer(ocss HttpServerOCSS) (*HttpServer, error) {
// 	o := &HttpServer{
// 		HttpServerOCSS: ocss,
// 	}

// 	return o, nil
// }

// // HandlePostTrafficMatrix handles POST requests to receive traffic matrix data.
// func (s *HttpServer) HandlePostTrafficMatrix(w http.ResponseWriter, r *http.Request) {
// 	vars := mux.Vars(r)
// 	dpid := vars["dpid"]

// 	if dpid == "" {
// 		http.Error(w, "dpid is required in the URL", http.StatusBadRequest)
// 		return
// 	}

// 	// Decode the JSON payload
// 	var payload map[string]TrafficMatrix
// 	decoder := json.NewDecoder(r.Body)
// 	if err := decoder.Decode(&payload); err != nil {
// 		logger.HttpLog.Infof("Error decoding JSON payload: %v", err)
// 		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
// 		return
// 	}

// 	// Extract traffic matrix for the given dpid
// 	matrix, exists := payload[dpid]
// 	if !exists {
// 		logger.HttpLog.Infof("Traffic matrix for dpid %s not found in payload", dpid)
// 		http.Error(w, fmt.Sprintf("Traffic matrix for dpid %s not found", dpid), http.StatusBadRequest)
// 		return
// 	}

// 	// // Store the traffic matrix
// 	// s.mu.Lock()
// 	// s.trafficMatrices[dpid] = matrix
// 	// s.mu.Unlock()

// 	logger.HttpLog.Infof("Received and stored traffic matrix for dpid %s: %v", dpid, matrix)

// 	w.WriteHeader(http.StatusOK)
// 	w.Write([]byte("Traffic matrix received successfully"))
// }

// // setupRoutes configures the HTTP routes using Gorilla Mux.
// func (s *HttpServer) setupRoutes(router *mux.Router) {
// 	router.HandleFunc("/traffic_matrix/{dpid}", s.HandlePostTrafficMatrix).Methods("POST")
// }

// func (s *HttpServer) Start(ctx context.Context, wg *sync.WaitGroup) {
// 	// Create a new router
// 	router := mux.NewRouter()
// 	s.setupRoutes(router)

// 	// Configure server port
// 	port := getEnv("OCSS_SERVER_PORT", "8081")

// 	// Start the HTTP server
// 	logger.HttpLog.Infof("Starting OCSS Server on port %s...", port)
// 	if err := http.ListenAndServe(":"+port, loggingMiddleware(router)); err != nil {
// 		logger.HttpLog.Errorf("Failed to start server: %v", err)
// 	}
// }

// // loggingMiddleware logs incoming HTTP requests.
// func loggingMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		logger.HttpLog.Infof("[%s] %s %s %s", r.RemoteAddr, r.Method, r.RequestURI, r.Proto)
// 		next.ServeHTTP(w, r)
// 	})
// }

// // getEnv retrieves environment variables or returns a default value if not set.
// func getEnv(key, defaultVal string) string {
// 	if value, exists := os.LookupEnv(key); exists {
// 		return value
// 	}
// 	return defaultVal
// }
