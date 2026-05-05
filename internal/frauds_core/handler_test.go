package fraudscore

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleFraudScore(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		body       interface{}
		wantStatus int
	}{
		{
			name:       "accepts POST method",
			method:     http.MethodPost,
			body:       map[string]interface{}{"amount": 100},
			wantStatus: http.StatusOK,
		},
		{
			name:       "rejects GET method",
			method:     http.MethodGet,
			body:       nil,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reqBody *bytes.Buffer
			if tt.body != nil {
				bodyBytes, _ := json.Marshal(tt.body)
				reqBody = bytes.NewBuffer(bodyBytes)
			} else {
				reqBody = &bytes.Buffer{}
			}

			req := httptest.NewRequest(tt.method, "/fraud-score", reqBody)
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			w := httptest.NewRecorder()
			HandleFraudScore(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("HandleFraudScore() status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}
