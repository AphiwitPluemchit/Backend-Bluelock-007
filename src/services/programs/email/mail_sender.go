package email

import (
	// "crypto/tls"
	gomail "gopkg.in/gomail.v2"
	"os"
	"strconv"
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

func NewSMTPSenderFromEnv() (*SMTPSender, error) {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	return &SMTPSender{
		Host: os.Getenv("SMTP_HOST"),
		Port: port,
		User: os.Getenv("SMTP_USER"),
		Pass: os.Getenv("SMTP_PASS"),
		From: os.Getenv("SMTP_FROM"),
	}, nil
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
