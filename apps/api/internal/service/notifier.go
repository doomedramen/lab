package service

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/doomedramen/lab/apps/api/internal/model"
)

// Notifier is the interface for sending alert notifications
type Notifier interface {
	Send(alert *model.Alert, channel *model.NotificationChannel) error
}

// EmailNotifier sends notifications via email
type EmailNotifier struct{}

// WebhookNotifier sends notifications via HTTP webhook
type WebhookNotifier struct {
	httpClient *http.Client
}

// NewEmailNotifier creates a new email notifier
func NewEmailNotifier() *EmailNotifier {
	return &EmailNotifier{}
}

// NewWebhookNotifier creates a new webhook notifier
func NewWebhookNotifier() *WebhookNotifier {
	return &WebhookNotifier{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// EmailPayload represents the email notification payload
type EmailPayload struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
	To      string `json:"to"`
}

// WebhookPayload represents the webhook notification payload
type WebhookPayload struct {
	ID           string            `json:"id"`
	RuleID       string            `json:"rule_id"`
	RuleName     string            `json:"rule_name"`
	EntityType   string            `json:"entity_type,omitempty"`
	EntityID     string            `json:"entity_id,omitempty"`
	EntityName   string            `json:"entity_name,omitempty"`
	Message      string            `json:"message"`
	Severity     string            `json:"severity"`
	Status       string            `json:"status"`
	FiredAt      string            `json:"fired_at"`
	Timestamp    string            `json:"timestamp"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// Send sends an email notification
func (n *EmailNotifier) Send(alert *model.Alert, channel *model.NotificationChannel) error {
	config := parseEmailConfig(channel.Config)

	if config.SMTPHost == "" || config.SMTPPort == 0 {
		return fmt.Errorf("email channel missing SMTP configuration")
	}

	if config.ToAddresses == "" {
		return fmt.Errorf("email channel missing recipient addresses")
	}

	subject := fmt.Sprintf("[Lab Alert] %s - %s", alert.Severity, alert.RuleName)
	body := n.formatBody(alert)

	// Parse from address
	from, err := mail.ParseAddress(config.FromAddress)
	if err != nil {
		from = &mail.Address{Address: config.FromAddress}
	}

	// Setup authentication
	var auth smtp.Auth
	if config.SMTPUser != "" {
		auth = smtp.PlainAuth("", config.SMTPUser, config.SMTPPassword, config.SMTPHost)
	}

	// Build message
	msg := bytes.NewBuffer(nil)
	msg.WriteString(fmt.Sprintf("From: %s\r\n", from.String()))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", config.ToAddresses))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	// Send email
	addr := fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort)

	// Use TLS for port 465, STARTTLS for port 587
	if config.SMTPPort == 465 {
		return n.sendWithTLS(addr, auth, from.Address, strings.Split(config.ToAddresses, ","), msg.Bytes(), config.SMTPHost)
	}

	return smtp.SendMail(addr, auth, from.Address, strings.Split(config.ToAddresses, ","), msg.Bytes())
}

func (n *EmailNotifier) sendWithTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte, host string) error {
	// TLS connection for port 465
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("failed to authenticate: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return fmt.Errorf("failed to set recipient: %w", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return client.Quit()
}

func (n *EmailNotifier) formatBody(alert *model.Alert) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("Alert: %s\n", alert.RuleName))
	buf.WriteString(fmt.Sprintf("Severity: %s\n", alert.Severity))
	buf.WriteString(fmt.Sprintf("Status: %s\n", alert.Status))
	buf.WriteString(fmt.Sprintf("Time: %s\n", alert.FiredAt.Format(time.RFC3339)))
	buf.WriteString("\n")
	buf.WriteString(fmt.Sprintf("Message:\n%s\n", alert.Message))

	if alert.EntityName != "" {
		buf.WriteString(fmt.Sprintf("\nEntity: %s (%s)\n", alert.EntityName, alert.EntityType))
	}

	if len(alert.Metadata) > 0 {
		buf.WriteString("\nAdditional Information:\n")
		for k, v := range alert.Metadata {
			buf.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}

	buf.WriteString("\n---\n")
	buf.WriteString("This is an automated alert from Lab.\n")

	return buf.String()
}

// Send sends a webhook notification
func (n *WebhookNotifier) Send(alert *model.Alert, channel *model.NotificationChannel) error {
	config := parseWebhookConfig(channel.Config)

	if config.URL == "" {
		return fmt.Errorf("webhook channel missing URL")
	}

	payload := &WebhookPayload{
		ID:         alert.ID,
		RuleID:     alert.RuleID,
		RuleName:   alert.RuleName,
		EntityType: alert.EntityType,
		EntityID:   alert.EntityID,
		EntityName: alert.EntityName,
		Message:    alert.Message,
		Severity:   string(alert.Severity),
		Status:     string(alert.Status),
		FiredAt:    alert.FiredAt.Format(time.RFC3339),
		Timestamp:  time.Now().Format(time.RFC3339),
		Metadata:   alert.Metadata,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	method := config.Method
	if method == "" {
		method = "POST"
	}

	req, err := http.NewRequest(method, config.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Lab-Alerts/1.0")

	// Add custom headers
	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	slog.Info("Webhook notification sent",
		"url", config.URL,
		"alert_id", alert.ID,
		"status", resp.StatusCode)

	return nil
}

// Helper functions

func parseEmailConfig(config map[string]string) model.EmailChannelConfig {
	cfg := model.EmailChannelConfig{
		SMTPPort: 587, // default to STARTTLS
	}

	if v, ok := config["smtp_host"]; ok {
		cfg.SMTPHost = v
	}
	if v, ok := config["smtp_port"]; ok {
		var port int
		fmt.Sscanf(v, "%d", &port)
		if port > 0 {
			cfg.SMTPPort = port
		}
	}
	if v, ok := config["smtp_user"]; ok {
		cfg.SMTPUser = v
	}
	if v, ok := config["smtp_pass"]; ok {
		cfg.SMTPPassword = v
	}
	if v, ok := config["from_address"]; ok {
		cfg.FromAddress = v
	}
	if v, ok := config["to_addresses"]; ok {
		cfg.ToAddresses = v
	}

	return cfg
}

func parseWebhookConfig(config map[string]string) model.WebhookChannelConfig {
	cfg := model.WebhookChannelConfig{
		Method:  "POST",
		Headers: make(map[string]string),
	}

	if v, ok := config["url"]; ok {
		cfg.URL = v
	}
	if v, ok := config["method"]; ok {
		cfg.Method = v
	}

	// Parse headers from JSON if present
	if headersJSON, ok := config["headers"]; ok && headersJSON != "" {
		json.Unmarshal([]byte(headersJSON), &cfg.Headers)
	}

	return cfg
}
