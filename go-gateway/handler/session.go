package handler

import (
	"encoding/json"
	"net/http"
	"wa-gateway/service"
)

type SessionHandler struct {
	wa *service.WhatsAppService
}

func NewSessionHandler(wa *service.WhatsAppService) *SessionHandler {
	return &SessionHandler{wa: wa}
}

type JSONResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func (h *SessionHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status := h.wa.GetStatus()
	_ = json.NewEncoder(w).Encode(JSONResponse{
		Success: true,
		Data:    status,
	})
}

func (h *SessionHandler) GetQR(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status := h.wa.GetStatus()

	if status.LoggedIn {
		_ = json.NewEncoder(w).Encode(JSONResponse{
			Success: true,
			Message: "WhatsApp sudah terhubung / login",
			Data: map[string]interface{}{
				"logged_in": true,
				"phone":     status.Phone,
			},
		})
		return
	}

	if h.wa.CurrentQR == "" {
		_ = json.NewEncoder(w).Encode(JSONResponse{
			Success: false,
			Message: "QR Code belum siap atau sedang digenerate. Silahkan coba 2 detik lagi.",
		})
		return
	}

	_ = json.NewEncoder(w).Encode(JSONResponse{
		Success: true,
		Data: map[string]interface{}{
			"qr":  h.wa.CurrentQR,
			"raw": h.wa.CurrentQRRaw,
		},
	})
}

func (h *SessionHandler) Logout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	err := h.wa.Logout(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(JSONResponse{
			Success: false,
			Message: "Gagal melakukan logout: " + err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(JSONResponse{
		Success: true,
		Message: "Berhasil logout. Sesi telah direset dan QR Code baru sedang disiapkan.",
	})
}

type UpdateWebhookRequest struct {
	WebhookURL string `json:"webhook_url"`
}

func (h *SessionHandler) UpdateWebhook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req UpdateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(JSONResponse{
			Success: false,
			Message: "Request body tidak valid",
		})
		return
	}

	h.wa.Webhook.SetWebhookURL(req.WebhookURL)

	_ = json.NewEncoder(w).Encode(JSONResponse{
		Success: true,
		Message: "Webhook URL berhasil diperbarui ke: " + req.WebhookURL,
	})
}
