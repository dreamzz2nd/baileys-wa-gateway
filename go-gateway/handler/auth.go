package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"wa-gateway/config"
)

type AuthHandler struct {
	cfg *config.Config
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{cfg: cfg}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func generateToken(username string, secret string) string {
	exp := time.Now().Add(24 * 7 * time.Hour).Unix() // 7 hari
	data := fmt.Sprintf("%s:%d", username, exp)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	signature := hex.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("%s:%s", data, signature)
}

func VerifyToken(token string, secret string) bool {
	parts := strings.Split(token, ":")
	if len(parts) != 3 {
		return false
	}

	username := parts[0]
	expStr := parts[1]
	signature := parts[2]

	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false // Expired
	}

	data := fmt.Sprintf("%s:%d", username, exp)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(JSONResponse{
			Success: false,
			Message: "Format JSON request tidak valid",
		})
		return
	}

	if req.Username != h.cfg.AdminUsername || req.Password != h.cfg.AdminPassword {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(JSONResponse{
			Success: false,
			Message: "Username atau Password salah!",
		})
		return
	}

	token := generateToken(req.Username, h.cfg.AdminSecret)

	_ = json.NewEncoder(w).Encode(JSONResponse{
		Success: true,
		Message: "Login Admin berhasil",
		Data: map[string]interface{}{
			"token":    token,
			"username": req.Username,
			"api_key":  h.cfg.APIKey,
		},
	})
}

func (h *AuthHandler) CheckSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		token = r.Header.Get("x-admin-token")
	}

	if !VerifyToken(token, h.cfg.AdminSecret) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(JSONResponse{
			Success: false,
			Message: "Sesi admin kedaluwarsa atau tidak valid",
		})
		return
	}

	_ = json.NewEncoder(w).Encode(JSONResponse{
		Success: true,
		Message: "Sesi valid",
		Data: map[string]interface{}{
			"username": h.cfg.AdminUsername,
			"api_key":  h.cfg.APIKey,
		},
	})
}
