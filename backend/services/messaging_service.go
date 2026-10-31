package services

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"strings"
	"time"

	"investment-tracker/config"
	"investment-tracker/models"
	"investment-tracker/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// MessagingService delivers reminder digests by email (SMTP) and Telegram.
// Both channels are optional and enabled only when the server is configured.
type MessagingService struct {
	cfg       *config.Config
	repo      *repository.NotificationSettingsRepo
	notifRepo *repository.NotificationRepo
	userRepo  *repository.UserRepo
	client    *http.Client
}

func NewMessagingService(cfg *config.Config, repo *repository.NotificationSettingsRepo, notifRepo *repository.NotificationRepo, userRepo *repository.UserRepo) *MessagingService {
	return &MessagingService{cfg: cfg, repo: repo, notifRepo: notifRepo, userRepo: userRepo, client: &http.Client{Timeout: 15 * time.Second}}
}

func (s *MessagingService) EmailAvailable() bool {
	return s.cfg.SMTPHost != "" && s.cfg.SMTPFrom != ""
}

func (s *MessagingService) TelegramAvailable() bool { return s.cfg.TelegramBotToken != "" }

func stripCRLF(v string) string { return strings.NewReplacer("\r", "", "\n", " ").Replace(v) }

func (s *MessagingService) SendEmail(to, subject, body string) error {
	if !s.EmailAvailable() {
		return errors.New("email is not configured on the server")
	}
	if _, err := mail.ParseAddress(to); err != nil {
		return invalidf("invalid email address")
	}
	from := s.cfg.SMTPFrom
	fromAddr := from
	if a, err := mail.ParseAddress(from); err == nil {
		fromAddr = a.Address
	}
	msg := "From: " + stripCRLF(from) + "\r\n" +
		"To: " + stripCRLF(to) + "\r\n" +
		"Subject: " + stripCRLF(subject) + "\r\n" +
		"MIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" +
		strings.ReplaceAll(body, "\n", "\r\n")

	addr := net.JoinHostPort(s.cfg.SMTPHost, s.cfg.SMTPPort)
	var auth smtp.Auth
	if s.cfg.SMTPUsername != "" {
		auth = smtp.PlainAuth("", s.cfg.SMTPUsername, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	}
	if s.cfg.SMTPPort != "465" {
		// smtp.SendMail upgrades with STARTTLS when the server offers it.
		return smtp.SendMail(addr, auth, fromAddr, []string{to}, []byte(msg))
	}
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 15 * time.Second}, "tcp", addr, &tls.Config{ServerName: s.cfg.SMTPHost})
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, s.cfg.SMTPHost)
	if err != nil {
		conn.Close()
		return err
	}
	defer c.Close()
	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return err
		}
	}
	if err := c.Mail(fromAddr); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

