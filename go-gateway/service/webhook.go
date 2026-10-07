package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type WebhookPayload struct {
	Event     string      `json:"event"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data"`
}

type MessageData struct {
	ID          string `json:"id"`
	From        string `json:"from"`
	SenderName  string `json:"sender_name"`
	IsFromMe    bool   `json:"is_from_me"`
	IsGroup     bool   `json:"is_group"`
	GroupJID    string `json:"group_jid,omitempty"`
	MessageType string `json:"message_type"`
	Text        string `json:"text"`
}

type WebhookService struct {
	client     *http.Client
	webhookURL string
}

func NewWebhookService(webhookURL string) *WebhookService {
	return &WebhookService{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		webhookURL: webhookURL,
	}
}

func (w *WebhookService) SetWebhookURL(url string) {
	w.webhookURL = url
}

func (w *WebhookService) GetWebhookURL() string {
	return w.webhookURL
}

func (w *WebhookService) Send(event string, data interface{}) {
	if w.webhookURL == "" {
		return
	}

	payload := WebhookPayload{
		Event:     event,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}

	go func() {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			fmt.Printf("[Webhook] Error marshalling json: %v\n", err)
			return
		}

		req, err := http.NewRequest("POST", w.webhookURL, bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("[Webhook] Error creating request: %v\n", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Go-WhatsApp-Gateway/1.0")

		resp, err := w.client.Do(req)
		if err != nil {
			fmt.Printf("[Webhook] Gagal mengirim webhook ke %s: %v\n", w.webhookURL, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// Webhook success
			fmt.Printf("[Webhook] Berhasil meneruskan event '%s' ke %s (HTTP %d)\n", event, w.webhookURL, resp.StatusCode)
		} else {
			fmt.Printf("[Webhook] Webhook endpoint merespon dengan status: %d\n", resp.StatusCode)
		}
	}()
}
