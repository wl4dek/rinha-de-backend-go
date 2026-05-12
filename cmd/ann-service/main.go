package main

import (
	"log"
	"net/http"
	"os"

	ann "rinha-de-backend/internal/ann"
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

	log.Printf("Starting ANN service on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
