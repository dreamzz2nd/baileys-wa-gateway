package handler

import (
	"encoding/json"
	"net/http"
	"wa-gateway/service"
)

type MessageHandler struct {
	wa *service.WhatsAppService
}

func NewMessageHandler(wa *service.WhatsAppService) *MessageHandler {
	return &MessageHandler{wa: wa}
}

type SendTextRequest struct {
	Receiver string `json:"receiver"`
	Message  string `json:"message"`
}

type SendImageRequest struct {
	Receiver string `json:"receiver"`
	ImageURL string `json:"image_url"`
	Caption  string `json:"caption"`
}

type SendDocumentRequest struct {
	Receiver    string `json:"receiver"`
	DocumentURL string `json:"document_url"`
	Filename    string `json:"filename"`
	Caption     string `json:"caption"`
}

func (h *MessageHandler) SendText(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req SendTextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(JSONResponse{
			Success: false,
			Message: "JSON request body tidak valid",
		})
		return
	}

	if req.Receiver == "" || req.Message == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(JSONResponse{
			Success: false,
			Message: "Field 'receiver' dan 'message' wajib diisi",
		})
		return
	}

	resp, err := h.wa.SendTextMessage(r.Context(), req.Receiver, req.Message)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(JSONResponse{
			Success: false,
			Message: "Gagal mengirim pesan: " + err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(JSONResponse{
		Success: true,
		Message: "Pesan berhasil terkirim",
		Data: map[string]interface{}{
			"id":        resp.ID,
			"timestamp": resp.Timestamp.Unix(),
			"receiver":  req.Receiver,
		},
	})
}

func (h *MessageHandler) SendImage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req SendImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(JSONResponse{
			Success: false,
			Message: "JSON request body tidak valid",
		})
		return
	}

	if req.Receiver == "" || req.ImageURL == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(JSONResponse{
			Success: false,
			Message: "Field 'receiver' dan 'image_url' wajib diisi",
		})
		return
	}

	resp, err := h.wa.SendImageMessage(r.Context(), req.Receiver, req.ImageURL, req.Caption)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(JSONResponse{
			Success: false,
			Message: "Gagal mengirim pesan gambar: " + err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(JSONResponse{
		Success: true,
		Message: "Gambar berhasil terkirim",
		Data: map[string]interface{}{
			"id":        resp.ID,
			"timestamp": resp.Timestamp.Unix(),
			"receiver":  req.Receiver,
		},
	})
}

func (h *MessageHandler) SendDocument(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req SendDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(JSONResponse{
			Success: false,
			Message: "JSON request body tidak valid",
		})
		return
	}

	if req.Receiver == "" || req.DocumentURL == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(JSONResponse{
			Success: false,
			Message: "Field 'receiver' dan 'document_url' wajib diisi",
		})
		return
	}

	resp, err := h.wa.SendDocumentMessage(r.Context(), req.Receiver, req.DocumentURL, req.Filename, req.Caption)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(JSONResponse{
			Success: false,
			Message: "Gagal mengirim dokumen: " + err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(JSONResponse{
		Success: true,
		Message: "Dokumen berhasil terkirim",
		Data: map[string]interface{}{
			"id":        resp.ID,
			"timestamp": resp.Timestamp.Unix(),
			"receiver":  req.Receiver,
		},
	})
}
