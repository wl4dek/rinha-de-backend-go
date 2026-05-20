package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	fraudscore "rinha-de-backend/internal/frauds_core"
	"rinha-de-backend/internal/metrics"
	ready "rinha-de-backend/internal/ready"
	"runtime"
	"strconv"
	"time"
)

type Config struct {
	Port         string
	PprofEnabled bool
	IVFDataPath  string
}

func loadConfig() Config {
	config := Config{
		Port:         getEnv("PORT", "8080"),
		PprofEnabled: os.Getenv("PPROF_ENABLED") == "true",
		IVFDataPath:  getEnv("IVF_DATA_PATH", "./ivf_data"),
	}
	return config
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/ready", ready.HandleReady)
	mux.HandleFunc("/fraud-score", fraudscore.HandleFraudScore)
	mux.Handle("/metrics", metrics.Handler())

	return mux
}

func main() {
	runtime.GOMAXPROCS(1)
	config := loadConfig()

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	mux := setupRoutes()
	handler := metrics.MetricsMiddleware(mux, "server")

	log.Printf("Starting server on port %s", config.Port)

	srv := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("server error: %v", err)
		}
	}()

	if config.PprofEnabled {
		pprofSrv := &http.Server{
			Addr:         "0.0.0.0:6060",
			Handler:      http.DefaultServeMux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		go func() {
			log.Println("pprof on :6060")
			if err := pprofSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("pprof error: %v", err)
			}
		}()
	}

	select {}
}
