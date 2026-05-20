package fraudscore

import (
	"net/http"
	"time"

	"rinha-de-backend/internal/metrics"

	"github.com/bytedance/sonic"
)

var semaphore = make(chan struct{}, 80)

func HandleFraudScore(w http.ResponseWriter, r *http.Request) {
	semaphore <- struct{}{}
	defer func() { <-semaphore }()

	start := time.Now()
	defer func() {
		metrics.FraudRequestDuration.WithLabelValues("/fraud-score").Observe(time.Since(start).Seconds())
	}()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	var req FraudScoreRequest

	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(&req); err != nil {
		metrics.FraudRequestsTotal.WithLabelValues("400").Inc()
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	resp := CalculateFraudScore(req, DefaultRules)
	metrics.FraudScoreValue.Set(float64(resp.FraudScore))
	metrics.FraudRequestsTotal.WithLabelValues("200").Inc()

	w.Header().Set("Content-Type", "application/json")
	sonic.ConfigDefault.NewEncoder(w).Encode(resp)
}
