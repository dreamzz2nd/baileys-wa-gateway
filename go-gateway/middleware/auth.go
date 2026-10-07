package middleware

import (
	"encoding/json"
	"net/http"
	"strings"
)

type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func AuthMiddleware(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Jika API_KEY kosong di config, bypass auth
			if apiKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Cek header x-api-key
			reqKey := r.Header.Get("x-api-key")

			// Cek header Authorization: Bearer <key>
			if reqKey == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					reqKey = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}

			// Cek query param ?api_key=... (berguna untuk testing browser/image preview)
			if reqKey == "" {
				reqKey = r.URL.Query().Get("api_key")
			}

			if reqKey != apiKey {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(ErrorResponse{
					Success: false,
					Message: "Unauthorized: API Key tidak valid atau tidak disertakan pada header 'x-api-key'",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
