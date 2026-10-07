package middleware

import (
	"encoding/json"
	"net/http"
	"strings"
	"wa-gateway/handler"
)

type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func AuthMiddleware(apiKey string, adminSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Cek header x-api-key (Untuk API Client / Web Presensi)
			reqKey := r.Header.Get("x-api-key")
			if reqKey == "" {
				reqKey = r.URL.Query().Get("api_key")
			}
			if apiKey != "" && reqKey == apiKey {
				next.ServeHTTP(w, r)
				return
			}

			// 2. Cek Admin Session Token (Untuk Dashboard Web UI)
			adminToken := r.Header.Get("x-admin-token")
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				adminToken = strings.TrimPrefix(authHeader, "Bearer ")
			}
			if adminToken != "" && handler.VerifyToken(adminToken, adminSecret) {
				next.ServeHTTP(w, r)
				return
			}

			// Jika keduanya tidak valid
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(ErrorResponse{
				Success: false,
				Message: "Unauthorized: Silahkan sertakan 'x-api-key' yang valid atau login sebagai admin",
			})
		})
	}
}
