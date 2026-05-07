package ann

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/coder/hnsw"
)

const Dimensions = 14

// LoadOrBuild loads HNSW index from disk or builds a new one from references
func LoadOrBuild(refPath, indexBinPath string) (*hnsw.Graph[int], []Reference, error) {
	// Try to load existing index
	if _, err := os.Stat(indexBinPath); err == nil {
		log.Printf("Loading HNSW index from %s", indexBinPath)
		g, err := loadIndex(indexBinPath)
		if err == nil {
			refs, err := loadReferences(refPath)
			if err == nil {
				return g, refs, nil
			}
			log.Printf("Failed to load references, rebuilding: %v", err)
		} else {
			log.Printf("Failed to load index, rebuilding: %v", err)
		}
	}

	// Build new index
	log.Printf("Building HNSW index from %s (optimized build)", refPath)
	refs, err := loadReferences(refPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load references: %w", err)
	}

	g, err := buildHNSW(refs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build HNSW: %w", err)
	}

	if err := saveIndex(g, indexBinPath); err != nil {
		return nil, nil, fmt.Errorf("failed to save index: %w", err)
	}

	return g, refs, nil
}

// buildHNSW builds a new HNSW index from references
func buildHNSW(refs []Reference) (*hnsw.Graph[int], error) {
	g := hnsw.NewGraph[int]()

	// Optimize build speed: reduce M (fewer neighbors per node = faster insertion)
	// Default M=16, we use 12 for ~25% faster build with slight recall tradeoff
	g.M = 12
	g.EfSearch = 64 // Lower efSearch for faster search during construction

	batchSize := 200000
	items := make([]hnsw.Node[int], 0, batchSize)

	for i, ref := range refs {
		if len(ref.Vector) != Dimensions {
			return nil, fmt.Errorf("reference %d has %d dimensions, expected %d", i, len(ref.Vector), Dimensions)
		}

		items = append(items, hnsw.MakeNode(i, ref.Vector))

		if len(items) >= batchSize || i == len(refs)-1 {
			g.Add(items...)
			items = items[:0]
			log.Printf("Inserted %d/%d vectors (%.1f%%)", i+1, len(refs), float64(i+1)/float64(len(refs))*100)
		}
	}
	log.Printf("Finished building HNSW index with %d vectors (M=%d, EfSearch=%d)", len(refs), g.M, g.EfSearch)
	return g, nil
}

// saveIndex saves the HNSW graph using g.Export
func saveIndex(g *hnsw.Graph[int], path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Wrap with bufio for efficiency
	bw := bufio.NewWriter(f)
	defer bw.Flush()

	if err := g.Export(bw); err != nil {
		return fmt.Errorf("failed to export graph: %w", err)
	}
	log.Printf("Saved HNSW index to %s", path)
	return nil
}

// loadIndex loads the HNSW graph using g.Import
func loadIndex(path string) (*hnsw.Graph[int], error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Wrap with bufio to satisfy io.ByteReader requirement
	br := bufio.NewReader(f)
	g := hnsw.NewGraph[int]()
	if err := g.Import(br); err != nil {
		return nil, fmt.Errorf("failed to import graph: %w", err)
	}
	log.Printf("Loaded HNSW index from %s", path)
	return g, nil
}

// loadReferences loads reference vectors from a gzipped JSON file
func loadReferences(path string) ([]Reference, error) {
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

	var refs []Reference
	if err := json.NewDecoder(gr).Decode(&refs); err != nil {
		return nil, err
	}

	return refs, nil
}

// searchHit represents a search result with dist and label
type searchHit struct {
	Dist  float32
	Label string
}

// l2Distance calculates the L2 (squared Euclidean) dist between two vectors
func l2Distance(a, b []float32) float32 {
	var sum float32
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return sum
}

// Search performs a KNN search on the HNSW graph
func Search(g *hnsw.Graph[int], vector []float32, k int, refs []Reference) ([]searchHit, error) {
	results := g.Search(vector, k)
	hits := make([]searchHit, len(results))
	for i, r := range results {
		dist := l2Distance(vector, r.Value)
		label := "legit"
		if r.Key < len(refs) {
			label = refs[r.Key].Label
		}
		hits[i] = searchHit{Dist: dist, Label: label}
	}
	return hits, nil
}

// SaveIndexToWriter saves the HNSW graph to an io.Writer (for testing)
func SaveIndexToWriter(g *hnsw.Graph[int], w io.Writer) error {
	return g.Export(w)
}

// LoadIndexFromReader loads the HNSW graph from an io.Reader (for testing)
func LoadIndexFromReader(r io.Reader) (*hnsw.Graph[int], error) {
	g := hnsw.NewGraph[int]()
	if err := g.Import(r); err != nil {
		return nil, err
	}
	return g, nil
}
