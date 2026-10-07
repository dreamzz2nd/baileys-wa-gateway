package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"
)

type WhatsAppService struct {
	Client       *whatsmeow.Client
	Container    *sqlstore.Container
	Webhook      *WebhookService
	CurrentQR    string
	CurrentQRRaw string
	IsConnected  bool
	IsLoggedIn   bool
	mu           sync.RWMutex
	qrChan       <-chan whatsmeow.QRChannelItem
	cancelQR     context.CancelFunc
	dbPath       string
	logger       waLog.Logger
}

type SessionStatus struct {
	Connected  bool   `json:"connected"`
	LoggedIn   bool   `json:"logged_in"`
	JID        string `json:"jid,omitempty"`
	Phone      string `json:"phone,omitempty"`
	PushName   string `json:"push_name,omitempty"`
	HasQR      bool   `json:"has_qr"`
	WebhookURL string `json:"webhook_url,omitempty"`
}

func NewWhatsAppService(dbPath string, webhook *WebhookService, logLevel string) (*WhatsAppService, error) {
	ctx := context.Background()
	log := waLog.Stdout("Main", logLevel, true)
	dbLog := waLog.Stdout("Database", logLevel, true)

	container, err := sqlstore.New(ctx, "sqlite", "file:"+dbPath+"?_pragma=foreign_keys(1)", dbLog)
	if err != nil {
		return nil, fmt.Errorf("gagal membuka database sqlite: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal memuat device store: %w", err)
	}

	clientLog := waLog.Stdout("Client", logLevel, true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	service := &WhatsAppService{
		Client:    client,
		Container: container,
		Webhook:   webhook,
		dbPath:    dbPath,
		logger:    log,
	}

	client.AddEventHandler(service.handleEvents)

	return service, nil
}

func (s *WhatsAppService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Client.Store.ID == nil {
		// Device belum login, siapkan channel QR Code
		ctx, cancel := context.WithCancel(context.Background())
		s.cancelQR = cancel
		qrChan, err := s.Client.GetQRChannel(ctx)
		if err != nil {
			return fmt.Errorf("gagal menginisialisasi QR channel: %w", err)
		}
		s.qrChan = qrChan

		err = s.Client.Connect()
		if err != nil {
			return fmt.Errorf("gagal menghubungkan client: %w", err)
		}

		go s.listenQR(qrChan)
	} else {
		// Device sudah login sebelumnya
		err := s.Client.Connect()
		if err != nil {
			return fmt.Errorf("gagal menghubungkan client: %w", err)
		}
		s.IsConnected = true
		s.IsLoggedIn = true
		s.logger.Infof("Sesi tersimpan ditemukan, berhasil login sebagai: %s", s.Client.Store.ID.String())
	}

	return nil
}

func (s *WhatsAppService) listenQR(qrChan <-chan whatsmeow.QRChannelItem) {
	for item := range qrChan {
		if item.Event == "code" {
			pngData, err := qrcode.Encode(item.Code, qrcode.Medium, 256)
			if err == nil {
				base64QR := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngData)
				s.mu.Lock()
				s.CurrentQR = base64QR
				s.CurrentQRRaw = item.Code
				s.mu.Unlock()
				s.logger.Infof("QR Code baru berhasil digenerate, silahkan scan via Web UI.")
			}
		} else if item.Event == "success" {
			s.mu.Lock()
			s.CurrentQR = ""
			s.CurrentQRRaw = ""
			s.IsConnected = true
			s.IsLoggedIn = true
			s.mu.Unlock()
			s.logger.Infof("Scan QR Berhasil! WhatsApp terhubung.")
			s.Webhook.Send("session.connected", map[string]interface{}{
				"status": "connected",
				"jid":    s.Client.Store.ID.String(),
			})
			return
		} else if item.Event == "timeout" {
			s.logger.Warnf("QR Code kedaluwarsa (timeout). Menunggu QR baru...")
		}
	}
}

