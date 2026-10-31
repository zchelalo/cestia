package mailer

import (
	"bytes"
	"context"
	"crypto/tls"
	"embed"
	"fmt"
	htmltemplate "html/template"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	texttemplate "text/template"
	"time"
)

//go:embed templates/*.txt templates/*.html
var templateFS embed.FS

type SMTP struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	TLS      bool
	StartTLS bool
	Timeout  time.Duration
}

type Mailer struct {
	cfg SMTP
}

type Message struct {
	To      string
	Subject string
	Data    any
	Kind    string
}

func New(cfg SMTP) *Mailer {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &Mailer{cfg: cfg}
}

func (m *Mailer) Send(ctx context.Context, message Message) error {
	htmlBody, textBody, err := render(message.Kind, message.Data)
	if err != nil {
		return err
	}
	from, err := mail.ParseAddress(m.cfg.From)
	if err != nil {
		return fmt.Errorf("parse sender: %w", err)
	}
	to, err := mail.ParseAddress(message.To)
	if err != nil {
		return fmt.Errorf("parse recipient: %w", err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	headers := []string{
		"From: " + from.String(),
		"To: " + to.String(),
		"Subject: " + mime.QEncoding.Encode("UTF-8", message.Subject),
		"MIME-Version: 1.0",
		"Content-Type: multipart/alternative; boundary=\"" + writer.Boundary() + "\"",
		"Date: " + time.Now().UTC().Format(time.RFC1123Z),
	}
	body.WriteString(strings.Join(headers, "\r\n") + "\r\n\r\n")
	for _, part := range []struct {
		contentType string
		content     string
	}{
		{"text/plain; charset=UTF-8", textBody},
		{"text/html; charset=UTF-8", htmlBody},
	} {
		header := textproto.MIMEHeader{}
		header.Set("Content-Type", part.contentType)
		partWriter, err := writer.CreatePart(header)
		if err != nil {
			return err
		}
		if _, err := partWriter.Write([]byte(part.content)); err != nil {
			return err
		}
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return m.send(ctx, from.Address, to.Address, body.Bytes())
}

func (m *Mailer) send(ctx context.Context, from, to string, body []byte) error {
	addr := net.JoinHostPort(m.cfg.Host, strconv.Itoa(m.cfg.Port))
	dialer := &net.Dialer{Timeout: m.cfg.Timeout}
	tlsConfig := &tls.Config{ServerName: m.cfg.Host, MinVersion: tls.VersionTLS12}
	var conn net.Conn
	var err error
	if m.cfg.TLS {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("dial smtp: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(m.cfg.Timeout))
	client, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		return fmt.Errorf("create smtp client: %w", err)
	}
	defer client.Close()
	if !m.cfg.TLS && m.cfg.StartTLS {
		ok, _ := client.Extension("STARTTLS")
		if !ok {
			return fmt.Errorf("smtp server does not support STARTTLS")
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("start tls: %w", err)
		}
	}
	if m.cfg.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := writer.Write(body); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write smtp body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close smtp body: %w", err)
	}
	return client.Quit()
}

func render(kind string, data any) (string, string, error) {
	htmlName := "templates/" + kind + ".html"
	textName := "templates/" + kind + ".txt"
	htmlRaw, err := templateFS.ReadFile(htmlName)
	if err != nil {
		return "", "", err
	}
	textRaw, err := templateFS.ReadFile(textName)
	if err != nil {
		return "", "", err
	}
	htmlTpl, err := htmltemplate.New(kind).Parse(string(htmlRaw))
	if err != nil {
		return "", "", err
	}
	textTpl, err := texttemplate.New(kind).Parse(string(textRaw))
	if err != nil {
		return "", "", err
	}
	var htmlBody, textBody bytes.Buffer
	if err := htmlTpl.Execute(&htmlBody, data); err != nil {
		return "", "", err
	}
	if err := textTpl.Execute(&textBody, data); err != nil {
		return "", "", err
	}
	return htmlBody.String(), textBody.String(), nil
}
