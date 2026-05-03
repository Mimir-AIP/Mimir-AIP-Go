package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"io"
	"net/http"
	"net/smtp"
	"os"
	"strconv"
)

// ── Notifications ─────────────────────────────────────────────────────────────

// sendEmail sends an email via SMTP.
//
// Parameters:
//   - to          (string, required): Recipient address.
//   - subject     (string, optional): Email subject.
//   - body        (string, optional): Plain-text body.
//   - from        (string, optional): Sender address; defaults to username.
//   - smtp_host   (string, optional): SMTP server hostname. Fallback: $SMTP_HOST.
//   - smtp_port   (string, optional): SMTP port. Default: "587".
//   - username    (string, optional): SMTP username. Fallback: $SMTP_USERNAME.
//   - password    (string, optional): SMTP password. Fallback: $SMTP_PASSWORD.
func (p *DefaultPlugin) sendEmail(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	resolve := func(key, fallbackEnv string) string {
		v, _ := params[key].(string)
		v = p.ResolveTemplates(v, ctx)
		if v == "" && fallbackEnv != "" {
			v = os.Getenv(fallbackEnv)
		}
		return v
	}

	to := resolve("to", "")
	if to == "" {
		return nil, fmt.Errorf("send_email: to parameter is required")
	}

	subject := resolve("subject", "")
	body := resolve("body", "")
	smtpHost := resolve("smtp_host", "SMTP_HOST")
	username := resolve("username", "SMTP_USERNAME")
	password := resolve("password", "SMTP_PASSWORD")
	from := resolve("from", "")
	if from == "" {
		from = username
	}

	portStr := resolve("smtp_port", "SMTP_PORT")
	smtpPort := 587
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			smtpPort = p
		}
	}

	if smtpHost == "" {
		return nil, fmt.Errorf("send_email: smtp_host is required (param or $SMTP_HOST)")
	}

	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)
	auth := smtp.PlainAuth("", username, password, smtpHost)
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body,
	)

	if err := smtp.SendMail(addr, auth, from, []string{to}, []byte(msg)); err != nil {
		return nil, fmt.Errorf("send_email: %w", err)
	}

	return map[string]interface{}{
		"sent": true,
		"to":   to,
	}, nil
}

// sendWebhook posts a JSON payload to an HTTP endpoint.
// This is useful for Slack, Teams, PagerDuty, and any other webhook-based system.
//
// Parameters:
//   - url       (string, required): Webhook endpoint URL.
//   - payload   (map|string, optional): JSON body. Supports templates. Default: {}.
//   - headers   (map, optional): Additional HTTP headers.
func (p *DefaultPlugin) sendWebhook(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("send_webhook: url parameter is required")
	}
	url = p.ResolveTemplates(url, ctx)

	var payloadBytes []byte
	rawPayload, hasPayload := params["payload"]
	if hasPayload {
		switch v := rawPayload.(type) {
		case string:
			resolved := p.ResolveTemplates(v, ctx)
			payloadBytes = []byte(resolved)
		default:
			var err error
			payloadBytes, err = json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("send_webhook: failed to marshal payload: %w", err)
			}
		}
	} else {
		payloadBytes = []byte("{}")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("send_webhook: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if headers, ok := params["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			if sv, ok := v.(string); ok {
				req.Header.Set(k, p.ResolveTemplates(sv, ctx))
			}
		}
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send_webhook: request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	return map[string]interface{}{
		"sent":        true,
		"status_code": resp.StatusCode,
		"response":    string(respBody),
	}, nil
}
