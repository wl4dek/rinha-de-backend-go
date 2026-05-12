package ann

import (
	"net/http"

	"github.com/bytedance/sonic"
)

var index *IVFIndex

func SetGraph(idx *IVFIndex) {
	index = idx
}

func HandleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	var req SearchRequest
	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if len(req.Vector) != Dimensions {
		http.Error(w, "vector must have 14 dimensions", http.StatusBadRequest)
		return
	}
	if req.K < 1 {
		req.K = 5
	}

	hits, err := index.Search(req.Vector, req.K)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out := make([]SearchResult, len(hits))
	for i, hit := range hits {
		out[i] = SearchResult{Label: hit.Label, Dist: hit.Dist}
	}

	resp := SearchResponse{Results: out}
	w.Header().Set("Content-Type", "application/json")
	sonic.ConfigDefault.NewEncoder(w).Encode(resp)
}

func HandleReady(w http.ResponseWriter, r *http.Request) {
	if index == nil {
		http.Error(w, "Not ready", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}
