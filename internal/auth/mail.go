package auth

import (
	"fmt"
	"net/smtp"
)

type MailService struct {
	smtpHost string
	smtpPort string
	username string
	password string
}

func NewMailService(
	host string,
	port string,
	username string,
	password string,
) *MailService {

	return &MailService{
		smtpHost: host,
		smtpPort: port,
		username: username,
		password: password,
	}
}

func (m *MailService) SendVerificationEmail(email string, otp string) error {

	auth := smtp.PlainAuth(
		"",
		m.username,
		m.password,
		m.smtpHost,
	)

	subject := "Subject: Verify Your Email\r\n"

	body := fmt.Sprintf(
		"Your verification code is: %s\r\nThis code expires in 10 minutes.",
		otp,
	)

	message :=
		[]byte(subject + "\r\n" + body)

	return smtp.SendMail(
		m.smtpHost+":"+m.smtpPort,
		auth,
		m.username,
		[]string{email},
		message,
	)
}
