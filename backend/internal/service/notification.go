package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"time"

	"blackgrid/internal/db"
	"blackgrid/internal/events"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	ChannelTypeWebhook = "webhook"
	ChannelTypeSMTP    = "smtp"

	EventIncidentOpened   = "incident.opened"
	EventIncidentResolved = "incident.resolved"
	EventTest             = "test"

	maskedSecret = "********"

	defaultNotificationTimeout = 10 * time.Second
)

var ErrChannelNotFound = errors.New("notification channel not found")

type WebhookConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Timeout int               `json:"timeout_seconds,omitempty"`
}

type SMTPConfig struct {
	Host     string   `json:"host"`
	Port     int      `json:"port"`
	Username string   `json:"username,omitempty"`
	Password string   `json:"password,omitempty"`
	From     string   `json:"from"`
	To       []string `json:"to"`
	UseTLS   bool     `json:"use_tls"`
}

type NotificationService struct {
	q          *db.Queries
	httpClient *http.Client
	smtpSender smtpSender // pluggable for tests
	bus        *events.EventBus
}

type smtpSender func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error

func NewNotificationService(q *db.Queries, bus *events.EventBus) *NotificationService {
	return &NotificationService{
		q:   q,
		bus: bus,
		httpClient: &http.Client{
			Timeout: defaultNotificationTimeout,
		},
		smtpSender: smtp.SendMail,
	}
}

// SetHTTPClient overrides the HTTP client (used in tests).
func (s *NotificationService) SetHTTPClient(c *http.Client) {
	s.httpClient = c
}

// SetSMTPSender overrides the SMTP sender (used in tests).
func (s *NotificationService) SetSMTPSender(sender smtpSender) {
	s.smtpSender = sender
}

// CreateChannel validates and creates a notification channel.
func (s *NotificationService) CreateChannel(ctx context.Context, name, channelType string, enabled bool, configRaw json.RawMessage) (db.NotificationChannel, error) {
	if err := validateChannel(name, channelType, configRaw); err != nil {
		return db.NotificationChannel{}, err
	}
	return s.q.CreateNotificationChannel(ctx, db.CreateNotificationChannelParams{
		Name:        name,
		ChannelType: channelType,
		Enabled:     enabled,
		Config:      []byte(configRaw),
	})
}

// UpdateChannel validates and updates a channel. If channelType is smtp and the
// password is empty in the new config, the existing password is preserved.
func (s *NotificationService) UpdateChannel(ctx context.Context, id pgtype.UUID, name, channelType string, enabled bool, configRaw json.RawMessage) (db.NotificationChannel, error) {
	existing, err := s.q.GetNotificationChannel(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.NotificationChannel{}, ErrChannelNotFound
		}
		return db.NotificationChannel{}, err
	}

	mergedConfig := configRaw
	if channelType == ChannelTypeSMTP {
		merged, err := mergeSMTPPassword(existing.Config, configRaw)
		if err != nil {
			return db.NotificationChannel{}, err
		}
		mergedConfig = merged
	}

	if err := validateChannel(name, channelType, mergedConfig); err != nil {
		return db.NotificationChannel{}, err
	}

	return s.q.UpdateNotificationChannel(ctx, db.UpdateNotificationChannelParams{
		ID:          id,
		Name:        name,
		ChannelType: channelType,
		Enabled:     enabled,
		Config:      []byte(mergedConfig),
	})
}

func (s *NotificationService) DeleteChannel(ctx context.Context, id pgtype.UUID) error {
	return s.q.DeleteNotificationChannel(ctx, id)
}

func (s *NotificationService) ListChannels(ctx context.Context) ([]db.NotificationChannel, error) {
	items, err := s.q.ListNotificationChannels(ctx)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []db.NotificationChannel{}
	}
	return items, nil
}

func (s *NotificationService) GetChannel(ctx context.Context, id pgtype.UUID) (db.NotificationChannel, error) {
	c, err := s.q.GetNotificationChannel(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.NotificationChannel{}, ErrChannelNotFound
		}
		return db.NotificationChannel{}, err
	}
	return c, nil
}

// MaskConfig returns the channel config with secrets masked, suitable for API responses.
func MaskConfig(channel db.NotificationChannel) []byte {
	if channel.ChannelType != ChannelTypeSMTP {
		return MaskWebhookHeaders(channel.Config)
	}
	var cfg map[string]any
	if err := json.Unmarshal(channel.Config, &cfg); err != nil {
		return channel.Config
	}
	if _, ok := cfg["password"]; ok {
		cfg["password"] = maskedSecret
	}
	out, err := json.Marshal(cfg)
	if err != nil {
		return channel.Config
	}
	return out
}

// sensitiveHeaderKeys lists lowercase header name substrings that should be masked.
var sensitiveHeaderKeys = []string{"authorization", "token", "secret", "key"}

