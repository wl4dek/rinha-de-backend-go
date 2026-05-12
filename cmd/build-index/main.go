package main

import (
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	ann "rinha-de-backend/internal/ann"
)

func main() {
	refsPath := flag.String("references", "./references.json.gz", "path to references.json.gz")
	output := flag.String("output", "./ivf_data", "output directory for IVF index files")
	clusters := flag.Int("clusters", 500, "number of IVF clusters")
	flag.Parse()

	log.Printf("Loading references from %s", *refsPath)
	refs, err := loadReferences(*refsPath)
	if err != nil {
		log.Fatalf("Failed to load references: %v", err)
	}
	log.Printf("Loaded %d references", len(refs))

	if len(refs) < *clusters {
		*clusters = len(refs)
		log.Printf("Reduced clusters to %d (less than number of references)", *clusters)
	}

	if err := os.MkdirAll(*output, 0755); err != nil {
		log.Fatalf("Failed to create output dir: %v", err)
	}

	if err := ann.BuildIVF(refs, *clusters, *output); err != nil {
		log.Fatalf("Failed to build IVF index: %v", err)
	}

	totalBytes, err := dirSize(*output)
	if err == nil {
		log.Printf("Total index size: %d MB", totalBytes/1024/1024)
	}

	fmt.Println(filepath.Clean(*output))
}

func loadReferences(path string) ([]ann.Reference, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	var refs []ann.Reference
	if err := json.NewDecoder(gr).Decode(&refs); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return refs, nil
}

func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}
