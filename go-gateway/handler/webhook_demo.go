package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type WebhookLog struct {
	Timestamp string      `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

type WebhookDemoHandler struct {
	mu   sync.RWMutex
	logs []WebhookLog
}

func NewWebhookDemoHandler() *WebhookDemoHandler {
	return &WebhookDemoHandler{
		logs: make([]WebhookLog, 0),
	}
}

func (h *WebhookDemoHandler) ReceiveWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Gagal membaca body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		payload = string(body)
	}

	h.mu.Lock()
	h.logs = append([]WebhookLog{{
		Timestamp: time.Now().Format("15:04:05"),
		Payload:   payload,
	}}, h.logs...)
	if len(h.logs) > 50 {
		h.logs = h.logs[:50]
	}
	h.mu.Unlock()

	fmt.Printf("[Demo Webhook Receiver] Menerima event: %s\n", string(body))

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Webhook received successfully",
	})
}

func (h *WebhookDemoHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	h.mu.RLock()
	defer h.mu.RUnlock()

	_ = json.NewEncoder(w).Encode(JSONResponse{
		Success: true,
		Data:    h.logs,
	})
}
