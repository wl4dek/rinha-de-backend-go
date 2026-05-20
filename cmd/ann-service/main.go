package main

import (
	"log"
	"net/http"
	"os"
	"time"

	ann "rinha-de-backend/internal/ann"
	"rinha-de-backend/internal/metrics"
)

func main() {
	port := os.Getenv("ANN_PORT")
	if port == "" {
		port = "8090"
	}

	ivfPath := os.Getenv("IVF_DATA_PATH")
	if ivfPath == "" {
		ivfPath = "./ivf_data"
	}

	log.Printf("Loading IVF index from %s", ivfPath)
	idx, err := ann.LoadIVF(ivfPath)
	if err != nil {
		log.Fatalf("Failed to load IVF index: %v", err)
	}

	ann.SetGraph(idx)

	mux := http.NewServeMux()
	mux.HandleFunc("/search", ann.HandleSearch)
	mux.HandleFunc("/ready", ann.HandleReady)
	mux.Handle("/metrics", metrics.Handler())
	handler := metrics.MetricsMiddleware(mux, "ann-service")

	log.Printf("Starting ANN service on port %s", port)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
