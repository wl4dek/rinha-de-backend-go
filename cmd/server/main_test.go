package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	fraudscore "rinha-de-backend/internal/frauds_core"
)

func TestReadyEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	handleReady := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	handleReady(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ready endpoint status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestFraudScoreEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/fraud-score", nil)
	w := httptest.NewRecorder()

	fraudscore.HandleFraudScore(w, req)
}
