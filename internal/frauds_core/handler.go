package fraudscore

import (
	"encoding/json"
	"net/http"
)

func HandleFraudScore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	var req FraudScoreRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	resp := CalculateFraudScore(req, DefaultRules)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
