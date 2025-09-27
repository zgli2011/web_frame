package check

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	httpServer *http.Server
	router     *mux.Router
	port       int
	routes     []RouteInfo
	mu         sync.RWMutex
	config     *EndpointConfig
}

type EndpointConfig struct {
	HealthEnabled  bool
	APIsEnabled    bool
	MetricsEnabled bool
}

type RouteInfo struct {
	Path    string `json:"path"`
	Method  string `json:"method"`
	Handler string `json:"handler"`
}

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Uptime    string    `json:"uptime"`
	Version   string    `json:"version,omitempty"`
}

type APIListResponse struct {
	Count           int        `json:"count"`
	BuiltinAPIs     []RouteInfo `json:"builtin_apis"`
	RegisteredAPIs  []*APIInfo `json:"registered_apis"`
	TotalCount      int        `json:"total_count"`
}

var (
	startTime = time.Now()

	// Prometheus metrics
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status_code"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "Duration of HTTP requests in seconds",
		},
		[]string{"method", "path"},
	)

	activeConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_connections",
			Help: "Number of active connections",
		},
	)
)

func init() {
	// Register metrics with Prometheus
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(activeConnections)
}

func NewServer(port int) *Server {
	return NewServerWithConfig(port, &EndpointConfig{
		HealthEnabled:  true,
		APIsEnabled:    true,
		MetricsEnabled: true,
	})
}

func NewServerWithConfig(port int, config *EndpointConfig) *Server {
	router := mux.NewRouter()

	server := &Server{
		router: router,
		port:   port,
		routes: make([]RouteInfo, 0),
		config: config,
	}

	// Register built-in endpoints based on configuration
	server.registerBuiltinEndpoints()

	return server
}

func (s *Server) registerBuiltinEndpoints() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Register health endpoint if enabled
	if s.config.HealthEnabled {
		s.router.HandleFunc("/health", s.healthHandler).Methods("GET")
		s.routes = append(s.routes, RouteInfo{Path: "/health", Method: "GET", Handler: "healthHandler"})
	}

	// Register APIs endpoint if enabled
	if s.config.APIsEnabled {
		s.router.HandleFunc("/apis", s.apiListHandler).Methods("GET")
		s.routes = append(s.routes, RouteInfo{Path: "/apis", Method: "GET", Handler: "apiListHandler"})
	}

	// Register metrics endpoint if enabled
	if s.config.MetricsEnabled {
		s.router.Handle("/metrics", promhttp.Handler()).Methods("GET")
		s.routes = append(s.routes, RouteInfo{Path: "/metrics", Method: "GET", Handler: "prometheusHandler"})
		// Add middleware for metrics only if metrics are enabled
		s.router.Use(s.metricsMiddleware)
	}
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(startTime)

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Uptime:    uptime.String(),
		Version:   "1.0.0", // 可以从配置或环境变量中获取
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (s *Server) apiListHandler(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	builtinRoutes := make([]RouteInfo, len(s.routes))
	copy(builtinRoutes, s.routes)
	s.mu.RUnlock()

	// Get registered APIs from registry
	registeredAPIs := GetAllAPIs()

	// Support filtering by query parameters
	query := r.URL.Query()
	if tag := query.Get("tag"); tag != "" {
		registeredAPIs = GetAPIsByTag(tag)
	}
	if service := query.Get("service"); service != "" {
		registeredAPIs = GetAPIsByService(service)
	}

	response := APIListResponse{
		Count:          len(builtinRoutes),
		BuiltinAPIs:    builtinRoutes,
		RegisteredAPIs: registeredAPIs,
		TotalCount:     len(builtinRoutes) + len(registeredAPIs),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (s *Server) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		activeConnections.Inc()
		defer activeConnections.Dec()

		// Wrap response writer to capture status code
		wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrappedWriter, r)

		duration := time.Since(start).Seconds()
		statusCode := fmt.Sprintf("%d", wrappedWriter.statusCode)

		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, statusCode).Inc()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (s *Server) RegisterRoute(path, method, handler string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.routes = append(s.routes, RouteInfo{
		Path:    path,
		Method:  method,
		Handler: handler,
	})
}

func (s *Server) GetRouter() *mux.Router {
	return s.router
}

func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Server starting on port %d", s.port)
	log.Println("Available endpoints:")
	if s.config.HealthEnabled {
		log.Printf("  - Health check: http://localhost:%d/health", s.port)
	}
	if s.config.APIsEnabled {
		log.Printf("  - API list: http://localhost:%d/apis", s.port)
	}
	if s.config.MetricsEnabled {
		log.Printf("  - Metrics: http://localhost:%d/metrics", s.port)
	}

	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}