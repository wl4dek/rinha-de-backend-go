package ann

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/coder/hnsw"
)

var graph *hnsw.Graph[int]
var references []Reference

func SetGraph(g *hnsw.Graph[int], refs []Reference) {
	graph = g
	references = refs
}

func HandleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.K == 0 {
		req.K = 5
	}

	hits, err := Search(graph, req.Vector, req.K, references)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Search received: vector=%v, k=%d, results=%v", req.Vector, req.K, hits)
	out := make([]SearchResult, 0, len(hits))
	for _, hit := range hits {
		out = append(out, SearchResult{
			Label: hit.Label,
			Dist:  hit.Dist,
		})
	}

	resp := SearchResponse{Results: out}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func HandleReady(w http.ResponseWriter, r *http.Request) {
	if graph == nil {
		http.Error(w, "Not ready", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}