// MaskWebhookHeaders returns webhook config with sensitive header values masked.
func MaskWebhookHeaders(configBytes []byte) []byte {
	if len(configBytes) == 0 {
		return configBytes
	}
	var cfg map[string]any
	if err := json.Unmarshal(configBytes, &cfg); err != nil {
		return configBytes
	}
	headers, ok := cfg["headers"]
	if !ok {
		return configBytes
	}
	headerMap, ok := headers.(map[string]any)
	if !ok {
		return configBytes
	}
	changed := false
	for k, v := range headerMap {
		lk := strings.ToLower(k)
		for _, sensitive := range sensitiveHeaderKeys {
			if strings.Contains(lk, sensitive) {
				if _, isStr := v.(string); isStr {
					headerMap[k] = maskedSecret
					changed = true
				}
				break
			}
		}
	}
	if !changed {
		return configBytes
	}
	cfg["headers"] = headerMap
	out, err := json.Marshal(cfg)
	if err != nil {
		return configBytes
	}
	return out
}



// SendIncidentOpened delivers an incident.opened event to all enabled channels.
func (s *NotificationService) SendIncidentOpened(ctx context.Context, incident db.Incident, monitor db.Monitor) {
	payload := buildIncidentPayload(EventIncidentOpened, incident, monitor)
	s.dispatch(ctx, EventIncidentOpened, payload)
}

// SendIncidentResolved delivers an incident.resolved event to all enabled channels.
func (s *NotificationService) SendIncidentResolved(ctx context.Context, incident db.Incident, monitor db.Monitor) {
	payload := buildIncidentPayload(EventIncidentResolved, incident, monitor)
	s.dispatch(ctx, EventIncidentResolved, payload)
}

// TestChannel sends a test payload to the given channel and stores the delivery result.
func (s *NotificationService) TestChannel(ctx context.Context, id pgtype.UUID) (db.NotificationDelivery, error) {
	channel, err := s.GetChannel(ctx, id)
	if err != nil {
		return db.NotificationDelivery{}, err
	}

	payload := map[string]any{
		"event":   EventTest,
		"message": "Blackgrid test notification",
		"sent_at": time.Now().UTC().Format(time.RFC3339),
	}
	return s.deliver(ctx, channel, EventTest, payload), nil
}

func (s *NotificationService) dispatch(ctx context.Context, eventType string, payload map[string]any) {
	channels, err := s.q.ListEnabledNotificationChannels(ctx)
	if err != nil {
		log.Printf("notification dispatch: list channels failed: %v", err)
		return
	}
	for _, ch := range channels {
		// Each channel attempted independently — never let one fail block another.
		func(ch db.NotificationChannel) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("notification channel %s panic: %v", ch.Name, r)
				}
			}()
			s.deliver(ctx, ch, eventType, payload)
		}(ch)
	}
}

func (s *NotificationService) deliver(ctx context.Context, channel db.NotificationChannel, eventType string, payload map[string]any) db.NotificationDelivery {
	payloadBytes, _ := json.Marshal(payload)

	var sendErr error
	switch channel.ChannelType {
	case ChannelTypeWebhook:
		sendErr = s.sendWebhook(ctx, channel.Config, payloadBytes)
	case ChannelTypeSMTP:
		sendErr = s.sendSMTP(channel.Config, eventType, payload)
	default:
		sendErr = fmt.Errorf("unsupported channel type: %s", channel.ChannelType)
	}

	status := "sent"
	var lastErr pgtype.Text
	var sentAt pgtype.Timestamptz
	if sendErr != nil {
		status = "failed"
		lastErr = pgtype.Text{String: sendErr.Error(), Valid: true}
	} else {
		sentAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	}

	delivery, err := s.q.CreateNotificationDelivery(ctx, db.CreateNotificationDeliveryParams{
		ChannelID: channel.ID,
		EventType: eventType,
		Status:    status,
		Attempts:  1,
		LastError: lastErr,
		Payload:   payloadBytes,
		SentAt:    sentAt,
	})
	if err != nil {
		log.Printf("notification delivery store failed for channel %s: %v", channel.Name, err)
	}

	if s.bus != nil {
		eventType := events.NotificationSent
		if sendErr != nil {
			eventType = events.NotificationFailed
		}
		s.bus.Publish(ctx, events.Event{
			Type:       eventType,
			ObjectType: "notification_delivery",
			ObjectID:   events.FormatUUID(delivery.ID),
			Payload: map[string]any{
				"channel_id":   events.FormatUUID(channel.ID),
				"channel_name": channel.Name,
				"event":        eventType,
				"error":        lastErr.String,
			},
		})
	}

	return delivery
}

func (s *NotificationService) sendWebhook(ctx context.Context, configBytes, body []byte) error {
	var cfg WebhookConfig
	if err := json.Unmarshal(configBytes, &cfg); err != nil {
		return fmt.Errorf("invalid webhook config: %w", err)
	}
	if err := validateWebhookURL(cfg.URL); err != nil {
		return err
	}
	method := strings.ToUpper(cfg.Method)
	if method == "" {
		method = "POST"
	}

	reqCtx := ctx
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(ctx, time.Duration(cfg.Timeout)*time.Second)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(reqCtx, method, cfg.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook responded with status %d", resp.StatusCode)
	}
	return nil
}

