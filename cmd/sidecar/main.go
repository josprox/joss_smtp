package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"
)

type request struct {
	Protocol string        `json:"protocol"`
	ID       string        `json:"id"`
	Method   string        `json:"method"`
	Args     []interface{} `json:"args"`
}

type response struct {
	ID     string      `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  interface{} `json:"error,omitempty"`
}

type sendConfig struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	User    string `json:"user"`
	Pass    string `json:"pass"`
	Secure  bool   `json:"secure"`
	Timeout int    `json:"timeout"`
}

func main() {
	var req request
	if err := json.NewDecoder(io.LimitReader(os.Stdin, 4<<20)).Decode(&req); err != nil {
		write(response{Error: map[string]string{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	if req.Protocol != "joss-rpc-v1" || req.Method != "send" || len(req.Args) != 1 {
		write(response{ID: req.ID, Error: map[string]string{"code": "BAD_REQUEST", "message": "se requiere send(config) sobre joss-rpc-v1"}})
		return
	}
	data, _ := json.Marshal(req.Args[0])
	var cfg sendConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		write(response{ID: req.ID, Error: map[string]string{"code": "BAD_CONFIG", "message": err.Error()}})
		return
	}
	if err := send(cfg); err != nil {
		write(response{ID: req.ID, Result: map[string]interface{}{"ok": false, "error": err.Error()}})
		return
	}
	write(response{ID: req.ID, Result: map[string]interface{}{"ok": true, "error": ""}})
}

func send(cfg sendConfig) error {
	if strings.TrimSpace(cfg.To) == "" {
		return fmt.Errorf("destinatario vacío")
	}
	if cfg.User == "" {
		cfg.User = os.Getenv("MAIL_USERNAME")
	}
	if cfg.Pass == "" {
		cfg.Pass = os.Getenv("MAIL_PASSWORD")
	}
	if apiKey := strings.TrimSpace(os.Getenv("BREVO_API")); apiKey != "" {
		return sendBrevo(apiKey, cfg)
	}
	host := envDefault("MAIL_HOST", "smtp.gmail.com")
	port := envDefault("MAIL_PORT", "587")
	if cfg.Timeout <= 0 || cfg.Timeout > 3600 {
		cfg.Timeout = 30
	}
	timeout := time.Duration(cfg.Timeout) * time.Second
	dialer := &net.Dialer{Timeout: timeout}
	var conn net.Conn
	var err error
	if port == "465" {
		conn, err = tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(host, port), &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
	} else {
		conn, err = dialer.Dial("tcp", net.JoinHostPort(host, port))
	}
	if err != nil {
		return fmt.Errorf("conexión SMTP: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()
	if supported, _ := client.Extension("STARTTLS"); supported && (cfg.Secure || port == "587") {
		if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("STARTTLS: %w", err)
		}
	}
	if cfg.User != "" && cfg.Pass != "" {
		if err := client.Auth(smtp.PlainAuth("", cfg.User, cfg.Pass, host)); err != nil {
			return fmt.Errorf("autenticación SMTP: %w", err)
		}
	}
	if err := client.Mail(cfg.User); err != nil {
		return err
	}
	if err := client.Rcpt(cfg.To); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	defer writer.Close()
	from := cfg.User
	if name := strings.TrimSpace(os.Getenv("MAIL_FROM_NAME")); name != "" {
		from = fmt.Sprintf("%q <%s>", cleanHeader(name), cfg.User)
	}
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s\r\n",
		cleanHeader(from), cleanHeader(cfg.To), cleanHeader(cfg.Subject), time.Now().Format(time.RFC1123Z), cfg.Body)
	_, err = io.WriteString(writer, message)
	return err
}

func sendBrevo(apiKey string, cfg sendConfig) error {
	sender := map[string]string{"email": cfg.User}
	if name := strings.TrimSpace(os.Getenv("MAIL_FROM_NAME")); name != "" {
		sender["name"] = name
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"sender": sender, "to": []map[string]string{{"email": cfg.To}},
		"subject": cfg.Subject, "htmlContent": cfg.Body,
	})
	req, err := http.NewRequest(http.MethodPost, "https://api.brevo.com/v3/smtp/email", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("api-key", apiKey)
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30
	}
	resp, err := (&http.Client{Timeout: time.Duration(timeout) * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("Brevo HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func cleanHeader(value string) string {
	return strings.TrimSpace(strings.NewReplacer("\r", "", "\n", "").Replace(value))
}

func envDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func write(value response) { _ = json.NewEncoder(os.Stdout).Encode(value) }
