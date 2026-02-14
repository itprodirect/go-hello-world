package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/itprodirect/go-hello-world/internal/greeter"
	"github.com/itprodirect/go-hello-world/internal/metrics"
)

type helloResponse struct {
	Message string `json:"message"`
	Count   uint64 `json:"count"`
}

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	counters := metrics.NewCounters()
	startedAt := time.Now()

	mux := http.NewServeMux()

	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		name := r.URL.Query().Get("name")
		count := counters.Inc("hello_requests")
		message := greeter.BuildGreeting(name, int(count))

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(w).Encode(helloResponse{
			Message: message,
			Count:   count,
		}); err != nil {
			logger.Printf("encode /hello response: %v", err)
		}
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		counters.Inc("health_requests")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		counters.Inc("metrics_requests")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(counters.PlainText()))
	})

	handler := instrumentRequests(counters, mux)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go runUptimeTicker(ctx, logger, counters, startedAt)

	go func() {
		<-ctx.Done()
		logger.Println("shutdown signal received")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Printf("server shutdown error: %v", err)
		}
	}()

	logger.Printf("hello-server listening on http://localhost%s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("server error: %v", err)
	}

	logger.Println("server stopped")
}

func runUptimeTicker(ctx context.Context, logger *log.Logger, counters *metrics.Counters, startedAt time.Time) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			tick := counters.Inc("uptime_ticks")
			uptime := now.Sub(startedAt).Round(time.Second)
			logger.Printf("uptime tick #%d (%s)", tick, uptime)
		}
	}
}

func instrumentRequests(counters *metrics.Counters, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counters.Inc("http_requests_total")

		path := strings.Trim(r.URL.Path, "/")
		if path == "" {
			path = "root"
		}
		counters.Inc("path_" + path + "_requests")

		next.ServeHTTP(w, r)
	})
}
