package utils

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/smtp"
	"os"
	"strings"
	"sync"
)

// EmailSender handles sending email notifications
type EmailSender struct {
	host     string
	port     string
	username string
	password string
	from     string
	logger   *Logger
}

// EmailConfig represents email configuration
type EmailConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	source   string // "database" or "environment" (internal tracking)
}

// NewEmailSender creates a new email sender
func NewEmailSender(config EmailConfig) *EmailSender {
	return &EmailSender{
		host:     config.Host,
		port:     config.Port,
		username: config.Username,
		password: config.Password,
		from:     config.From,
		logger:   GetLogger(),
	}
}

// Send sends an email to the specified recipients
func (s *EmailSender) Send(to []string, subject string, body string) error {
	if s.host == "" {
		return fmt.Errorf("SMTP host not configured")
	}

	// Build email message
	msg := s.buildMessage(to, subject, body)

	// Setup authentication
	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%s", s.host, s.port)

	// Try TLS first, fall back to plain SMTP
	err := s.sendWithTLS(addr, auth, s.from, to, []byte(msg))
	if err != nil {
		// Fall back to plain SMTP
		s.logger.Warn("TLS connection failed, falling back to plain SMTP", Error(err))
		err = smtp.SendMail(addr, auth, s.from, to, []byte(msg))
	}

	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info("Email sent successfully",
		Int("recipients", len(to)),
		String("subject", subject))

	return nil
}

// sendWithTLS sends email using TLS
func (s *EmailSender) sendWithTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	// TLS config
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         s.host,
	}

	// Connect to server
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return err
	}
	defer client.Close()

	// Authenticate
	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return err
		}
	}

	// Set sender
	if err = client.Mail(from); err != nil {
		return err
	}

	// Set recipients
	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			return err
		}
	}

	// Send message
	w, err := client.Data()
	if err != nil {
		return err
	}

	_, err = w.Write(msg)
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	return client.Quit()
}

// buildMessage builds an RFC 822-style email message
func (s *EmailSender) buildMessage(to []string, subject string, body string) string {
	headers := make(map[string]string)
	headers["From"] = s.from
	headers["To"] = strings.Join(to, ", ")
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/plain; charset=\"utf-8\""

	var msg strings.Builder
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(body)

	return msg.String()
}

// Global email sender instance
var (
	globalEmailSender     *EmailSender
	globalEmailSenderOnce sync.Once
	globalEmailDB         *sql.DB
)

// SetEmailSenderDB sets the database connection for email sender config
func SetEmailSenderDB(db *sql.DB) {
	globalEmailDB = db
}

// GetEmailSender returns the global email sender instance
func GetEmailSender() *EmailSender {
	globalEmailSenderOnce.Do(func() {
		config := loadEmailConfig()

		// Only create sender if SMTP is configured
		if config.Host != "" {
			globalEmailSender = NewEmailSender(config)
			GetLogger().Info("Email sender initialized",
				String("host", config.Host),
				String("port", config.Port),
				String("from", config.From),
				String("source", config.source))
		} else {
			GetLogger().Warn("Email sender not configured (set SMTP_HOST env var or configure via Settings > Notifications)")
		}
	})

	return globalEmailSender
}

// loadEmailConfig loads config from database first, then falls back to env vars
func loadEmailConfig() EmailConfig {
	config := EmailConfig{}

	// Try loading from database (plugin_config table)
	if globalEmailDB != nil {
		dbConfig := loadEmailConfigFromDB()
		if dbConfig != nil {
			return *dbConfig
		}
	}

	// Fall back to environment variables
	config.Host = os.Getenv("SMTP_HOST")
	config.Port = os.Getenv("SMTP_PORT")
	config.Username = os.Getenv("SMTP_USERNAME")
	config.Password = os.Getenv("SMTP_PASSWORD")
	config.From = os.Getenv("SMTP_FROM")
	config.source = "environment"

	// Set defaults
	if config.Port == "" {
		config.Port = "587"
	}
	if config.From == "" && config.Username != "" {
		config.From = config.Username
	}

	return config
}

// loadEmailConfigFromDB loads SMTP config from the plugin_config table
func loadEmailConfigFromDB() *EmailConfig {
	if globalEmailDB == nil {
		return nil
	}

	var configJSON string
	err := globalEmailDB.QueryRow(`
		SELECT config FROM plugin_config WHERE plugin_name = 'smtp' OR plugin_name = 'email'
	`).Scan(&configJSON)

	if err != nil {
		return nil
	}

	var dbConfig struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
		From     string `json:"from"`
		UseTLS   bool   `json:"use_tls"`
	}

	if err := json.Unmarshal([]byte(configJSON), &dbConfig); err != nil {
		GetLogger().Warn("Failed to parse SMTP config from database", Error(err))
		return nil
	}

	if dbConfig.Host == "" {
		return nil
	}

	config := &EmailConfig{
		Host:     dbConfig.Host,
		Port:     dbConfig.Port,
		Username: dbConfig.Username,
		Password: dbConfig.Password,
		From:     dbConfig.From,
		source:   "database",
	}

	// Set defaults
	if config.Port == "" {
		config.Port = "587"
	}
	if config.From == "" && config.Username != "" {
		config.From = config.Username
	}

	return config
}

// RefreshEmailSender reloads the email sender with updated config
func RefreshEmailSender() {
	globalEmailSender = nil
	globalEmailSenderOnce = sync.Once{}
	GetEmailSender() // Reinitialize
}

// ResetGlobalEmailSender resets the global email sender (for testing)
func ResetGlobalEmailSender() {
	globalEmailSender = nil
	globalEmailSenderOnce = sync.Once{}
}