func (s *WhatsAppService) handleEvents(evt interface{}) {
	switch v := evt.(type) {
	case *events.Connected:
		s.mu.Lock()
		s.IsConnected = true
		if s.Client.Store.ID != nil {
			s.IsLoggedIn = true
		}
		s.mu.Unlock()
		s.logger.Infof("WhatsApp WebSocket Connected")

	case *events.Disconnected:
		s.mu.Lock()
		s.IsConnected = false
		s.mu.Unlock()
		s.logger.Warnf("WhatsApp WebSocket Terputus")

	case *events.LoggedOut:
		s.mu.Lock()
		s.IsLoggedIn = false
		s.IsConnected = false
		s.CurrentQR = ""
		s.mu.Unlock()
		s.logger.Warnf("Perangkat telah di-logout dari WhatsApp")
		s.Webhook.Send("session.logged_out", map[string]interface{}{
			"status": "logged_out",
			"reason": v.Reason.String(),
		})

	case *events.Message:
		text := ""
		msgType := "unknown"

		if v.Message.GetConversation() != "" {
			text = v.Message.GetConversation()
			msgType = "conversation"
		} else if v.Message.GetExtendedTextMessage() != nil {
			text = v.Message.GetExtendedTextMessage().GetText()
			msgType = "extended_text"
		} else if v.Message.GetImageMessage() != nil {
			text = v.Message.GetImageMessage().GetCaption()
			msgType = "image"
		} else if v.Message.GetVideoMessage() != nil {
			text = v.Message.GetVideoMessage().GetCaption()
			msgType = "video"
		} else if v.Message.GetDocumentMessage() != nil {
			text = v.Message.GetDocumentMessage().GetCaption()
			msgType = "document"
		}

		senderJID := v.Info.Sender.ToNonAD().String()
		chatJID := v.Info.Chat.String()
		isGroup := v.Info.IsGroup

		msgData := MessageData{
			ID:          v.Info.ID,
			From:        senderJID,
			SenderName:  v.Info.PushName,
			IsFromMe:    v.Info.IsFromMe,
			IsGroup:     isGroup,
			GroupJID:    chatJID,
			MessageType: msgType,
			Text:        text,
		}

		s.logger.Infof("[Pesan Masuk] Dari: %s | Teks: %s", senderJID, text)

		// Teruskan ke webhook
		s.Webhook.Send("message.upsert", msgData)
	}
}

func (s *WhatsAppService) FormatJID(target string) (types.JID, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return types.EmptyJID, errors.New("nomor penerima tidak boleh kosong")
	}

	// Jika sudah berakhiran @s.whatsapp.net atau @g.us
	if strings.Contains(target, "@") {
		return types.ParseJID(target)
	}

	// Bersihkan karakter non-digit (+, -, spasi, dll)
	var clean strings.Builder
	for _, r := range target {
		if r >= '0' && r <= '9' {
			clean.WriteRune(r)
		}
	}
	cleaned := clean.String()

	// Format nomor HP Indonesia (08... -> 628...)
	if strings.HasPrefix(cleaned, "08") {
		cleaned = "62" + cleaned[1:]
	} else if strings.HasPrefix(cleaned, "8") {
		cleaned = "62" + cleaned
	}

	return types.NewJID(cleaned, types.DefaultUserServer), nil
}

func (s *WhatsAppService) SendTextMessage(ctx context.Context, target string, message string) (*whatsmeow.SendResponse, error) {
	if !s.IsLoggedIn || !s.Client.IsConnected() {
		return nil, errors.New("whatsapp belum terhubung atau belum login")
	}

	jid, err := s.FormatJID(target)
	if err != nil {
		return nil, fmt.Errorf("format nomor tidak valid: %w", err)
	}

	msg := &waProto.Message{
		Conversation: proto.String(message),
	}

	resp, err := s.Client.SendMessage(ctx, jid, msg)
	if err != nil {
		return nil, fmt.Errorf("gagal mengirim pesan: %w", err)
	}

	return &resp, nil
}

