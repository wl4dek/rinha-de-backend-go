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

	refPath := os.Getenv("REFERENCES_PATH")
	if refPath == "" {
		refPath = "./references.json.gz"
	}

	indexBinPath := os.Getenv("INDEX_BIN_PATH")
	if indexBinPath == "" {
		indexBinPath = "./index.bin"
	}

	g, refs, err := ann.LoadOrBuild(refPath, indexBinPath)
	if err != nil {
		log.Fatalf("Failed to load/build HNSW: %v", err)
	}

	ann.SetGraph(g, refs)

	mux := http.NewServeMux()
	mux.HandleFunc("/search", ann.HandleSearch)
	mux.HandleFunc("/ready", ann.HandleReady)

	log.Printf("Starting ANN service on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