func (s *NotificationService) sendSMTP(configBytes []byte, eventType string, payload map[string]any) error {
	var cfg SMTPConfig
	if err := json.Unmarshal(configBytes, &cfg); err != nil {
		return fmt.Errorf("invalid smtp config: %w", err)
	}
	if err := validateSMTP(cfg); err != nil {
		return err
	}

	subject := "Blackgrid Notification"
	if msg, ok := payload["message"].(string); ok && msg != "" {
		subject = msg
	}

	bodyJSON, _ := json.MarshalIndent(payload, "", "  ")
	body := fmt.Sprintf("Event: %s\n\n%s\n", eventType, string(bodyJSON))

	headers := map[string]string{
		"From":         cfg.From,
		"To":           strings.Join(cfg.To, ", "),
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/plain; charset=UTF-8",
	}

	var msg bytes.Buffer
	for k, v := range headers {
		fmt.Fprintf(&msg, "%s: %s\r\n", k, v)
	}
	msg.WriteString("\r\n")
	msg.WriteString(body)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}

	if cfg.UseTLS {
		// Implicit TLS connection — we use net/smtp with TLS-wrapped dialer.
		tlsCfg := &tls.Config{ServerName: cfg.Host}
		conn, err := tls.Dial("tcp", addr, tlsCfg)
		if err != nil {
			return fmt.Errorf("smtp tls dial: %w", err)
		}
		client, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			return fmt.Errorf("smtp client: %w", err)
		}
		defer client.Close()

		if auth != nil {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
		if err := client.Mail(cfg.From); err != nil {
			return err
		}
		for _, to := range cfg.To {
			if err := client.Rcpt(to); err != nil {
				return err
			}
		}
		w, err := client.Data()
		if err != nil {
			return err
		}
		if _, err := w.Write(msg.Bytes()); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		return client.Quit()
	}

	return s.smtpSender(addr, auth, cfg.From, cfg.To, msg.Bytes())
}

// --- helpers ---

func validateChannel(name, channelType string, configRaw json.RawMessage) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("name is required")
	}
	switch channelType {
	case ChannelTypeWebhook:
		var cfg WebhookConfig
		if len(configRaw) > 0 {
			if err := json.Unmarshal(configRaw, &cfg); err != nil {
				return fmt.Errorf("invalid webhook config: %w", err)
			}
		}
		return validateWebhookURL(cfg.URL)
	case ChannelTypeSMTP:
		var cfg SMTPConfig
		if len(configRaw) > 0 {
			if err := json.Unmarshal(configRaw, &cfg); err != nil {
				return fmt.Errorf("invalid smtp config: %w", err)
			}
		}
		return validateSMTP(cfg)
	default:
		return fmt.Errorf("unsupported channel_type: %s (must be webhook or smtp)", channelType)
	}
}

func validateWebhookURL(rawURL string) error {
	if rawURL == "" {
		return errors.New("webhook url is required")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("webhook url must be http or https")
	}
	if u.Host == "" {
		return errors.New("webhook url must include a host")
	}
	return nil
}

func validateSMTP(cfg SMTPConfig) error {
	if cfg.Host == "" {
		return errors.New("smtp host is required")
	}
	if cfg.Port <= 0 {
		return errors.New("smtp port is required")
	}
	if cfg.From == "" {
		return errors.New("smtp from is required")
	}
	if len(cfg.To) == 0 {
		return errors.New("at least one smtp recipient is required")
	}
	return nil
}

func mergeSMTPPassword(existingConfig []byte, newConfig json.RawMessage) (json.RawMessage, error) {
	var newCfg map[string]any
	if err := json.Unmarshal(newConfig, &newCfg); err != nil {
		return nil, fmt.Errorf("invalid smtp config: %w", err)
	}
	pw, _ := newCfg["password"].(string)
	if pw == "" || pw == maskedSecret {
		var oldCfg map[string]any
		if err := json.Unmarshal(existingConfig, &oldCfg); err == nil {
			if oldPw, ok := oldCfg["password"].(string); ok {
				newCfg["password"] = oldPw
			}
		}
	}
	return json.Marshal(newCfg)
}

func buildIncidentPayload(event string, incident db.Incident, monitor db.Monitor) map[string]any {
	statusForMessage := monitor.Status
	verb := "is " + statusForMessage
	if event == EventIncidentResolved {
		verb = "recovered"
	}
	message := fmt.Sprintf("Monitor %s %s", monitor.Name, verb)

	return map[string]any{
		"event":    event,
		"severity": incident.Severity,
		"incident": map[string]any{
			"id":              uuidToHex(incident.ID),
			"status":          incident.Status,
			"started_at":      timeOrNil(incident.StartedAt),
			"acknowledged_at": timeOrNil(incident.AcknowledgedAt),
			"resolved_at":     timeOrNil(incident.ResolvedAt),
		},
		"monitor": map[string]any{
			"id":     uuidToHex(monitor.ID),
			"name":   monitor.Name,
			"type":   monitor.MonitorType,
			"target": monitor.Target,
			"status": monitor.Status,
		},
		"message": message,
	}
}

func uuidToHex(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func timeOrNil(t pgtype.Timestamptz) any {
	if !t.Valid {
		return nil
	}
	return t.Time.UTC().Format(time.RFC3339)
}