func (s *WhatsAppService) SendImageMessage(ctx context.Context, target string, imageURL string, caption string) (*whatsmeow.SendResponse, error) {
	if !s.IsLoggedIn || !s.Client.IsConnected() {
		return nil, errors.New("whatsapp belum terhubung atau belum login")
	}

	jid, err := s.FormatJID(target)
	if err != nil {
		return nil, fmt.Errorf("format nomor tidak valid: %w", err)
	}

	// Download image bytes
	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat request gambar: %w", err)
	}
	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gagal mengunduh gambar dari URL: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status response unduh gambar: %d", httpResp.StatusCode)
	}

	data, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca data gambar: %w", err)
	}

	mimeType := httpResp.Header.Get("Content-Type")
	if mimeType == "" || !strings.HasPrefix(mimeType, "image/") {
		mimeType = "image/jpeg"
	}

	uploaded, err := s.Client.Upload(ctx, data, whatsmeow.MediaImage)
	if err != nil {
		return nil, fmt.Errorf("gagal upload media ke server WhatsApp: %w", err)
	}

	msg := &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			Caption:       proto.String(caption),
			Mimetype:      proto.String(mimeType),
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
		},
	}

	resp, err := s.Client.SendMessage(ctx, jid, msg)
	if err != nil {
		return nil, fmt.Errorf("gagal mengirim pesan gambar: %w", err)
	}

	return &resp, nil
}

func (s *WhatsAppService) SendDocumentMessage(ctx context.Context, target string, docURL string, filename string, caption string) (*whatsmeow.SendResponse, error) {
	if !s.IsLoggedIn || !s.Client.IsConnected() {
		return nil, errors.New("whatsapp belum terhubung atau belum login")
	}

	jid, err := s.FormatJID(target)
	if err != nil {
		return nil, fmt.Errorf("format nomor tidak valid: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", docURL, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat request dokumen: %w", err)
	}
	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gagal mengunduh dokumen dari URL: %w", err)
	}
	defer httpResp.Body.Close()

	data, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca data dokumen: %w", err)
	}

	mimeType := httpResp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	if filename == "" {
		filename = "document.pdf"
	}

	uploaded, err := s.Client.Upload(ctx, data, whatsmeow.MediaDocument)
	if err != nil {
		return nil, fmt.Errorf("gagal upload dokumen ke WhatsApp: %w", err)
	}

	msg := &waProto.Message{
		DocumentMessage: &waProto.DocumentMessage{
			Caption:       proto.String(caption),
			Title:         proto.String(filename),
			FileName:      proto.String(filename),
			Mimetype:      proto.String(mimeType),
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
		},
	}

	resp, err := s.Client.SendMessage(ctx, jid, msg)
	if err != nil {
		return nil, fmt.Errorf("gagal mengirim pesan dokumen: %w", err)
	}

	return &resp, nil
}

func (s *WhatsAppService) GetStatus() SessionStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := SessionStatus{
		Connected:  s.Client.IsConnected(),
		LoggedIn:   s.Client.IsLoggedIn(),
		HasQR:      s.CurrentQR != "",
		WebhookURL: s.Webhook.GetWebhookURL(),
	}

	if s.Client.Store.ID != nil {
		status.JID = s.Client.Store.ID.String()
		status.Phone = s.Client.Store.ID.User
		status.PushName = s.Client.Store.PushName
	}

	return status
}

func (s *WhatsAppService) Logout(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Client.IsLoggedIn() {
		err := s.Client.Logout(ctx)
		if err != nil {
			return err
		}
	}

	s.IsConnected = false
	s.IsLoggedIn = false
	s.CurrentQR = ""
	s.CurrentQRRaw = ""

	// Reset device store
	_ = s.Client.Store.Delete(ctx)

	// Mulai ulang listening QR
	if s.cancelQR != nil {
		s.cancelQR()
	}
	newCtx, cancel := context.WithCancel(context.Background())
	s.cancelQR = cancel
	qrChan, err := s.Client.GetQRChannel(newCtx)
	if err == nil {
		s.qrChan = qrChan
		_ = s.Client.Connect()
		go s.listenQR(qrChan)
	}

	return nil
}
