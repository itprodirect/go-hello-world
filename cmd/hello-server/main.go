package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/itprodirect/go-hello-world/internal/apperror"
	"github.com/itprodirect/go-hello-world/internal/config"
	"github.com/itprodirect/go-hello-world/internal/greeter"
	"github.com/itprodirect/go-hello-world/internal/metrics"
	"github.com/itprodirect/go-hello-world/internal/middleware"
)

type helloResponse struct {
	Message string `json:"message"`
	Count   uint64 `json:"count"`
}

func main() {
	cfgPath := flag.String("config", "", "path to config JSON file")
	flag.Parse()

	cfg := config.MustLoad(*cfgPath)
	logger := log.New(os.Stdout, "", log.LstdFlags)
	counters := metrics.NewCounters()
	startedAt := time.Now()
	logger.Printf("loaded config: %s (port %d)", cfg.Name, cfg.Port)

	mux := http.NewServeMux()

	mux.Handle("/hello", middleware.AllowMethods([]string{http.MethodGet},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			name := r.URL.Query().Get("name")
			if strings.TrimSpace(name) == "" {
				name = cfg.DefaultGreet
			}
			if err := validateHelloName(name); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			style := r.URL.Query().Get("style")
			g := greeter.New(style)
			count := counters.Inc("hello_requests")
			message := g.Greet(name, int(count))

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			if err := json.NewEncoder(w).Encode(helloResponse{
				Message: message,
				Count:   count,
			}); err != nil {
				logger.Printf("encode /hello response: %v", err)
			}
		}),
	))

	mux.Handle("/health", middleware.AllowMethods([]string{http.MethodGet},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			counters.Inc("health_requests")
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok\n"))
		}),
	))

	mux.Handle("/metrics", middleware.AllowMethods([]string{http.MethodGet},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			counters.Inc("metrics_requests")
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte(counters.PlainText()))
		}),
	))

	handler := middleware.Chain(
		mux,
		func(h http.Handler) http.Handler { return middleware.Logger(logger, h) },
		func(h http.Handler) http.Handler { return middleware.Recover(logger, h) },
		func(h http.Handler) http.Handler { return middleware.RequestCounter(counters, h) },
	)

	server := &http.Server{
		Addr:              cfg.Addr(),
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

	logger.Printf("hello-server listening on http://%s", server.Addr)
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

func validateHelloName(name string) error {
	for _, ch := range strings.TrimSpace(name) {
		if ch == '<' || ch == '>' || ch == '&' {
			return apperror.NewFieldError("name", "contains unsafe characters", apperror.ErrValidation)
		}
	}
	return nil
}
