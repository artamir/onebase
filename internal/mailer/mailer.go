package mailer

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

// Config holds SMTP settings from config/app.yaml section "email".
type Config struct {
	SMTPHost    string `yaml:"smtp_host"`
	SMTPPort    int    `yaml:"smtp_port"`
	SMTPUser    string `yaml:"smtp_user"`
	SMTPPass    string `yaml:"smtp_password"` // or "env:VAR_NAME"
	FromName    string `yaml:"from_name"`
	FromAddress string `yaml:"from_address"`
}

type Mailer struct {
	cfg Config
}

func New(cfg Config) *Mailer {
	return &Mailer{cfg: cfg}
}

func (m *Mailer) Configured() bool {
	return m != nil && m.cfg.SMTPHost != ""
}

// Send delivers an email. Pass empty htmlBody for plain-text only.
func (m *Mailer) Send(to, subject, textBody, htmlBody string) error {
	if !m.Configured() {
		return fmt.Errorf("email не настроен — добавьте секцию email в config/app.yaml")
	}
	port := m.cfg.SMTPPort
	if port == 0 {
		port = 587
	}
	addr := fmt.Sprintf("%s:%d", m.cfg.SMTPHost, port)

	from := m.cfg.FromAddress
	if m.cfg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", m.cfg.FromName, m.cfg.FromAddress)
	}

	msg := buildMsg(from, to, subject, textBody, htmlBody)

	var auth smtp.Auth
	if m.cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", m.cfg.SMTPUser, m.password(), m.cfg.SMTPHost)
	}

	if port == 465 {
		return sendTLS(addr, m.cfg.SMTPHost, auth, m.cfg.FromAddress, to, msg)
	}
	return smtp.SendMail(addr, auth, m.cfg.FromAddress, []string{to}, msg)
}

func (m *Mailer) password() string {
	if strings.HasPrefix(m.cfg.SMTPPass, "env:") {
		return os.Getenv(strings.TrimPrefix(m.cfg.SMTPPass, "env:"))
	}
	return m.cfg.SMTPPass
}

func sendTLS(addr, host string, auth smtp.Auth, from, to string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer c.Quit() //nolint:errcheck
	if auth != nil {
		if err = c.Auth(auth); err != nil {
			return err
		}
	}
	if err = c.Mail(from); err != nil {
		return err
	}
	if err = c.Rcpt(to); err != nil {
		return err
	}
	wc, err := c.Data()
	if err != nil {
		return err
	}
	defer wc.Close() //nolint:errcheck
	_, err = wc.Write(msg)
	return err
}

func buildMsg(from, to, subject, textBody, htmlBody string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")

	if htmlBody != "" {
		const boundary = "==boundary_onebase_email"
		b.WriteString("Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n\r\n")
		if textBody != "" {
			b.WriteString("--" + boundary + "\r\n")
			b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
			b.WriteString(textBody + "\r\n")
		}
		b.WriteString("--" + boundary + "\r\n")
		b.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		b.WriteString(htmlBody + "\r\n")
		b.WriteString("--" + boundary + "--\r\n")
	} else {
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		b.WriteString(textBody + "\r\n")
	}
	return []byte(b.String())
}
