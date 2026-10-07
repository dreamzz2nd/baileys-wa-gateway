package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wa-gateway/config"
	"wa-gateway/handler"
	appMiddleware "wa-gateway/middleware"
	"wa-gateway/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	cfg := config.LoadConfig()

	fmt.Println("==================================================")
	fmt.Println("       🚀 WHATSAPP GATEWAY (GOLANG + WHATSMEOW)   ")
	fmt.Println("==================================================")
	fmt.Printf(" Port       : %s\n", cfg.Port)
	fmt.Printf(" API Key    : %s\n", cfg.APIKey)
	fmt.Printf(" Webhook    : %s\n", cfg.WebhookURL)
	fmt.Printf(" SQLite DB  : %s\n", cfg.DBPath)
	fmt.Println("==================================================")

	// Inisialisasi Webhook Service
	webhookSvc := service.NewWebhookService(cfg.WebhookURL)

	// Inisialisasi WhatsApp Service
	waSvc, err := service.NewWhatsAppService(cfg.DBPath, webhookSvc, cfg.LogLevel)
	if err != nil {
		fmt.Printf("❌ Gagal menginisialisasi WhatsApp Service: %v\n", err)
		os.Exit(1)
	}

	// Mulai WhatsApp Service (Connect / QR Listener)
	err = waSvc.Start()
	if err != nil {
		fmt.Printf("❌ Gagal memulai koneksi WhatsApp: %v\n", err)
		os.Exit(1)
	}

	// Inisialisasi Handlers
	sessionHandler := handler.NewSessionHandler(waSvc)
	messageHandler := handler.NewMessageHandler(waSvc)
	demoWebhookHandler := handler.NewWebhookDemoHandler()

	// Router Setup
	r := chi.NewRouter()

	// Global Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "x-api-key"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Public Routes (Status, QR, Demo Webhook)
	r.Get("/api/session/status", sessionHandler.GetStatus)
	r.Get("/api/session/qr", sessionHandler.GetQR)
	r.Post("/webhook/test", demoWebhookHandler.ReceiveWebhook)
	r.Get("/webhook/logs", demoWebhookHandler.GetLogs)

	// Protected API Routes (Membutuhkan x-api-key)
	r.Group(func(protected chi.Router) {
		protected.Use(appMiddleware.AuthMiddleware(cfg.APIKey))

		// Send Messages
		protected.Post("/api/send/text", messageHandler.SendText)
		protected.Post("/api/send/image", messageHandler.SendImage)
		protected.Post("/api/send/document", messageHandler.SendDocument)

		// Session Control
		protected.Post("/api/session/logout", sessionHandler.Logout)
		protected.Post("/api/session/webhook", sessionHandler.UpdateWebhook)
	})

	// Static Files (Web UI Dashboard)
	workDir, _ := os.Getwd()
	publicDir := http.Dir(workDir + "/public")
	r.Handle("/*", http.FileServer(publicDir))

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// Jalankan server HTTP di goroutine
	go func() {
		fmt.Printf("🌐 Server berjalan di http://localhost:%s\n", cfg.Port)
		fmt.Printf("📱 Buka http://localhost:%s di browser untuk Scan QR & Dashboard\n\n", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("❌ HTTP server error: %v\n", err)
		}
	}()

	// Graceful Shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("\n🛑 Menghentikan server dan memutuskan koneksi WhatsApp...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = server.Shutdown(ctx)
	waSvc.Client.Disconnect()
	fmt.Println("✅ Server WhatsApp Gateway berhasil dihentikan dengan aman.")
}
