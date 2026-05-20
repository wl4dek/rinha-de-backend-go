package fraudscore

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"rinha-de-backend/internal/metrics"

	"github.com/bytedance/sonic"
)

func limitar(value, max float32) float32 {
	resultado := value / max

	if resultado < 0 {
		return 0
	}
	if resultado > 1 {
		return 1
	}

	return resultado
}

func CalculateFraudScore(req FraudScoreRequest, rules Rules) FraudScoreResponse {
	score := 0.0

	var vector [14]float32
	vector[0] = limitar(req.Transaction.Amount, rules.MaxAmount)
	vector[1] = limitar(float32(req.Transaction.Installments), float32(rules.MaxInstallments))
	vector[2] = limitar(req.Transaction.Amount/req.Customer.AvgAmount, rules.AmountVsAvgRatio)
	vector[3] = limitar(float32(req.Transaction.RequestedAt.Hour()), float32(rules.MaxHour))
	vector[4] = limitar(float32(req.Transaction.RequestedAt.Weekday()), 6)

	if req.LastTx != nil {
		vector[5] = limitar(float32(req.LastTx.Timestamp.Minute()), float32(rules.MaxMinutes))
		vector[6] = limitar(req.LastTx.KmFromCurrent, rules.MaxKm)
	} else {
		vector[5] = -1
		vector[6] = -1
	}

	vector[7] = limitar(req.Terminal.KmFromHome, rules.MaxKm)
	vector[8] = limitar(float32(req.Customer.TxCount24h), float32(rules.MaxTxCount24h))

	if req.Terminal.IsOnline {
		vector[9] = 1
	} else {
		vector[9] = 0
	}

	if req.Terminal.CardPresent {
		vector[10] = 1
	} else {
		vector[10] = 0
	}

	known := false
	for _, m := range req.Customer.KnownMerchants {
		if m == req.Merchant.ID {
			known = true
			break
		}
	}
	if known {
		vector[11] = 1
	} else {
		vector[11] = 0
	}

	weight, ok := MCCRisk[req.Merchant.MCC]
	if !ok {
		weight = 0.5
	}
	vector[12] = weight

	vector[13] = limitar(req.Merchant.AvgAmount, rules.MaxMerchantAvgAmount)

	annScore := annSearch(vector[:])
	score = float64(annScore)

	approved := score < 0.6

	return FraudScoreResponse{
		Approved:   approved,
		FraudScore: float32(score),
	}
}

type annSearchRequest struct {
	Vector []float32 `json:"vector"`
	K      int       `json:"k"`
}

type annSearchResult struct {
	Label string `json:"label"`
}

type annSearchResponse struct {
	Results []annSearchResult `json:"results"`
}

var annSearch func([]float32) float32

func init() {
	annSearch = queryANNHTTP
}

func SetANNSearch(fn func([]float32) float32) {
	annSearch = fn
}

var annHTTPClient = &http.Client{
	Timeout: 2 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true,
	},
}

func queryANNHTTP(vector []float32) float32 {
	annURL := os.Getenv("ANN_SERVICE_URL")
	if annURL == "" {
		annURL = "http://localhost:8090"
	}

	start := time.Now()
	defer func() {
		metrics.AnnClientDuration.Observe(time.Since(start).Seconds())
	}()

	req := annSearchRequest{Vector: vector, K: 5}
	body, _ := sonic.ConfigDefault.Marshal(req)
	resp, err := annHTTPClient.Post(annURL+"/search", "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("ANN service error: %v", err)
		metrics.AnnClientRequestsTotal.WithLabelValues("error").Inc()
		return 0.0
	}
	defer resp.Body.Close()

	var searchResp annSearchResponse
	if err := sonic.ConfigDefault.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		log.Printf("Failed to decode ANN response: %v", err)
		metrics.AnnClientRequestsTotal.WithLabelValues("error").Inc()
		return 0.0
	}

	if len(searchResp.Results) == 0 {
		metrics.AnnClientRequestsTotal.WithLabelValues("ok").Inc()
		return 0.0
	}

	fraudCount := 0
	for _, r := range searchResp.Results {
		if strings.EqualFold(r.Label, "fraud") {
			fraudCount++
		}
	}

	metrics.AnnClientRequestsTotal.WithLabelValues("ok").Inc()
	return float32(fraudCount) / float32(len(searchResp.Results))
}
