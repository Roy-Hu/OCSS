package ocss

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	ocss_context "github.com/comp590/ocss/internal/context"
	"github.com/comp590/ocss/internal/logger"
	"github.com/comp590/ocss/pkg/app"
	"github.com/gorilla/mux"
)

type HttpServerOCSS interface {
	app.App
}

type HttpServer struct {
	HttpServerOCSS
}

func NewHttpServer(ocss HttpServerOCSS) (*HttpServer, error) {
	o := &HttpServer{
		HttpServerOCSS: ocss,
	}

	return o, nil
}

// HandlePostTrafficMatrix handles POST requests to receive traffic matrix data.
func (s *HttpServer) HandlePostStartIter(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appId := vars["appId"]
	op := vars["op"]
	iter := vars["iter"]
	self := ocss_context.GetSelf()

	if appId == "" {
		http.Error(w, "appId is required in the URL", http.StatusBadRequest)
		return
	}

	// Decode the JSON payload
	var payload []string
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&payload); err != nil {
		logger.HttpLog.Errorf("Error decoding JSON payload: %v", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	iterNum, err := strconv.Atoi(iter)
	if err != nil {
		logger.HttpLog.Errorf("Error converting iteration number to integer: %v", err)
		http.Error(w, "Invalid iteration number", http.StatusBadRequest)
		return
	}

	servers := make([]string, len(payload))

	for i, ips := range payload {
		server := ocss_context.GetServerByIp(ips)
		if server == "" {
			logger.HttpLog.Errorf("Server not found for IP %s", ips)
			http.Error(w, "Server not found", http.StatusBadRequest)
			return
		}

		servers[i] = server
	}

	opInfo := &ocss_context.OpInfo{
		App:     appId,
		Op:      op,
		Servers: servers,
		Status:  ocss_context.START,
		Iter:    iterNum,
	}

	self.OpInfoChan <- opInfo

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Start iter received successfully"))
}

// HandlePostTrafficMatrix handles POST requests to receive traffic matrix data.
func (s *HttpServer) HandlePostEndIter(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appId := vars["appId"]
	op := vars["op"]
	iter := vars["iter"]
	self := ocss_context.GetSelf()

	if appId == "" {
		http.Error(w, "appId is required in the URL", http.StatusBadRequest)
		return
	}

	// Decode the JSON payload
	var payload []string
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&payload); err != nil {
		logger.HttpLog.Errorf("Error decoding JSON payload: %v", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	iterNum, err := strconv.Atoi(iter)
	if err != nil {
		logger.HttpLog.Errorf("Error converting iteration number to integer: %v", err)
		http.Error(w, "Invalid iteration number", http.StatusBadRequest)
		return
	}

	servers := make([]string, len(payload))

	for i, ips := range payload {
		server := ocss_context.GetServerByIp(ips)
		if server == "" {
			logger.HttpLog.Errorf("Server not found for IP %s", ips)
			http.Error(w, "Server not found", http.StatusBadRequest)
			return
		}

		servers[i] = server
	}

	opInfo := &ocss_context.OpInfo{
		App:     appId,
		Op:      op,
		Servers: servers,
		Status:  ocss_context.END,
		Iter:    iterNum,
	}

	self.OpInfoChan <- opInfo

	logger.HttpLog.Infof("Finished iteration %v for app %v", iterNum, appId)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("End iter received successfully"))
}

// setupRoutes configures the HTTP routes using Gorilla Mux.
func (s *HttpServer) setupRoutes(router *mux.Router) {
	router.HandleFunc("/startiter/{appId}/{op}/{iter}", s.HandlePostStartIter).Methods("POST")
	router.HandleFunc("/enditer/{appId}/{op}/{iter}", s.HandlePostEndIter).Methods("POST")
}

func (s *HttpServer) Start(ctx context.Context, wg *sync.WaitGroup) {
	// Increment the WaitGroup count by 1 for this component
	wg.Add(1)

	// Run everything in a goroutine so this method doesn’t block forever.
	go func() {
		// Decrement the WaitGroup count when the function returns
		defer wg.Done()

		// Create a new router
		router := mux.NewRouter()
		s.setupRoutes(router)

		self := ocss_context.GetSelf()

		// Create a new http.Server so we can manage graceful shutdown
		srv := &http.Server{
			Addr:    self.HttpServerAddr,
			Handler: loggingMiddleware(router),
		}

		// Start the server in its own goroutine
		go func() {
			logger.HttpLog.Infof("Starting OCSS Server %s...", srv.Addr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.HttpLog.Errorf("Failed to start server: %v", err)
			}
		}()

		// Block until the context is canceled or deadline exceeded
		<-ctx.Done()

		// Attempt a graceful shutdown
		logger.HttpLog.Infof("Shutting down OCSS server...")

		// Create a fresh context with a timeout for the shutdown process
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Shutdown the server
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.HttpLog.Errorf("OCSS server shutdown failed: %v", err)
		} else {
			logger.HttpLog.Info("OCSS server stopped gracefully")
		}
	}()
}

// loggingMiddleware logs incoming HTTP requests.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.HttpLog.Infof("[%s] %s %s %s", r.RemoteAddr, r.Method, r.RequestURI, r.Proto)
		next.ServeHTTP(w, r)
	})
}

// getEnv retrieves environment variables or returns a default value if not set.
func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