func (s *MessagingService) SendTelegram(chatID, text string) error {
	if !s.TelegramAvailable() {
		return errors.New("telegram is not configured on the server")
	}
	if strings.TrimSpace(chatID) == "" {
		return invalidf("telegram_chat_id is required")
	}
	endpoint := "https://api.telegram.org/bot" + s.cfg.TelegramBotToken + "/sendMessage"
	resp, err := s.client.PostForm(endpoint, url.Values{"chat_id": {chatID}, "text": {text}, "disable_web_page_preview": {"true"}})
	if err != nil {
		return errors.New("could not reach Telegram")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var body struct {
			Description string `json:"description"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&body)
		return fmt.Errorf("telegram rejected the message: %s", body.Description)
	}
	return nil
}

func (s *MessagingService) GetSettings(ctx context.Context, userID string) (*models.NotificationSettings, error) {
	st, err := s.repo.FindOne(ctx, bson.M{"user_id": userID})
	if errors.Is(err, mongo.ErrNoDocuments) {
		st = &models.NotificationSettings{UserID: userID}
		if oid, err := primitive.ObjectIDFromHex(userID); err == nil {
			if u, err := s.userRepo.GetByID(ctx, oid); err == nil {
				st.Email = u.Email
			}
		}
	} else if err != nil {
		return nil, err
	}
	st.EmailAvailable, st.TelegramAvailable = s.EmailAvailable(), s.TelegramAvailable()
	return st, nil
}

func (s *MessagingService) UpdateSettings(ctx context.Context, userID string, req models.NotificationSettingsRequest) (*models.NotificationSettings, error) {
	req.Email = strings.TrimSpace(req.Email)
	req.TelegramChatID = strings.TrimSpace(req.TelegramChatID)
	if req.Email != "" {
		if _, err := mail.ParseAddress(req.Email); err != nil {
			return nil, invalidf("invalid email address")
		}
	}
	if req.EmailEnabled && req.Email == "" {
		return nil, invalidf("email is required to enable email reminders")
	}
	if req.TelegramEnabled && req.TelegramChatID == "" {
		return nil, invalidf("telegram_chat_id is required to enable Telegram reminders")
	}
	err := s.repo.Upsert(ctx, bson.M{"user_id": userID}, bson.M{
		"user_id": userID, "email_enabled": req.EmailEnabled, "email": req.Email,
		"telegram_enabled": req.TelegramEnabled, "telegram_chat_id": req.TelegramChatID,
	})
	if err != nil {
		return nil, err
	}
	return s.GetSettings(ctx, userID)
}

// SendTest sends a test message on every enabled channel.
func (s *MessagingService) SendTest(ctx context.Context, userID string) (map[string]string, error) {
	st, err := s.GetSettings(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !st.EmailEnabled && !st.TelegramEnabled {
		return nil, invalidf("enable email or Telegram reminders first")
	}
	result := map[string]string{}
	text := "Test from Investment Tracker: reminders will arrive here."
	if st.EmailEnabled {
		result["email"] = "sent"
		if err := s.SendEmail(st.Email, "Investment Tracker test", text); err != nil {
			result["email"] = err.Error()
		}
	}
	if st.TelegramEnabled {
		result["telegram"] = "sent"
		if err := s.SendTelegram(st.TelegramChatID, text); err != nil {
			result["telegram"] = err.Error()
		}
	}
	return result, nil
}

// DispatchPending sends one digest of recent unread reminders not yet sent.
func (s *MessagingService) DispatchPending(ctx context.Context, userID string) (int, error) {
	if !s.EmailAvailable() && !s.TelegramAvailable() {
		return 0, nil
	}
	st, err := s.repo.FindOne(ctx, bson.M{"user_id": userID})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return 0, nil
		}
		return 0, err
	}
	email := st.EmailEnabled && s.EmailAvailable()
	telegram := st.TelegramEnabled && s.TelegramAvailable()
	if !email && !telegram {
		return 0, nil
	}
	unread, err := s.notifRepo.GetUnread(ctx, userID)
	if err != nil {
		return 0, err
	}
	since := time.Now().Add(-48 * time.Hour)
	if st.LastDispatchAt != nil && st.LastDispatchAt.After(since) {
		since = *st.LastDispatchAt
	}
	var lines []string
	for _, n := range unread {
		if n.CreatedAt.After(since) {
			lines = append(lines, "• "+n.Title+": "+n.Message)
		}
	}
	if len(lines) == 0 {
		return 0, nil
	}
	body := "Your Investment Tracker reminders:\n\n" + strings.Join(lines, "\n")
	var errs []string
	if email {
		if err := s.SendEmail(st.Email, fmt.Sprintf("%d reminder(s) from Investment Tracker", len(lines)), body); err != nil {
			errs = append(errs, "email: "+err.Error())
		}
	}
	if telegram {
		if err := s.SendTelegram(st.TelegramChatID, body); err != nil {
			errs = append(errs, "telegram: "+err.Error())
		}
	}
	if err := s.repo.Set(ctx, st.ID, bson.M{"last_dispatch_at": time.Now()}); err != nil {
		return 0, err
	}
	if len(errs) > 0 {
		return len(lines), errors.New(strings.Join(errs, "; "))
	}
	return len(lines), nil
}
