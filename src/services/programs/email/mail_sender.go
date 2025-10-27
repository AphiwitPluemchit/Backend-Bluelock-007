package email

import (
	// "crypto/tls"
	"fmt"
	"os"
	"strconv"
	"strings"

	gomail "gopkg.in/gomail.v2"
)

type MailSender interface {
	Send(to, subject, html string) error
}

type SMTPSender struct {
	Host string
	Port int
	User string
	Pass string
	From string
}

// mail_sender.go
func NewSMTPSenderFromEnv() (*SMTPSender, error) {
    host := os.Getenv("SMTP_HOST")
    port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
    user := os.Getenv("SMTP_USER")
    pass := os.Getenv("SMTP_PASS")
    from := os.Getenv("SMTP_FROM")

    missing := []string{}
    if host == "" { missing = append(missing, "SMTP_HOST") }
    if port == 0  { missing = append(missing, "SMTP_PORT") }
    if user == "" { missing = append(missing, "SMTP_USER") }
    if pass == "" { missing = append(missing, "SMTP_PASS") }
    if from == "" { missing = append(missing, "SMTP_FROM") }

    if len(missing) > 0 {
        return nil, fmt.Errorf("missing SMTP env: %v", strings.Join(missing, ", "))
    }
    return &SMTPSender{Host: host, Port: port, User: user, Pass: pass, From: from}, nil
}


func (s *SMTPSender) Send(to, subject, html string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", html)

	d := gomail.NewDialer(s.Host, s.Port, s.User, s.Pass)

	// d.TLSConfig = &tls.Config{
	// 	ServerName: s.Host,
	// 	MinVersion: tls.VersionTLS12,
	// }

	return d.DialAndSend(m)
}
