package email

import (
	"context"
	"fmt"
	"net/smtp"
)

type SMTPSender struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewSMTPSender(host, port, username, password string) *SMTPSender {
	from := username
	if from == "" {
		from = "noreply@social.local"
	}
	return &SMTPSender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}


func (s *SMTPSender) Send(_ context.Context, to, subject, body string) error {
	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n%s",
		s.from, to, subject, body,
	)
	return smtp.SendMail(s.host+":"+s.port, auth, s.from, []string{to}, []byte(msg))
}
